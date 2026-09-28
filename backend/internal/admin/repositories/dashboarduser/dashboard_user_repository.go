package dashboarduser

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

type ListParams struct {
	Search   string
	RoleID   int
	Page     int
	PageSize int
}

type UserRow struct {
	ID        int       `gorm:"column:id"`
	FirstName string    `gorm:"column:first_name"`
	LastName  string    `gorm:"column:last_name"`
	Email     string    `gorm:"column:email"`
	CreatedAt time.Time `gorm:"column:created_at"`
	RoleID    *int      `gorm:"column:role_id"`
	RoleName  *string   `gorm:"column:role_name"`
	RoleType  *string   `gorm:"column:role_type"`
}

type DashboardUserRepository struct {
	db *gorm.DB
}

func NewDashboardUserRepository(database *gorm.DB) *DashboardUserRepository {
	return &DashboardUserRepository{db: database}
}

func (r *DashboardUserRepository) WithTx(tx *gorm.DB) *DashboardUserRepository {
	return &DashboardUserRepository{db: tx}
}

func (r *DashboardUserRepository) rows(ctx context.Context, customerID int) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("dashboard_users AS u").
		Select("u.id, u.first_name, u.last_name, u.email, u.created_at, u.role_id, r.name AS role_name, r.type AS role_type").
		Joins("LEFT JOIN roles AS r ON r.id = u.role_id").
		Where("u.customer_id = ?", customerID)
}

func (r *DashboardUserRepository) List(ctx context.Context, customerID int, params ListParams) ([]UserRow, int64, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	query := r.rows(ctx, customerID)

	if search := utils.NormalizeName(params.Search); search != "" {
		pattern := utils.LikePattern(search)
		query = query.Where(
			`(u.email ILIKE ? ESCAPE '\' OR u.first_name ILIKE ? ESCAPE '\' OR u.last_name ILIKE ? ESCAPE '\')`,
			pattern, pattern, pattern,
		)
	}

	if params.RoleID > 0 {
		query = query.Where("u.role_id = ?", params.RoleID)
	}

	query = query.Session(&gorm.Session{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]UserRow, 0, pageSize)

	if total > int64(utils.Offset(page, pageSize)) {
		err := query.
			Order("u.created_at DESC, u.id DESC").
			Limit(pageSize).
			Offset(utils.Offset(page, pageSize)).
			Scan(&rows).Error
		if err != nil {
			return nil, 0, err
		}
	}

	return rows, total, nil
}

func (r *DashboardUserRepository) Find(ctx context.Context, customerID, id int) (UserRow, error) {
	var row UserRow

	err := r.rows(ctx, customerID).
		Where("u.id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *DashboardUserRepository) Lock(ctx context.Context, customerID, id int) (db.DashboardUser, error) {
	var row db.DashboardUser

	err := r.db.WithContext(ctx).
		Model(&db.DashboardUser{}).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND customer_id = ?", id, customerID).
		Take(&row).Error

	return row, err
}

func (r *DashboardUserRepository) LockAdministratorIDs(ctx context.Context, customerID int) ([]int, error) {
	ids := make([]int, 0, 2)

	err := r.db.WithContext(ctx).
		Model(&db.DashboardUser{}).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("customer_id = ? AND role_id IN (?)", customerID,
			r.db.Model(&db.Role{}).Select("id").Where("type = ?", db.RoleTypeAdmin),
		).
		Order("id ASC").
		Pluck("id", &ids).Error

	return ids, err
}

func (r *DashboardUserRepository) Insert(ctx context.Context, row *db.DashboardUser) error {
	return r.db.WithContext(ctx).Omit("Customer", "Role").Create(row).Error
}

func (r *DashboardUserRepository) Update(ctx context.Context, customerID, id int, updates map[string]any) error {
	return r.db.WithContext(ctx).
		Model(&db.DashboardUser{}).
		Where("id = ? AND customer_id = ?", id, customerID).
		Updates(updates).Error
}

func (r *DashboardUserRepository) Delete(ctx context.Context, customerID, id int) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("id = ? AND customer_id = ?", id, customerID).
		Delete(&db.DashboardUser{})

	return result.RowsAffected, result.Error
}
