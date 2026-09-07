package group

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

type MemberListParams struct {
	Search   string
	Page     int
	PageSize int
}

type GroupMemberCount struct {
	GroupID int   `gorm:"column:group_id"`
	Total   int64 `gorm:"column:total"`
}

type GroupRepository struct {
	db *gorm.DB
}

func NewGroupRepository(database *gorm.DB) *GroupRepository {
	return &GroupRepository{db: database}
}

func (r *GroupRepository) WithTx(tx *gorm.DB) *GroupRepository {
	return &GroupRepository{db: tx}
}

func (r *GroupRepository) List(ctx context.Context, customerID int, params ListParams) ([]db.Group, int64, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	query := r.db.WithContext(ctx).
		Model(&db.Group{}).
		Scopes(db.TenantScope(customerID)).
		Where("type = ?", db.GroupTypeUser)

	if search := utils.NormalizeName(params.Search); search != "" {
		query = query.Where(`(name ILIKE ? ESCAPE '\')`, utils.LikePattern(search))
	}

	query = query.Session(&gorm.Session{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]db.Group, 0, pageSize)

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

func (r *GroupRepository) Find(ctx context.Context, customerID, id int) (db.Group, error) {
	var row db.Group

	err := r.db.WithContext(ctx).
		Model(&db.Group{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ? AND type = ?", id, db.GroupTypeUser).
		Take(&row).Error

	return row, err
}

func (r *GroupRepository) Insert(ctx context.Context, row *db.Group) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *GroupRepository) Update(ctx context.Context, customerID, id int, updates map[string]any) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&db.Group{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ? AND type = ?", id, db.GroupTypeUser).
		Updates(updates)

	return result.RowsAffected, result.Error
}

func (r *GroupRepository) Delete(ctx context.Context, customerID, id int) (int64, error) {
	result := r.db.WithContext(ctx).
		Scopes(db.TenantScope(customerID)).
		Where("id = ? AND type = ?", id, db.GroupTypeUser).
		Delete(&db.Group{})

	return result.RowsAffected, result.Error
}

func (r *GroupRepository) MemberCounts(ctx context.Context, customerID int, groupIDs []int) ([]GroupMemberCount, error) {
	rows := make([]GroupMemberCount, 0, len(groupIDs))

	if len(groupIDs) == 0 {
		return rows, nil
	}

	err := r.db.WithContext(ctx).
		Table("email_user_group_mapping AS m").
		Select("m.group_id AS group_id, count(*) AS total").
		Scopes(db.TenantScopeOn("m", customerID)).
		Where("m.group_id IN ?", groupIDs).
		Group("m.group_id").
		Scan(&rows).Error

	return rows, err
}

func (r *GroupRepository) FindEmailUser(ctx context.Context, customerID, emailUserID int) (db.EmailUser, error) {
	var row db.EmailUser

	err := r.db.WithContext(ctx).
		Model(&db.EmailUser{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", emailUserID).
		Take(&row).Error

	return row, err
}

func (r *GroupRepository) AddMember(ctx context.Context, customerID, groupID, emailUserID int) error {
	row := db.EmailUserGroupMapping{
		EmailUserID: emailUserID,
		GroupID:     groupID,
		CustomerID:  customerID,
	}

	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *GroupRepository) RemoveMember(ctx context.Context, customerID, groupID, emailUserID int) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("group_id = ? AND email_user_id = ? AND customer_id = ?", groupID, emailUserID, customerID).
		Delete(&db.EmailUserGroupMapping{})

	return result.RowsAffected, result.Error
}

func (r *GroupRepository) Members(ctx context.Context, customerID, groupID int, params MemberListParams) ([]db.EmailUser, int64, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	query := r.db.WithContext(ctx).
		Table("email_user_group_mapping AS m").
		Joins("JOIN email_users AS u ON u.id = m.email_user_id AND u.customer_id = m.customer_id").
		Scopes(db.TenantScopeOn("m", customerID)).
		Where("m.group_id = ?", groupID)

	if search := utils.NormalizeName(params.Search); search != "" {
		pattern := utils.LikePattern(search)
		query = query.Where(
			`(u.email ILIKE ? ESCAPE '\' OR u.first_name ILIKE ? ESCAPE '\' OR u.last_name ILIKE ? ESCAPE '\')`,
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
			Select("u.id, u.customer_id, u.email, u.first_name, u.last_name, u.created_at, u.updated_at").
			Order("u.email ASC, u.id ASC").
			Limit(pageSize).
			Offset(utils.Offset(page, pageSize)).
			Scan(&rows).Error
		if err != nil {
			return nil, 0, err
		}
	}

	return rows, total, nil
}
