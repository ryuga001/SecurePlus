package branding

import (
	"context"

	"gorm.io/gorm"

	"dpdp-backend/internal/db"
)

type BrandingRepository struct {
	db *gorm.DB
}

func NewBrandingRepository(database *gorm.DB) *BrandingRepository {
	return &BrandingRepository{db: database}
}

func (r *BrandingRepository) Find(ctx context.Context, customerID int) (db.CustomerBranding, error) {
	var row db.CustomerBranding

	err := r.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		Take(&row).Error

	return row, err
}

func (r *BrandingRepository) Update(ctx context.Context, customerID int, updates map[string]any) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&db.CustomerBranding{}).
		Where("customer_id = ?", customerID).
		Updates(updates)

	return result.RowsAffected, result.Error
}
