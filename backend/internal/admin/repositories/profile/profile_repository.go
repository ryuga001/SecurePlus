package profile

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"dpdp-backend/internal/db"
)

var ErrOrgNameConflict = errors.New("organization name conflict")

type ProfileRow struct {
	OrgName   string `gorm:"column:org_name"`
	FirstName string `gorm:"column:first_name"`
	LastName  string `gorm:"column:last_name"`
	Email     string `gorm:"column:email"`
}

type Changes struct {
	OrgName   *string
	FirstName *string
	LastName  *string
}

type ProfileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(database *gorm.DB) *ProfileRepository {
	return &ProfileRepository{db: database}
}

func (r *ProfileRepository) Find(ctx context.Context, customerID, userID int) (ProfileRow, error) {
	return find(r.db.WithContext(ctx), customerID, userID)
}

func (r *ProfileRepository) Update(ctx context.Context, customerID, userID int, changes Changes) (ProfileRow, error) {
	var row ProfileRow

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if changes.OrgName != nil {
			if err := renameOrganization(tx, customerID, *changes.OrgName); err != nil {
				return err
			}
		}

		if updates := nameUpdates(changes); len(updates) > 0 {
			result := tx.Model(&db.DashboardUser{}).
				Where("id = ? AND customer_id = ?", userID, customerID).
				Updates(updates)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
		}

		var err error
		row, err = find(tx, customerID, userID)

		return err
	})

	return row, err
}

func renameOrganization(tx *gorm.DB, customerID int, name string) error {
	var taken int64

	err := tx.Model(&db.Customer{}).
		Where("lower(org_name) = lower(?) AND id <> ?", name, customerID).
		Count(&taken).Error
	if err != nil {
		return err
	}
	if taken > 0 {
		return ErrOrgNameConflict
	}

	result := tx.Model(&db.Customer{}).Where("id = ?", customerID).Update("org_name", name)
	if db.IsDuplicate(result.Error) {
		return ErrOrgNameConflict
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func nameUpdates(changes Changes) map[string]any {
	updates := make(map[string]any, 2)

	if changes.FirstName != nil {
		updates["first_name"] = *changes.FirstName
	}
	if changes.LastName != nil {
		updates["last_name"] = *changes.LastName
	}

	return updates
}

func find(tx *gorm.DB, customerID, userID int) (ProfileRow, error) {
	var row ProfileRow

	err := tx.Table("dashboard_users AS u").
		Select("c.org_name, u.first_name, u.last_name, u.email").
		Joins("JOIN customers c ON c.id = u.customer_id").
		Where("u.id = ? AND u.customer_id = ?", userID, customerID).
		Take(&row).Error

	return row, err
}
