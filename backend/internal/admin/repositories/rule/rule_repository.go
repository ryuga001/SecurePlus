package rule

import (
	"context"

	"gorm.io/gorm"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

type ListParams struct {
	Search   string
	Page     int
	PageSize int
}

type RuleRepository struct {
	db *gorm.DB
}

func NewRuleRepository(database *gorm.DB) *RuleRepository {
	return &RuleRepository{db: database}
}

func (r *RuleRepository) WithTx(tx *gorm.DB) *RuleRepository {
	return &RuleRepository{db: tx}
}

func (r *RuleRepository) List(ctx context.Context, customerID int, params ListParams) ([]db.Rule, int64, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	query := r.db.WithContext(ctx).
		Model(&db.Rule{}).
		Scopes(db.TenantScope(customerID))

	if search := utils.NormalizeName(params.Search); search != "" {
		query = query.Where(`(rule_name ILIKE ? ESCAPE '\')`, utils.LikePattern(search))
	}

	query = query.Session(&gorm.Session{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]db.Rule, 0, pageSize)

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

func (r *RuleRepository) Find(ctx context.Context, customerID, id int) (db.Rule, error) {
	var row db.Rule

	err := r.db.WithContext(ctx).
		Model(&db.Rule{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *RuleRepository) Insert(ctx context.Context, row *db.Rule) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *RuleRepository) Update(ctx context.Context, customerID, id int, updates map[string]any) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&db.Rule{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Updates(updates)

	return result.RowsAffected, result.Error
}

func (r *RuleRepository) Delete(ctx context.Context, customerID, id int) (int64, error) {
	result := r.db.WithContext(ctx).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Delete(&db.Rule{})

	return result.RowsAffected, result.Error
}
