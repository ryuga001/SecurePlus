package datadiscovery

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

var configurationColumns = []string{
	"id", "customer_id", "name", "description", "configuration_type",
	"config", "status", "last_tested_at", "created_at", "updated_at",
}

var credentialMetaColumns = []string{
	"id", "customer_id", "configuration_id", "key_version", "created_at", "updated_at",
}

type ConfigurationListParams struct {
	Search             string
	ConfigurationTypes []string
	Statuses           []string
	Page               int
	PageSize           int
}

type ConfigurationPolicyCount struct {
	ConfigurationID int   `gorm:"column:configuration_id"`
	Total           int64 `gorm:"column:total"`
}

type ConfigurationRepository struct {
	db *gorm.DB
}

func NewConfigurationRepository(database *gorm.DB) *ConfigurationRepository {
	return &ConfigurationRepository{db: database}
}

func (r *ConfigurationRepository) WithTx(tx *gorm.DB) *ConfigurationRepository {
	return &ConfigurationRepository{db: tx}
}

func (r *ConfigurationRepository) List(
	ctx context.Context,
	customerID int,
	params ConfigurationListParams,
) ([]db.DataDiscoveryConfiguration, int64, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	query := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryConfiguration{}).
		Scopes(db.TenantScope(customerID))

	if search := utils.NormalizeName(params.Search); search != "" {
		query = query.Where(`(name ILIKE ? ESCAPE '\')`, utils.LikePattern(search))
	}

	if len(params.ConfigurationTypes) > 0 {
		query = query.Where("configuration_type IN ?", params.ConfigurationTypes)
	}

	if len(params.Statuses) > 0 {
		query = query.Where("status IN ?", params.Statuses)
	}

	query = query.Session(&gorm.Session{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]db.DataDiscoveryConfiguration, 0, pageSize)

	if total > int64(utils.Offset(page, pageSize)) {
		err := query.
			Select(configurationColumns).
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

func (r *ConfigurationRepository) Find(
	ctx context.Context,
	customerID, id int,
) (db.DataDiscoveryConfiguration, error) {
	var row db.DataDiscoveryConfiguration

	err := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryConfiguration{}).
		Select(configurationColumns).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *ConfigurationRepository) Lock(
	ctx context.Context,
	customerID, id int,
) (db.DataDiscoveryConfiguration, error) {
	var row db.DataDiscoveryConfiguration

	err := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryConfiguration{}).
		Select(configurationColumns).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *ConfigurationRepository) Insert(ctx context.Context, row *db.DataDiscoveryConfiguration) error {
	return r.db.WithContext(ctx).Omit("CreatedAt", "UpdatedAt").Create(row).Error
}

func (r *ConfigurationRepository) Update(
	ctx context.Context,
	customerID, id int,
	updates map[string]any,
) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryConfiguration{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Updates(updates)

	return result.RowsAffected, result.Error
}

func (r *ConfigurationRepository) Delete(ctx context.Context, customerID, id int) (int64, error) {
	result := r.db.WithContext(ctx).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Delete(&db.DataDiscoveryConfiguration{})

	return result.RowsAffected, result.Error
}

func (r *ConfigurationRepository) UpsertCredential(
	ctx context.Context,
	customerID, configurationID int,
	sealed []byte,
	keyVersion int,
) error {
	row := db.DataDiscoveryCredential{
		CustomerID:      customerID,
		ConfigurationID: configurationID,
		Secret:          sealed,
		KeyVersion:      keyVersion,
	}

	return r.db.WithContext(ctx).
		Omit("CreatedAt", "UpdatedAt").
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "configuration_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"secret", "key_version"}),
		}).
		Create(&row).Error
}

func (r *ConfigurationRepository) LoadSealed(
	ctx context.Context,
	customerID, configurationID int,
) (db.DataDiscoveryCredential, error) {
	var row db.DataDiscoveryCredential

	err := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryCredential{}).
		Scopes(db.TenantScope(customerID)).
		Where("configuration_id = ?", configurationID).
		Take(&row).Error

	return row, err
}

func (r *ConfigurationRepository) HasCredential(
	ctx context.Context,
	customerID, configurationID int,
) (bool, error) {
	var row db.DataDiscoveryCredential

	err := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryCredential{}).
		Select(credentialMetaColumns).
		Scopes(db.TenantScope(customerID)).
		Where("configuration_id = ?", configurationID).
		Take(&row).Error

	if db.IsNotFound(err) {
		return false, nil
	}

	return err == nil, err
}

func (r *ConfigurationRepository) PolicyCountsFor(
	ctx context.Context,
	customerID int,
	configurationIDs []int,
) ([]ConfigurationPolicyCount, error) {
	rows := make([]ConfigurationPolicyCount, 0, len(configurationIDs))

	if len(configurationIDs) == 0 {
		return rows, nil
	}

	err := r.db.WithContext(ctx).
		Table("data_discovery_policies AS p").
		Select("p.configuration_id AS configuration_id, count(*) AS total").
		Scopes(db.TenantScopeOn("p", customerID)).
		Where("p.configuration_id IN ?", configurationIDs).
		Group("p.configuration_id").
		Scan(&rows).Error

	return rows, err
}

func (r *ConfigurationRepository) CountPolicies(
	ctx context.Context,
	customerID, configurationID int,
) (int64, error) {
	var total int64

	err := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryPolicy{}).
		Scopes(db.TenantScope(customerID)).
		Where("configuration_id = ?", configurationID).
		Count(&total).Error

	return total, err
}

func (r *ConfigurationRepository) Capabilities(ctx context.Context) ([]db.DataDiscoverySourceCapability, error) {
	rows := make([]db.DataDiscoverySourceCapability, 0)

	err := r.db.WithContext(ctx).
		Model(&db.DataDiscoverySourceCapability{}).
		Where("active = ?", true).
		Order("configuration_type ASC, source_type ASC").
		Find(&rows).Error

	return rows, err
}
