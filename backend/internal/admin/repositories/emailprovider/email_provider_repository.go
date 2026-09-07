package emailprovider

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

type ListParams struct {
	Search   string
	Page     int
	PageSize int
}

var configColumns = []string{
	"id", "customer_id", "name", "domain", "provider",
	"dkim_public_key", "access_token_expires_at", "created_at", "updated_at",
}

type EmailProviderRepository struct {
	db *gorm.DB
}

func NewEmailProviderRepository(database *gorm.DB) *EmailProviderRepository {
	return &EmailProviderRepository{db: database}
}

func (r *EmailProviderRepository) WithTx(tx *gorm.DB) *EmailProviderRepository {
	return &EmailProviderRepository{db: tx}
}

func (r *EmailProviderRepository) List(ctx context.Context, customerID int, params ListParams) ([]db.EmailProviderConfiguration, int64, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	query := r.db.WithContext(ctx).
		Model(&db.EmailProviderConfiguration{}).
		Scopes(db.TenantScope(customerID))

	if search := utils.NormalizeName(params.Search); search != "" {
		pattern := utils.LikePattern(search)
		query = query.Where(`(name ILIKE ? ESCAPE '\' OR domain ILIKE ? ESCAPE '\')`, pattern, pattern)
	}

	query = query.Session(&gorm.Session{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]db.EmailProviderConfiguration, 0, pageSize)

	if total > int64(utils.Offset(page, pageSize)) {
		err := query.Select(configColumns).
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

func (r *EmailProviderRepository) Find(ctx context.Context, customerID, id int) (db.EmailProviderConfiguration, error) {
	var row db.EmailProviderConfiguration

	err := r.db.WithContext(ctx).
		Model(&db.EmailProviderConfiguration{}).
		Select(configColumns).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *EmailProviderRepository) Lock(ctx context.Context, customerID, id int) (db.EmailProviderConfiguration, error) {
	var row db.EmailProviderConfiguration

	err := r.db.WithContext(ctx).
		Model(&db.EmailProviderConfiguration{}).
		Select(configColumns).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *EmailProviderRepository) Insert(ctx context.Context, row *db.EmailProviderConfiguration) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *EmailProviderRepository) Update(ctx context.Context, customerID, id int, updates map[string]any) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&db.EmailProviderConfiguration{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Updates(updates)

	return result.RowsAffected, result.Error
}

func (r *EmailProviderRepository) Delete(ctx context.Context, customerID, id int) (int64, error) {
	result := r.db.WithContext(ctx).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Delete(&db.EmailProviderConfiguration{})

	return result.RowsAffected, result.Error
}

func (r *EmailProviderRepository) TenantSecret(ctx context.Context, customerID int) (string, error) {
	var customer db.Customer

	err := r.db.WithContext(ctx).
		Model(&db.Customer{}).
		Select("id", "jwt_secret").
		Where("id = ?", customerID).
		Take(&customer).Error

	return customer.JWTSecret, err
}
