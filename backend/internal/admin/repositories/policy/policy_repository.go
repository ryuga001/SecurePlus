package policy

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

type ListParams struct {
	Search   string
	Active   *bool
	Page     int
	PageSize int
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

func (r *PolicyRepository) List(ctx context.Context, customerID int, params ListParams) ([]db.Policy, int64, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	query := r.db.WithContext(ctx).
		Model(&db.Policy{}).
		Scopes(db.TenantScope(customerID))

	if search := utils.NormalizeName(params.Search); search != "" {
		query = query.Where(`(policy_name ILIKE ? ESCAPE '\')`, utils.LikePattern(search))
	}

	if params.Active != nil {
		query = query.Where("active = ?", *params.Active)
	}

	query = query.Session(&gorm.Session{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]db.Policy, 0, pageSize)

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

func (r *PolicyRepository) Find(ctx context.Context, customerID, id int) (db.Policy, error) {
	var row db.Policy

	err := r.db.WithContext(ctx).
		Model(&db.Policy{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *PolicyRepository) Lock(ctx context.Context, customerID, id int) (db.Policy, error) {
	var row db.Policy

	err := r.db.WithContext(ctx).
		Model(&db.Policy{}).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *PolicyRepository) Insert(ctx context.Context, row *db.Policy) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *PolicyRepository) Update(ctx context.Context, customerID, id int, updates map[string]any) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&db.Policy{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Updates(updates)

	return result.RowsAffected, result.Error
}

func (r *PolicyRepository) Delete(ctx context.Context, customerID, id int) (int64, error) {
	result := r.db.WithContext(ctx).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Delete(&db.Policy{})

	return result.RowsAffected, result.Error
}

func (r *PolicyRepository) LockOwnedGroupIDs(ctx context.Context, customerID int, ids []int) ([]int, error) {
	found := make([]int, 0, len(ids))

	err := r.db.WithContext(ctx).
		Model(&db.Group{}).
		Clauses(clause.Locking{Strength: "KEY SHARE"}).
		Scopes(db.TenantScope(customerID)).
		Where("id IN ? AND type = ?", ids, db.GroupTypeUser).
		Pluck("id", &found).Error

	return found, err
}

func (r *PolicyRepository) LockOwnedRuleIDs(ctx context.Context, customerID int, ids []int) ([]int, error) {
	found := make([]int, 0, len(ids))

	err := r.db.WithContext(ctx).
		Model(&db.Rule{}).
		Clauses(clause.Locking{Strength: "KEY SHARE"}).
		Scopes(db.TenantScope(customerID)).
		Where("id IN ?", ids).
		Pluck("id", &found).Error

	return found, err
}

func (r *PolicyRepository) ReplaceGroups(ctx context.Context, customerID, policyID int, ids []int) error {
	err := r.db.WithContext(ctx).
		Where("policy_id = ? AND customer_id = ?", policyID, customerID).
		Delete(&db.PolicyGroupMapping{}).Error
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	rows := make([]db.PolicyGroupMapping, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, db.PolicyGroupMapping{PolicyID: policyID, GroupID: id, CustomerID: customerID})
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&rows).Error
}

func (r *PolicyRepository) ReplaceRules(ctx context.Context, customerID, policyID int, ids []int) error {
	err := r.db.WithContext(ctx).
		Where("policy_id = ? AND customer_id = ?", policyID, customerID).
		Delete(&db.PolicyRuleMapping{}).Error
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	rows := make([]db.PolicyRuleMapping, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, db.PolicyRuleMapping{PolicyID: policyID, RuleID: id, CustomerID: customerID})
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&rows).Error
}

func (r *PolicyRepository) GroupsFor(ctx context.Context, customerID int, policyIDs []int) ([]PolicyReference, error) {
	rows := make([]PolicyReference, 0, len(policyIDs))

	if len(policyIDs) == 0 {
		return rows, nil
	}

	err := r.db.WithContext(ctx).
		Table("policy_group_mapping AS m").
		Select("m.policy_id AS policy_id, g.id AS id, g.name AS name").
		Joins("JOIN groups AS g ON g.id = m.group_id AND g.customer_id = m.customer_id").
		Scopes(db.TenantScopeOn("m", customerID)).
		Where("m.policy_id IN ?", policyIDs).
		Order("g.name ASC, g.id ASC").
		Scan(&rows).Error

	return rows, err
}

func (r *PolicyRepository) RulesFor(ctx context.Context, customerID int, policyIDs []int) ([]PolicyReference, error) {
	rows := make([]PolicyReference, 0, len(policyIDs))

	if len(policyIDs) == 0 {
		return rows, nil
	}

	err := r.db.WithContext(ctx).
		Table("policy_rule_mapping AS m").
		Select("m.policy_id AS policy_id, r.id AS id, r.rule_name AS name").
		Joins("JOIN rules AS r ON r.id = m.rule_id AND r.customer_id = m.customer_id").
		Scopes(db.TenantScopeOn("m", customerID)).
		Where("m.policy_id IN ?", policyIDs).
		Order("r.rule_name ASC, r.id ASC").
		Scan(&rows).Error

	return rows, err
}
