package emailtemplate

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"dpdp-backend/internal/db"
	"dpdp-backend/internal/delivery/utils"
)

type EmailTemplateRepository struct {
	db *gorm.DB
}

func NewEmailTemplateRepository(database *gorm.DB) *EmailTemplateRepository {
	return &EmailTemplateRepository{db: database}
}

func (r *EmailTemplateRepository) Find(ctx context.Context, customerID int, name string) (db.EmailTemplate, error) {
	var template db.EmailTemplate

	err := r.db.WithContext(ctx).
		Where("name = ? AND customer_id IN ?", name, []int{db.SystemCustomerID, customerID}).
		Clauses(clause.OrderBy{
			Expression: clause.Expr{SQL: "(customer_id = ?) DESC", Vars: []any{customerID}},
		}).
		Take(&template).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.EmailTemplate{}, utils.ErrNoticeTemplateMissing
	}

	return template, err
}

func (r *EmailTemplateRepository) OrgName(ctx context.Context, customerID int) (string, error) {
	var customer db.Customer

	err := r.db.WithContext(ctx).
		Select("id", "org_name").
		Where("id = ?", customerID).
		Take(&customer).Error
	if err != nil {
		return "", err
	}

	return customer.OrgName, nil
}
