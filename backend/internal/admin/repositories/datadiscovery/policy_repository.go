package datadiscovery

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

type PolicyListParams struct {
	Search           string
	SourceTypes      []string
	ConfigurationIDs []int
	Statuses         []string
	Page             int
	PageSize         int
}

type PolicyReference struct {
	PolicyID int    `gorm:"column:policy_id"`
	ID       int    `gorm:"column:id"`
	Name     string `gorm:"column:name"`
}

type PolicyRepository struct {
	db *gorm.DB
}

func NewPolicyRepository(database *gorm.DB) *PolicyRepository {
	return &PolicyRepository{db: database}
}

func (r *PolicyRepository) WithTx(tx *gorm.DB) *PolicyRepository {
	return &PolicyRepository{db: tx}
}

func (r *PolicyRepository) List(
	ctx context.Context,
	customerID int,
	params PolicyListParams,
) ([]db.DataDiscoveryPolicy, int64, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	query := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryPolicy{}).
		Scopes(db.TenantScope(customerID))

	if search := utils.NormalizeName(params.Search); search != "" {
		query = query.Where(`(name ILIKE ? ESCAPE '\')`, utils.LikePattern(search))
	}

	if len(params.SourceTypes) > 0 {
		query = query.Where("source_type IN ?", params.SourceTypes)
	}

	if len(params.ConfigurationIDs) > 0 {
		query = query.Where("configuration_id IN ?", params.ConfigurationIDs)
	}

	if len(params.Statuses) > 0 {
		query = query.Where("status IN ?", params.Statuses)
	}

	query = query.Session(&gorm.Session{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]db.DataDiscoveryPolicy, 0, pageSize)

	if total > int64(utils.Offset(page, pageSize)) {
		err := query.
			Order("created_at DESC, id DESC").
			Limit(pageSize).
			Offset(utils.Offset(page, pageSize)).
			Find(&rows).Error
		if err != nil {
			return nil, 0, err
		}
	}

	return rows, total, nil
}

func (r *PolicyRepository) Find(ctx context.Context, customerID, id int) (db.DataDiscoveryPolicy, error) {
	var row db.DataDiscoveryPolicy

	err := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryPolicy{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *PolicyRepository) Lock(ctx context.Context, customerID, id int) (db.DataDiscoveryPolicy, error) {
	var row db.DataDiscoveryPolicy

	err := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryPolicy{}).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *PolicyRepository) Insert(ctx context.Context, row *db.DataDiscoveryPolicy) error {
	return r.db.WithContext(ctx).Omit("CreatedAt", "UpdatedAt").Create(row).Error
}

func (r *PolicyRepository) Update(
	ctx context.Context,
	customerID, id int,
	updates map[string]any,
) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryPolicy{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Updates(updates)

	return result.RowsAffected, result.Error
}

func (r *PolicyRepository) Delete(ctx context.Context, customerID, id int) (int64, error) {
	result := r.db.WithContext(ctx).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Delete(&db.DataDiscoveryPolicy{})

	return result.RowsAffected, result.Error
}

func (r *PolicyRepository) LockOwnedConfiguration(
	ctx context.Context,
	customerID, configurationID int,
) (db.DataDiscoveryConfiguration, error) {
	var row db.DataDiscoveryConfiguration

	err := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryConfiguration{}).
		Select("id", "customer_id", "configuration_type", "status").
		Clauses(clause.Locking{Strength: "KEY SHARE"}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", configurationID).
		Take(&row).Error

	return row, err
}

func (r *PolicyRepository) LockOwnedRuleIDs(
	ctx context.Context,
	customerID int,
	ids []int,
) ([]int, error) {
	found := make([]int, 0, len(ids))

	err := r.db.WithContext(ctx).
		Model(&db.Rule{}).
		Clauses(clause.Locking{Strength: "KEY SHARE"}).
		Scopes(db.TenantScope(customerID)).
		Where("id IN ?", ids).
		Pluck("id", &found).Error

	return found, err
}

func (r *PolicyRepository) KnownExtensions(ctx context.Context, extensions []string) ([]string, error) {
	found := make([]string, 0, len(extensions))

	if len(extensions) == 0 {
		return found, nil
	}

	err := r.db.WithContext(ctx).
		Model(&db.FileType{}).
		Where("extension IN ? AND active = ?", extensions, true).
		Pluck("extension", &found).Error

	return found, err
}

func (r *PolicyRepository) ReplaceRules(
	ctx context.Context,
	customerID, policyID int,
	ids []int,
) error {
	err := r.db.WithContext(ctx).
		Where("policy_id = ? AND customer_id = ?", policyID, customerID).
		Delete(&db.DataDiscoveryPolicyRuleMapping{}).Error
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	rows := make([]db.DataDiscoveryPolicyRuleMapping, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, db.DataDiscoveryPolicyRuleMapping{
			PolicyID: policyID, RuleID: id, CustomerID: customerID,
		})
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&rows).Error
}

func (r *PolicyRepository) RulesFor(
	ctx context.Context,
	customerID int,
	policyIDs []int,
) ([]PolicyReference, error) {
	rows := make([]PolicyReference, 0, len(policyIDs))

	if len(policyIDs) == 0 {
		return rows, nil
	}

	err := r.db.WithContext(ctx).
		Table("data_discovery_policy_rule_mapping AS m").
		Select("m.policy_id AS policy_id, r.id AS id, r.rule_name AS name").
		Joins("JOIN rules AS r ON r.id = m.rule_id AND r.customer_id = m.customer_id").
		Scopes(db.TenantScopeOn("m", customerID)).
		Where("m.policy_id IN ?", policyIDs).
		Order("r.rule_name ASC, r.id ASC").
		Scan(&rows).Error

	return rows, err
}

func (r *PolicyRepository) ConfigurationsFor(
	ctx context.Context,
	customerID int,
	configurationIDs []int,
) ([]utils.ReferenceItem, error) {
	rows := make([]utils.ReferenceItem, 0, len(configurationIDs))

	if len(configurationIDs) == 0 {
		return rows, nil
	}

	err := r.db.WithContext(ctx).
		Table("data_discovery_configurations AS c").
		Select("c.id AS id, c.name AS name").
		Scopes(db.TenantScopeOn("c", customerID)).
		Where("c.id IN ?", configurationIDs).
		Order("c.name ASC, c.id ASC").
		Scan(&rows).Error

	return rows, err
}
