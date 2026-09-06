package notification

import (
	"context"
	"errors"
	"html"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"dpdp-backend/internal/config"
	"dpdp-backend/internal/db"
)

const PlatformCustomerID = 1

const (
	TemplateEmailVerification = "email_verification"
	TemplateWelcome           = "welcome"
	TemplatePasswordReset     = "password_reset"
	TemplatePasswordChanged   = "password_changed"
)

var ErrTemplateNotFound = errors.New("email template not found")

type Email struct {
	CustomerID int
	Template   string
	To         string
	Vars       map[string]string
}

type Sender interface {
	Send(ctx context.Context, e Email) error
}

type Service struct {
	db   *gorm.DB
	smtp config.SMTP
}

func NewService(database *gorm.DB, smtp config.SMTP) *Service {
	return &Service{db: database, smtp: smtp}
}

func (s *Service) Send(ctx context.Context, e Email) error {
	template, err := s.resolve(ctx, e.CustomerID, e.Template)
	if err != nil {
		return err
	}

	return s.deliver(ctx, e.To, renderSubject(template.Subject, e.Vars), renderBody(template.Body, e.Vars))
}

func (s *Service) resolve(ctx context.Context, customerID int, name string) (db.EmailTemplate, error) {
	var template db.EmailTemplate

	err := s.db.WithContext(ctx).
		Where("name = ? AND customer_id IN ?", name, []int{PlatformCustomerID, customerID}).
		Clauses(clause.OrderBy{
			Expression: clause.Expr{SQL: "(customer_id = ?) DESC", Vars: []any{customerID}},
		}).
		Take(&template).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.EmailTemplate{}, ErrTemplateNotFound
	}

	return template, err
}

func renderBody(body string, vars map[string]string) string {
	pairs := make([]string, 0, len(vars)*2)
	for key, value := range vars {
		pairs = append(pairs, "{{"+key+"}}", html.EscapeString(value))
	}

	return strings.NewReplacer(pairs...).Replace(body)
}

var subjectSanitizer = strings.NewReplacer("\r", "", "\n", "")

func renderSubject(subject string, vars map[string]string) string {
	pairs := make([]string, 0, len(vars)*2)
	for key, value := range vars {
		pairs = append(pairs, "{{"+key+"}}", subjectSanitizer.Replace(value))
	}

	return strings.NewReplacer(pairs...).Replace(subject)
}
