package emailuser

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

type EmailUserRepository struct {
	db *gorm.DB
}

func NewEmailUserRepository(database *gorm.DB) *EmailUserRepository {
	return &EmailUserRepository{db: database}
}

func (r *EmailUserRepository) WithTx(tx *gorm.DB) *EmailUserRepository {
	return &EmailUserRepository{db: tx}
}

func (r *EmailUserRepository) List(ctx context.Context, customerID int, params ListParams) ([]db.EmailUser, int64, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	query := r.db.WithContext(ctx).
		Model(&db.EmailUser{}).
		Scopes(db.TenantScope(customerID))

	if search := utils.NormalizeName(params.Search); search != "" {
		pattern := utils.LikePattern(search)
		query = query.Where(
			`(email ILIKE ? ESCAPE '\' OR first_name ILIKE ? ESCAPE '\' OR last_name ILIKE ? ESCAPE '\')`,
			pattern, pattern, pattern,
		)
	}

	query = query.Session(&gorm.Session{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]db.EmailUser, 0, pageSize)

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

func (r *EmailUserRepository) Find(ctx context.Context, customerID, id int) (db.EmailUser, error) {
	var row db.EmailUser

	err := r.db.WithContext(ctx).
		Model(&db.EmailUser{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *EmailUserRepository) Insert(ctx context.Context, row *db.EmailUser) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *EmailUserRepository) Update(ctx context.Context, customerID, id int, updates map[string]any) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&db.EmailUser{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Updates(updates)

	return result.RowsAffected, result.Error
}

func (r *EmailUserRepository) Delete(ctx context.Context, customerID, id int) (int64, error) {
	result := r.db.WithContext(ctx).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Delete(&db.EmailUser{})

	return result.RowsAffected, result.Error
}

func (r *EmailUserRepository) GroupsOf(ctx context.Context, customerID, emailUserID int) ([]db.Group, error) {
	rows := make([]db.Group, 0)

	err := r.db.WithContext(ctx).
		Table("email_user_group_mapping AS m").
		Select("g.id, g.customer_id, g.name, g.type, g.created_at, g.updated_at").
		Joins("JOIN groups AS g ON g.id = m.group_id AND g.customer_id = m.customer_id").
		Scopes(db.TenantScopeOn("m", customerID)).
		Where("m.email_user_id = ?", emailUserID).
		Order("g.name ASC, g.id ASC").
		Scan(&rows).Error

	return rows, err
}
