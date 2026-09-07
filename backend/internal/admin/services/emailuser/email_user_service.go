package emailuser

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	repo "dpdp-backend/internal/admin/repositories/emailuser"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

type Listing struct {
	Items    []db.EmailUser
	Page     int
	PageSize int
	Total    int64
}

type EmailUserService struct {
	db   *gorm.DB
	repo *repo.EmailUserRepository
}

func NewEmailUserService(database *gorm.DB, repo *repo.EmailUserRepository) *EmailUserService {
	return &EmailUserService{db: database, repo: repo}
}

func (s *EmailUserService) List(ctx context.Context, customerID int, params repo.ListParams) (Listing, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	rows, total, err := s.repo.List(ctx, customerID, params)
	if err != nil {
		return Listing{}, err
	}

	return Listing{Items: rows, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *EmailUserService) Get(ctx context.Context, customerID, id int) (db.EmailUser, error) {
	row, err := s.repo.Find(ctx, customerID, id)
	if err != nil {
		return db.EmailUser{}, emailUserError(err, "")
	}

	return row, nil
}

func (s *EmailUserService) Create(ctx context.Context, customerID int, in EmailUserInput) (db.EmailUser, error) {
	input, err := NormalizeEmailUserInput(in)
	if err != nil {
		return db.EmailUser{}, err
	}

	row := db.EmailUser{
		CustomerID: customerID,
		Email:      input.Email,
		FirstName:  input.FirstName,
		LastName:   input.LastName,
	}

	if err := s.repo.Insert(ctx, &row); err != nil {
		return db.EmailUser{}, emailUserError(err, input.Email)
	}

	return row, nil
}

func (s *EmailUserService) Update(ctx context.Context, customerID, id int, in EmailUserInput) (db.EmailUser, error) {
	input, err := NormalizeEmailUserInput(in)
	if err != nil {
		return db.EmailUser{}, err
	}

	updates := map[string]any{
		"email":      input.Email,
		"first_name": input.FirstName,
		"last_name":  input.LastName,
		"updated_at": time.Now(),
	}

	affected, err := s.repo.Update(ctx, customerID, id, updates)
	if err != nil {
		return db.EmailUser{}, emailUserError(err, input.Email)
	}
	if affected == 0 {
		return db.EmailUser{}, utils.ErrEmailUserNotFound
	}

	return s.Get(ctx, customerID, id)
}

func (s *EmailUserService) Delete(ctx context.Context, customerID, id int) error {
	affected, err := s.repo.Delete(ctx, customerID, id)
	if err != nil {
		return emailUserError(err, "")
	}
	if affected == 0 {
		return utils.ErrEmailUserNotFound
	}

	return nil
}

func (s *EmailUserService) Groups(ctx context.Context, customerID, id int) ([]db.Group, error) {
	if _, err := s.Get(ctx, customerID, id); err != nil {
		return nil, err
	}

	rows, err := s.repo.GroupsOf(ctx, customerID, id)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func emailUserError(err error, email string) error {
	switch {
	case db.IsNotFound(err):
		return utils.ErrEmailUserNotFound
	case db.IsDuplicate(err):
		return utils.Taken(utils.ErrEmailTaken, email)
	}

	return err
}

type EmailUserInput struct {
	Email     string
	FirstName string
	LastName  string
}

func NormalizeEmailUserInput(in EmailUserInput) (EmailUserInput, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" || !strings.Contains(email, "@") || strings.ContainsAny(email, " \t") {
		return EmailUserInput{}, utils.ErrInvalidEmail
	}

	first := utils.NormalizeName(in.FirstName)
	last := utils.NormalizeName(in.LastName)

	if first == "" || last == "" {
		return EmailUserInput{}, utils.ErrNameNeeded
	}

	return EmailUserInput{Email: email, FirstName: first, LastName: last}, nil
}
