package dashboarduser

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"slices"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	repo "dpdp-backend/internal/admin/repositories/dashboarduser"
	rolerepo "dpdp-backend/internal/admin/repositories/role"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/db"
	"dpdp-backend/internal/notification"
)

const (
	maxNameLength      = 50
	maxEmailLength     = 100
	temporaryPassBytes = 15
)

type Cache interface {
	DropIdentity(ctx context.Context, userID int) error
	BumpVersion(ctx context.Context, userID int) error
}

type Hashing struct {
	Pepper string
	Cost   int
}

type Actor struct {
	CustomerID int
	UserID     int
}

type UserInput struct {
	FirstName string
	LastName  string
	Email     string
	RoleID    int
}

type User = repo.UserRow

type UserListing struct {
	Items    []User
	Page     int
	PageSize int
	Total    int64
}

type DashboardUserService struct {
	db       *gorm.DB
	repo     *repo.DashboardUserRepository
	roles    *rolerepo.RoleRepository
	cache    Cache
	sender   notification.Sender
	hashing  Hashing
	loginURL string
}

func NewDashboardUserService(
	database *gorm.DB,
	repository *repo.DashboardUserRepository,
	roles *rolerepo.RoleRepository,
	cache Cache,
	sender notification.Sender,
	hashing Hashing,
	frontendURL string,
) *DashboardUserService {
	return &DashboardUserService{
		db:       database,
		repo:     repository,
		roles:    roles,
		cache:    cache,
		sender:   sender,
		hashing:  hashing,
		loginURL: strings.TrimRight(frontendURL, "/") + "/login",
	}
}

func (s *DashboardUserService) List(ctx context.Context, customerID int, params repo.ListParams) (UserListing, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	rows, total, err := s.repo.List(ctx, customerID, params)
	if err != nil {
		return UserListing{}, err
	}

	return UserListing{Items: rows, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *DashboardUserService) Get(ctx context.Context, customerID, id int) (User, error) {
	row, err := s.repo.Find(ctx, customerID, id)
	if err != nil {
		return User{}, userError(err)
	}

	return row, nil
}

func (s *DashboardUserService) Create(ctx context.Context, actor Actor, in UserInput) (User, bool, error) {
	input, err := NormalizeUserInput(in, true)
	if err != nil {
		return User{}, false, err
	}

	password, hash, salt, err := s.temporaryCredentials()
	if err != nil {
		return User{}, false, err
	}

	var created User

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		if _, err := s.roles.WithTx(tx).LockVisible(ctx, actor.CustomerID, input.RoleID, rolerepo.LockKeyShare); err != nil {
			return roleLookupError(err)
		}

		row := db.DashboardUser{
			CustomerID:   actor.CustomerID,
			RoleID:       &input.RoleID,
			FirstName:    input.FirstName,
			LastName:     input.LastName,
			Email:        input.Email,
			PasswordHash: hash,
			PasswordSalt: salt,
		}

		if err := repository.Insert(ctx, &row); err != nil {
			return err
		}

		created, err = repository.Find(ctx, actor.CustomerID, row.ID)

		return err
	})

	if err != nil {
		return User{}, false, userError(err)
	}

	return created, s.invite(ctx, actor.CustomerID, created, password), nil
}

func (s *DashboardUserService) Update(ctx context.Context, actor Actor, id int, in UserInput) (User, error) {
	input, err := NormalizeUserInput(in, false)
	if err != nil {
		return User{}, err
	}

	var updated User

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		role, err := s.roles.WithTx(tx).LockVisible(ctx, actor.CustomerID, input.RoleID, rolerepo.LockKeyShare)
		if err != nil {
			return roleLookupError(err)
		}

		var administrators []int
		if role.Type != db.RoleTypeAdmin {
			administrators, err = repository.LockAdministratorIDs(ctx, actor.CustomerID)
			if err != nil {
				return err
			}
		}

		current, err := repository.Lock(ctx, actor.CustomerID, id)
		if err != nil {
			return err
		}

		if current.RoleID == nil || *current.RoleID != input.RoleID {
			if id == actor.UserID {
				return utils.ErrSelfModification
			}
			if isLastAdministrator(administrators, id) {
				return utils.ErrLastAdministrator
			}
		}

		updates := map[string]any{
			"first_name": input.FirstName,
			"last_name":  input.LastName,
			"role_id":    input.RoleID,
		}

		if err := repository.Update(ctx, actor.CustomerID, id, updates); err != nil {
			return err
		}

		updated, err = repository.Find(ctx, actor.CustomerID, id)

		return err
	})

	if err != nil {
		return User{}, userError(err)
	}

	s.dropIdentity(ctx, id)

	return updated, nil
}

func (s *DashboardUserService) Delete(ctx context.Context, actor Actor, id int) error {
	if id == actor.UserID {
		return utils.ErrSelfModification
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		administrators, err := repository.LockAdministratorIDs(ctx, actor.CustomerID)
		if err != nil {
			return err
		}

		if _, err := repository.Lock(ctx, actor.CustomerID, id); err != nil {
			return err
		}

		if isLastAdministrator(administrators, id) {
			return utils.ErrLastAdministrator
		}

		_, err = repository.Delete(ctx, actor.CustomerID, id)

		return err
	})

	if err != nil {
		return userError(err)
	}

	s.dropIdentity(ctx, id)

	if err := s.cache.BumpVersion(ctx, id); err != nil {
		slog.WarnContext(ctx, "deleted user session revocation failed", "user_id", id, "error", err)
	}

	return nil
}

func (s *DashboardUserService) temporaryCredentials() (string, string, string, error) {
	raw := make([]byte, temporaryPassBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", "", err
	}

	password := base64.RawURLEncoding.EncodeToString(raw)

	salt, err := auth.NewSalt()
	if err != nil {
		return "", "", "", err
	}

	hash, err := auth.HashPassword(s.hashing.Pepper, password, salt, s.hashing.Cost)
	if err != nil {
		return "", "", "", err
	}

	return password, hash, salt, nil
}

func (s *DashboardUserService) invite(ctx context.Context, customerID int, user User, password string) bool {
	err := s.sender.Send(ctx, notification.Email{
		CustomerID: customerID,
		Template:   notification.TemplateUserInvited,
		To:         []string{user.Email},
		Vars: map[string]string{
			"name":          user.FirstName,
			"email":         user.Email,
			"temp_password": password,
			"login_url":     s.loginURL,
		},
	})
	if err != nil {
		slog.WarnContext(ctx, "user invitation failed", "customer_id", customerID, "user_id", user.ID, "error", err)
		return false
	}

	return true
}

func (s *DashboardUserService) dropIdentity(ctx context.Context, userID int) {
	if err := s.cache.DropIdentity(ctx, userID); err != nil {
		slog.WarnContext(ctx, "identity cache invalidation failed", "user_id", userID, "error", err)
	}
}

func isLastAdministrator(administrators []int, id int) bool {
	return len(administrators) <= 1 && slices.Contains(administrators, id)
}

func roleLookupError(err error) error {
	if db.IsNotFound(err) {
		return utils.ErrUnknownRole
	}

	return err
}

func userError(err error) error {
	switch {
	case db.IsNotFound(err):
		return utils.ErrDashboardUserNotFound
	case db.IsDuplicate(err):
		return utils.ErrEmailTaken
	case db.IsMissingReference(err):
		return utils.ErrUnknownRole
	}

	return err
}

func NormalizeUserInput(in UserInput, creating bool) (UserInput, error) {
	first := utils.NormalizeName(in.FirstName)
	last := utils.NormalizeName(in.LastName)

	if first == "" || last == "" {
		return UserInput{}, utils.ErrNameNeeded
	}
	if utf8.RuneCountInString(first) > maxNameLength || utf8.RuneCountInString(last) > maxNameLength {
		return UserInput{}, utils.ErrNameTooLong
	}

	if in.RoleID < 1 {
		return UserInput{}, utils.ErrUnknownRole
	}

	normalized := UserInput{FirstName: first, LastName: last, RoleID: in.RoleID}

	if creating {
		email := utils.NormalizeEmail(in.Email)
		if email == "" || len(email) > maxEmailLength {
			return UserInput{}, utils.ErrInvalidEmail
		}

		normalized.Email = email
	}

	return normalized, nil
}
