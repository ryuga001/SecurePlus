package role

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

const (
	LockUpdate   = "UPDATE"
	LockKeyShare = "KEY SHARE"

	maxOptions = 500

	visibleOrder = "CASE WHEN roles.type = 'admin' THEN 0 ELSE 1 END, roles.name ASC, roles.id ASC"
)

type ListParams struct {
	Search   string
	Page     int
	PageSize int
}

type PrivilegeRef struct {
	ID   int    `gorm:"column:id"`
	Name string `gorm:"column:name"`
}

type RolePrivilege struct {
	RoleID int    `gorm:"column:role_id"`
	Name   string `gorm:"column:name"`
}

type UserPreview struct {
	RoleID    int    `gorm:"column:role_id"`
	ID        int    `gorm:"column:id"`
	FirstName string `gorm:"column:first_name"`
	LastName  string `gorm:"column:last_name"`
	Email     string `gorm:"column:email"`
}

type RoleCount struct {
	RoleID int `gorm:"column:role_id"`
	Total  int `gorm:"column:total"`
}

type RoleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(database *gorm.DB) *RoleRepository {
	return &RoleRepository{db: database}
}

func (r *RoleRepository) WithTx(tx *gorm.DB) *RoleRepository {
	return &RoleRepository{db: tx}
}

func Visible(customerID int) func(*gorm.DB) *gorm.DB {
	return func(query *gorm.DB) *gorm.DB {
		return query.Where(
			"((roles.customer_id = ? AND roles.type = ?) OR (roles.customer_id = ? AND roles.type = ?))",
			customerID, db.RoleTypeCustom, db.SystemCustomerID, db.RoleTypeAdmin,
		)
	}
}

func (r *RoleRepository) List(ctx context.Context, customerID int, params ListParams) ([]db.Role, int64, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	query := r.db.WithContext(ctx).
		Model(&db.Role{}).
		Scopes(Visible(customerID))

	if search := utils.NormalizeName(params.Search); search != "" {
		query = query.Where(`(roles.name ILIKE ? ESCAPE '\')`, utils.LikePattern(search))
	}

	query = query.Session(&gorm.Session{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]db.Role, 0, pageSize)

	if total > int64(utils.Offset(page, pageSize)) {
		err := query.
			Order(visibleOrder).
			Limit(pageSize).
			Offset(utils.Offset(page, pageSize)).
			Find(&rows).Error
		if err != nil {
			return nil, 0, err
		}
	}

	return rows, total, nil
}

func (r *RoleRepository) Options(ctx context.Context, customerID int) ([]db.Role, error) {
	rows := make([]db.Role, 0)

	err := r.db.WithContext(ctx).
		Model(&db.Role{}).
		Scopes(Visible(customerID)).
		Order(visibleOrder).
		Limit(maxOptions).
		Find(&rows).Error

	return rows, err
}

func (r *RoleRepository) FindVisible(ctx context.Context, customerID, id int) (db.Role, error) {
	var row db.Role

	err := r.db.WithContext(ctx).
		Model(&db.Role{}).
		Scopes(Visible(customerID)).
		Where("roles.id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *RoleRepository) LockVisible(ctx context.Context, customerID, id int, strength string) (db.Role, error) {
	var row db.Role

	err := r.db.WithContext(ctx).
		Model(&db.Role{}).
		Clauses(clause.Locking{Strength: strength}).
		Scopes(Visible(customerID)).
		Where("roles.id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *RoleRepository) SystemNames(ctx context.Context) ([]string, error) {
	names := make([]string, 0, 1)

	err := r.db.WithContext(ctx).
		Model(&db.Role{}).
		Where("customer_id = ? AND type = ?", db.SystemCustomerID, db.RoleTypeAdmin).
		Pluck("name", &names).Error

	return names, err
}

func (r *RoleRepository) Insert(ctx context.Context, row *db.Role) error {
	return r.db.WithContext(ctx).Omit("Privileges").Create(row).Error
}

func (r *RoleRepository) Update(ctx context.Context, id int, updates map[string]any) error {
	return r.db.WithContext(ctx).
		Model(&db.Role{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *RoleRepository) Delete(ctx context.Context, id int) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&db.Role{})

	return result.RowsAffected, result.Error
}

func (r *RoleRepository) FindPrivileges(ctx context.Context, names []string) ([]PrivilegeRef, error) {
	rows := make([]PrivilegeRef, 0, len(names))

	if len(names) == 0 {
		return rows, nil
	}

	err := r.db.WithContext(ctx).
		Model(&db.Privilege{}).
		Select("id, name").
		Where("name IN ? AND type = ?", names, db.PrivilegeTypeDashboard).
		Scan(&rows).Error

	return rows, err
}

func (r *RoleRepository) AssignablePrivileges(ctx context.Context) ([]string, error) {
	names := make([]string, 0)

	err := r.db.WithContext(ctx).
		Model(&db.Privilege{}).
		Where("type = ?", db.PrivilegeTypeDashboard).
		Order("name ASC").
		Pluck("name", &names).Error

	return names, err
}

func (r *RoleRepository) ReplacePrivileges(ctx context.Context, roleID int, privilegeIDs []int) error {
	err := r.db.WithContext(ctx).
		Exec("DELETE FROM role_privileges WHERE role_id = ?", roleID).Error
	if err != nil {
		return err
	}

	if len(privilegeIDs) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		Exec(`INSERT INTO role_privileges (role_id, privilege_id)
			SELECT ?, id FROM privileges WHERE id IN ?
			ON CONFLICT DO NOTHING`, roleID, privilegeIDs).Error
}

func (r *RoleRepository) PrivilegesFor(ctx context.Context, roleIDs []int) ([]RolePrivilege, error) {
	rows := make([]RolePrivilege, 0)

	if len(roleIDs) == 0 {
		return rows, nil
	}

	err := r.db.WithContext(ctx).
		Table("role_privileges AS rp").
		Select("rp.role_id AS role_id, p.name AS name").
		Joins("JOIN privileges AS p ON p.id = rp.privilege_id").
		Where("rp.role_id IN ?", roleIDs).
		Order("p.name ASC").
		Scan(&rows).Error

	return rows, err
}

func (r *RoleRepository) UserCounts(ctx context.Context, customerID int, roleIDs []int) ([]RoleCount, error) {
	rows := make([]RoleCount, 0, len(roleIDs))

	if len(roleIDs) == 0 {
		return rows, nil
	}

	err := r.db.WithContext(ctx).
		Model(&db.DashboardUser{}).
		Select("role_id, COUNT(*) AS total").
		Where("customer_id = ? AND role_id IN ?", customerID, roleIDs).
		Group("role_id").
		Scan(&rows).Error

	return rows, err
}

func (r *RoleRepository) UserPreviews(ctx context.Context, customerID int, roleIDs []int, limit int) ([]UserPreview, error) {
	rows := make([]UserPreview, 0)

	if len(roleIDs) == 0 {
		return rows, nil
	}

	err := r.db.WithContext(ctx).Raw(`
		SELECT t.role_id, t.id, t.first_name, t.last_name, t.email
		FROM (
			SELECT
				u.role_id,
				u.id,
				u.first_name,
				u.last_name,
				u.email,
				ROW_NUMBER() OVER (PARTITION BY u.role_id ORDER BY u.first_name, u.last_name, u.id) AS rn
			FROM dashboard_users u
			WHERE u.customer_id = ? AND u.role_id IN (?)
		) t
		WHERE t.rn <= ?`,
		customerID, roleIDs, limit,
	).Scan(&rows).Error

	return rows, err
}

func (r *RoleRepository) CountUsers(ctx context.Context, roleID int) (int64, error) {
	var total int64

	err := r.db.WithContext(ctx).
		Model(&db.DashboardUser{}).
		Where("role_id = ?", roleID).
		Count(&total).Error

	return total, err
}
