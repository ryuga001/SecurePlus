package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"gorm.io/gorm"

	"dpdp-backend/internal/config"
	"dpdp-backend/internal/db"
	"dpdp-backend/internal/notification"
)

type Service struct {
	db       *gorm.DB
	store    *Store
	notifier notification.Sender
	branding BrandingProvider
	cfg      config.Auth
	app      config.App
}

func NewService(database *gorm.DB, store *Store, notifier notification.Sender, branding BrandingProvider, cfg config.Auth, app config.App) *Service {
	return &Service{db: database, store: store, notifier: notifier, branding: branding, cfg: cfg, app: app}
}

type Identity struct {
	IdentitySnapshot
	CSRFToken string `json:"csrf_token"`
}

type IdentitySnapshot struct {
	User     IdentityUser      `json:"user"`
	Customer IdentityCustomer  `json:"customer"`
	Branding *BrandingSnapshot `json:"branding,omitempty"`
}

type BrandingSnapshot struct {
	LogoURL  *string `json:"logo_url"`
	Theme    string  `json:"theme"`
	Language string  `json:"language"`
	Timezone string  `json:"timezone"`
}

type BrandingProvider interface {
	Snapshot(ctx context.Context, customerID int) (BrandingSnapshot, error)
}

type IdentityUser struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role,omitempty"`
}

type IdentityCustomer struct {
	ID      int    `json:"id"`
	OrgName string `json:"org_name"`
}

type TokenPair struct {
	Access  Issued
	Refresh Issued
}

func (s *Service) TenantSecret(ctx context.Context, customerID int) (string, error) {
	var customer db.Customer

	err := s.db.WithContext(ctx).Select("jwt_secret").First(&customer, customerID).Error
	if err != nil {
		if db.IsNotFound(err) {
			return "", ErrInvalidToken
		}

		return "", ErrUnavailable
	}

	return customer.JWTSecret, nil
}

func (s *Service) userByID(ctx context.Context, userID int) (db.DashboardUser, error) {
	var user db.DashboardUser

	err := s.db.WithContext(ctx).
		Preload("Customer").
		Preload("Role").
		First(&user, userID).Error

	return user, err
}

func (s *Service) userByEmail(ctx context.Context, email string) (db.DashboardUser, error) {
	var user db.DashboardUser

	err := s.db.WithContext(ctx).
		Preload("Customer").
		Preload("Role").
		Where("email = ?", email).
		Take(&user).Error

	return user, err
}

func (s *Service) Login(ctx context.Context, email, password string) (IdentitySnapshot, TokenPair, error) {
	key := Normalize(email)

	user, err := s.userByEmail(ctx, key)
	if err != nil {
		BurnTime(s.cfg.Pepper, password)
		return IdentitySnapshot{}, TokenPair{}, ErrInvalidCredentials
	}

	if err := VerifyPassword(s.cfg.Pepper, user.PasswordHash, user.PasswordSalt, password); err != nil {
		return IdentitySnapshot{}, TokenPair{}, ErrInvalidCredentials
	}

	if user.RoleID == nil {
		return IdentitySnapshot{}, TokenPair{}, ErrNoRole
	}

	pair, err := s.issue(ctx, user)
	if err != nil {
		return IdentitySnapshot{}, TokenPair{}, err
	}

	s.syncPrivileges(ctx, user.Role)

	snapshot := s.snapshot(ctx, user)
	s.cacheIdentity(ctx, user.ID, snapshot)

	return snapshot, pair, nil
}

func (s *Service) syncPrivileges(ctx context.Context, role *db.Role) {
	if role == nil || role.Type != db.RoleTypeAdmin {
		return
	}

	grant := `INSERT INTO role_privileges (role_id, privilege_id)
		SELECT ?, id FROM privileges WHERE type = ? ON CONFLICT DO NOTHING`

	if err := s.db.WithContext(ctx).Exec(grant, role.ID, db.PrivilegeTypeDashboard).Error; err != nil {
		slog.WarnContext(ctx, "role privilege grant failed", "role_id", role.ID, "error", err)
		return
	}

	var names []string

	query := s.db.WithContext(ctx).
		Model(&db.Privilege{}).
		Select("privileges.name").
		Joins("JOIN role_privileges rp ON rp.privilege_id = privileges.id").
		Where("rp.role_id = ?", role.ID)

	if err := query.Scan(&names).Error; err != nil {
		slog.WarnContext(ctx, "role privilege lookup failed", "role_id", role.ID, "error", err)
		return
	}

	if err := s.store.CachePrivileges(ctx, role.ID, names, config.PrivCacheTTL); err != nil {
		slog.WarnContext(ctx, "role privilege cache refresh failed", "role_id", role.ID, "error", err)
	}
}

func (s *Service) issue(ctx context.Context, user db.DashboardUser) (TokenPair, error) {
	secret, err := s.TenantSecret(ctx, user.CustomerID)
	if err != nil {
		return TokenPair{}, err
	}

	version, err := s.store.Version(ctx, user.ID)
	if err != nil {
		return TokenPair{}, ErrUnavailable
	}

	subject := Subject{
		UserID:     user.ID,
		CustomerID: user.CustomerID,
		RoleID:     user.RoleID,
		Version:    version,
	}

	access, err := Issue(s.cfg, secret, subject, TypeAccess)
	if err != nil {
		return TokenPair{}, err
	}

	refresh, err := Issue(s.cfg, secret, subject, TypeRefresh)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{Access: access, Refresh: refresh}, nil
}

func (s *Service) StartRegistration(ctx context.Context, email string) (time.Duration, error) {
	key := Normalize(email)

	if err := s.ensureEmailAvailable(ctx, key); err != nil {
		return 0, err
	}

	if err := s.sendVerification(ctx, key); err != nil {
		return 0, err
	}

	return config.CodeTTL, nil
}

func (s *Service) ResendCode(ctx context.Context, email string) (time.Duration, error) {
	key := Normalize(email)

	if _, err := s.store.Signup(ctx, key); err != nil {
		return 0, err
	}

	if err := s.sendVerification(ctx, key); err != nil {
		return 0, err
	}

	return config.CodeTTL, nil
}

func (s *Service) sendVerification(ctx context.Context, key string) error {
	code, err := GenerateCode()
	if err != nil {
		return err
	}

	pending := PendingSignup{
		Email:       key,
		Code:        code,
		CodeExpires: time.Now().Add(config.CodeTTL).Unix(),
	}

	if err := s.store.PutSignup(ctx, key, pending, config.SignupTTL); err != nil {
		return ErrUnavailable
	}

	err = s.notifier.Send(ctx, notification.Email{
		CustomerID: notification.PlatformCustomerID,
		Template:   notification.TemplateEmailVerification,
		To:         []string{key},
		Vars: map[string]string{
			"name":           key,
			"otp":            code,
			"expiry_minutes": strconv.Itoa(int(config.CodeTTL.Minutes())),
		},
	})
	if err != nil {
		slog.ErrorContext(ctx, "verification email failed", "to", key, "error", err.Error())

		if errors.Is(err, notification.ErrTemplateNotFound) {
			return ErrTemplateMissing
		}

		return ErrEmailSendFailed
	}

	return nil
}

func (s *Service) VerifyEmail(ctx context.Context, email, code string) (string, time.Duration, error) {
	key := Normalize(email)

	pending, err := s.store.Signup(ctx, key)
	if err != nil {
		return "", 0, err
	}

	remaining, err := s.store.SignupAttempt(ctx, key, config.CodeMaxAttempt, config.SignupTTL)
	if err != nil {
		return "", 0, ErrUnavailable
	}
	if remaining < 0 {
		s.store.DropSignup(ctx, key)
		return "", 0, ErrCodeExhausted
	}

	if time.Now().Unix() > pending.CodeExpires {
		return "", 0, ErrInvalidCode
	}

	if !EqualCode(pending.Code, code) {
		return "", 0, ErrInvalidCode
	}

	token, err := NewToken()
	if err != nil {
		return "", 0, err
	}

	if err := s.store.PutRegistrationToken(ctx, HashToken(token), key, config.SignupTTL); err != nil {
		return "", 0, ErrUnavailable
	}

	s.store.DropSignup(ctx, key)

	return token, config.SignupTTL, nil
}

func (s *Service) CompleteRegistration(ctx context.Context, req CompleteRequest) (IdentitySnapshot, TokenPair, error) {
	email, err := s.store.TakeRegistrationToken(ctx, HashToken(req.RegistrationToken))
	if err != nil {
		return IdentitySnapshot{}, TokenPair{}, ErrEmailNotVerified
	}

	if err := s.ensureAvailable(ctx, email, req.OrgName); err != nil {
		return IdentitySnapshot{}, TokenPair{}, err
	}

	salt, err := NewSalt()
	if err != nil {
		return IdentitySnapshot{}, TokenPair{}, err
	}

	hashed, err := HashPassword(s.cfg.Pepper, req.Password, salt, s.cfg.BcryptCost)
	if err != nil {
		return IdentitySnapshot{}, TokenPair{}, err
	}

	user, err := s.createTenant(ctx, req, email, hashed, salt)
	if err != nil {
		return IdentitySnapshot{}, TokenPair{}, err
	}

	pair, err := s.issue(ctx, user)
	if err != nil {
		return IdentitySnapshot{}, TokenPair{}, err
	}

	s.syncPrivileges(ctx, user.Role)

	snapshot := s.snapshot(ctx, user)
	s.cacheIdentity(ctx, user.ID, snapshot)

	s.notify(ctx, notification.Email{
		CustomerID: user.CustomerID,
		Template:   notification.TemplateWelcome,
		To:         []string{user.Email},
		Vars: map[string]string{
			"name":      user.FirstName,
			"org_name":  req.OrgName,
			"login_url": s.app.FrontendBaseURL,
		},
	})

	return snapshot, pair, nil
}

func (s *Service) createTenant(ctx context.Context, req CompleteRequest, email, hashed, salt string) (db.DashboardUser, error) {
	var user db.DashboardUser

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		customer := db.Customer{OrgName: req.OrgName}
		if err := tx.Omit("JWTSecret", "CreatedAt").Create(&customer).Error; err != nil {
			return err
		}

		var role db.Role

		err := tx.Where("customer_id = ? AND type = ?", db.SystemCustomerID, db.RoleTypeAdmin).
			Take(&role).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("shared %s role is missing for customer %d", db.RoleTypeAdmin, db.SystemCustomerID)
		}
		if err != nil {
			return err
		}

		user = db.DashboardUser{
			CustomerID:   customer.ID,
			RoleID:       &role.ID,
			FirstName:    req.FirstName,
			LastName:     req.LastName,
			Email:        email,
			PasswordHash: hashed,
			PasswordSalt: salt,
		}

		if err := tx.Omit("Customer", "Role").Create(&user).Error; err != nil {
			return err
		}

		provision := `INSERT INTO customer_branding (customer_id) VALUES (?)
			ON CONFLICT (customer_id) DO NOTHING`

		if err := tx.Exec(provision, customer.ID).Error; err != nil {
			return err
		}

		user.Customer = &customer
		user.Role = &role

		return nil
	})

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return db.DashboardUser{}, ErrEmailTaken
	}

	return user, err
}

func (s *Service) ensureEmailAvailable(ctx context.Context, email string) error {
	var count int64

	if err := s.db.WithContext(ctx).Model(&db.DashboardUser{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrEmailTaken
	}

	return nil
}

func (s *Service) ensureAvailable(ctx context.Context, email, orgName string) error {
	if err := s.ensureEmailAvailable(ctx, email); err != nil {
		return err
	}

	var count int64

	if err := s.db.WithContext(ctx).Model(&db.Customer{}).Where("lower(org_name) = lower(?)", orgName).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrOrgNameTaken
	}

	return nil
}

func (s *Service) ForgotPassword(ctx context.Context, email string) (time.Duration, error) {
	key := Normalize(email)

	user, err := s.userByEmail(ctx, key)
	if err != nil {
		return config.CodeTTL, nil
	}

	code, err := GenerateCode()
	if err != nil {
		return 0, err
	}

	state := ResetState{
		UserID:      user.ID,
		Code:        code,
		CodeExpires: time.Now().Add(config.CodeTTL).Unix(),
	}

	if err := s.store.PutReset(ctx, key, state, 15*time.Minute); err != nil {
		return 0, ErrUnavailable
	}

	err = s.notifier.Send(ctx, notification.Email{
		CustomerID: user.CustomerID,
		Template:   notification.TemplatePasswordReset,
		To:         []string{user.Email},
		Vars: map[string]string{
			"name":           user.FirstName,
			"otp":            code,
			"expiry_minutes": strconv.Itoa(int(config.CodeTTL.Minutes())),
		},
	})
	if err != nil {
		slog.ErrorContext(ctx, "reset email failed", "error", err)
		return 0, ErrEmailSendFailed
	}

	return config.CodeTTL, nil
}

func (s *Service) ResetPassword(ctx context.Context, email, code, password string) error {
	key := Normalize(email)

	state, err := s.store.Reset(ctx, key)
	if err != nil {
		return err
	}

	remaining, err := s.store.ResetAttempt(ctx, key, config.CodeMaxAttempt, 15*time.Minute)
	if err != nil {
		return ErrUnavailable
	}
	if remaining < 0 {
		s.store.DropReset(ctx, key)
		return ErrCodeExhausted
	}

	if time.Now().Unix() > state.CodeExpires {
		return ErrInvalidCode
	}

	if !EqualCode(state.Code, code) {
		return ErrInvalidCode
	}

	salt, err := NewSalt()
	if err != nil {
		return err
	}

	hashed, err := HashPassword(s.cfg.Pepper, password, salt, s.cfg.BcryptCost)
	if err != nil {
		return err
	}

	err = s.db.WithContext(ctx).Model(&db.DashboardUser{}).
		Where("id = ?", state.UserID).
		Updates(map[string]any{"password_hash": hashed, "password_salt": salt}).Error
	if err != nil {
		return err
	}

	s.store.DropReset(ctx, key)
	s.DropIdentity(ctx, state.UserID)

	if err := s.store.BumpVersion(ctx, state.UserID); err != nil {
		return ErrUnavailable
	}

	user, err := s.userByID(ctx, state.UserID)
	if err == nil {
		s.notify(ctx, notification.Email{
			CustomerID: user.CustomerID,
			Template:   notification.TemplatePasswordChanged,
			To:         []string{user.Email},
			Vars:       map[string]string{"name": user.FirstName},
		})
	}

	return nil
}

func (s *Service) Refresh(ctx context.Context, raw string) (IdentitySnapshot, TokenPair, error) {
	claims, err := Verify(ctx, raw, s.cfg.Issuer, TypeRefresh, s.TenantSecret)
	if err != nil {
		return IdentitySnapshot{}, TokenPair{}, err
	}

	userID := UserID(claims)

	version, err := s.store.Version(ctx, userID)
	if err != nil {
		return IdentitySnapshot{}, TokenPair{}, ErrUnavailable
	}
	if claims.Version < version {
		return IdentitySnapshot{}, TokenPair{}, ErrInvalidToken
	}

	user, err := s.userByID(ctx, userID)
	if err != nil {
		if db.IsNotFound(err) {
			return IdentitySnapshot{}, TokenPair{}, ErrInvalidToken
		}

		return IdentitySnapshot{}, TokenPair{}, ErrUnavailable
	}

	pair, err := s.issue(ctx, user)
	if err != nil {
		return IdentitySnapshot{}, TokenPair{}, err
	}

	successor, err := encodePair(pair)
	if err != nil {
		return IdentitySnapshot{}, TokenPair{}, err
	}

	outcome, payload, err := s.store.Rotate(ctx, claims.ID, successor,
		time.Until(claims.ExpiresAt.Time), config.RotationGrace)
	if err != nil {
		return IdentitySnapshot{}, TokenPair{}, ErrUnavailable
	}

	snapshot := s.snapshot(ctx, user)
	s.cacheIdentity(ctx, user.ID, snapshot)

	switch outcome {
	case RotateOutcomeRotated:
		return snapshot, pair, nil

	case RotateOutcomeSuccessor:
		existing, err := decodePair(payload)
		if err != nil {
			return IdentitySnapshot{}, TokenPair{}, err
		}
		return snapshot, existing, nil

	default:
		slog.WarnContext(ctx, "refresh token replay detected", "user_id", userID, "customer_id", user.CustomerID)
		s.store.BumpVersion(ctx, userID)
		return IdentitySnapshot{}, TokenPair{}, ErrTokenReplayed
	}
}

func (s *Service) Logout(ctx context.Context, accessID, refreshID string, accessExp, refreshExp time.Time) error {
	if err := s.store.Blacklist(ctx, accessID, time.Until(accessExp)); err != nil {
		return ErrUnavailable
	}

	if refreshID != "" {
		if err := s.store.Blacklist(ctx, refreshID, time.Until(refreshExp)); err != nil {
			return ErrUnavailable
		}
	}

	return nil
}

func (s *Service) Identity(ctx context.Context, userID int) (IdentitySnapshot, error) {
	snapshot, err := s.store.Identity(ctx, userID)
	if err == nil {
		return snapshot, nil
	}
	if !errors.Is(err, ErrIdentityNotCached) {
		slog.WarnContext(ctx, "identity cache read failed", "user_id", userID, "error", err)
	}

	user, err := s.userByID(ctx, userID)
	if err != nil {
		return IdentitySnapshot{}, err
	}

	snapshot = s.snapshot(ctx, user)
	s.cacheIdentity(ctx, userID, snapshot)

	return snapshot, nil
}

func (s *Service) snapshot(ctx context.Context, user db.DashboardUser) IdentitySnapshot {
	snapshot := ToSnapshot(user)
	snapshot.Branding = s.brandingFor(ctx, user.CustomerID)

	return snapshot
}

func (s *Service) brandingFor(ctx context.Context, customerID int) *BrandingSnapshot {
	if s.branding == nil {
		return nil
	}

	branding, err := s.branding.Snapshot(ctx, customerID)
	if err != nil {
		slog.WarnContext(ctx, "branding lookup failed", "customer_id", customerID, "error", err)
		return nil
	}

	return &branding
}

func (s *Service) cacheIdentity(ctx context.Context, userID int, snapshot IdentitySnapshot) {
	if err := s.store.CacheIdentity(ctx, userID, snapshot, config.IdentityTTL); err != nil {
		slog.WarnContext(ctx, "identity cache write failed", "user_id", userID, "error", err)
	}
}

func (s *Service) DropIdentity(ctx context.Context, userID int) {
	if err := s.store.DropIdentity(ctx, userID); err != nil {
		slog.WarnContext(ctx, "identity cache invalidation failed", "user_id", userID, "error", err)
	}
}

func (s *Service) notify(ctx context.Context, email notification.Email) {
	if err := s.notifier.Send(ctx, email); err != nil {
		slog.WarnContext(ctx, "notification failed", "template", email.Template, "error", err)
	}
}

type pairPayload struct {
	AccessToken    string    `json:"at"`
	AccessID       string    `json:"ai"`
	AccessExpires  time.Time `json:"ae"`
	RefreshToken   string    `json:"rt"`
	RefreshID      string    `json:"ri"`
	RefreshExpires time.Time `json:"re"`
}

func encodePair(pair TokenPair) (string, error) {
	raw, err := json.Marshal(pairPayload{
		AccessToken:    pair.Access.Token,
		AccessID:       pair.Access.ID,
		AccessExpires:  pair.Access.ExpiresAt,
		RefreshToken:   pair.Refresh.Token,
		RefreshID:      pair.Refresh.ID,
		RefreshExpires: pair.Refresh.ExpiresAt,
	})

	return string(raw), err
}

func decodePair(raw string) (TokenPair, error) {
	var payload pairPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		Access:  Issued{Token: payload.AccessToken, ID: payload.AccessID, ExpiresAt: payload.AccessExpires},
		Refresh: Issued{Token: payload.RefreshToken, ID: payload.RefreshID, ExpiresAt: payload.RefreshExpires},
	}, nil
}

func ToSnapshot(user db.DashboardUser) IdentitySnapshot {
	snapshot := IdentitySnapshot{
		User: IdentityUser{
			ID:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		},
	}

	if user.Customer != nil {
		snapshot.Customer = IdentityCustomer{ID: user.Customer.ID, OrgName: user.Customer.OrgName}
	}

	if user.Role != nil {
		snapshot.User.Role = user.Role.Name
	}

	return snapshot
}
