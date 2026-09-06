package email

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/config"
	"dpdp-backend/internal/db"
)

type Service struct {
	db    *gorm.DB
	redis DomainRegistry
	cfg   config.Auth
}

func NewService(database *gorm.DB, redis DomainRegistry, cfg config.Auth) *Service {
	return &Service{db: database, redis: redis, cfg: cfg}
}

type ListParams struct {
	Search   string
	Page     int
	PageSize int
}

type Listing struct {
	Items    []db.EmailProviderConfiguration
	Page     int
	PageSize int
	Total    int64
}

type Input struct {
	Name     string
	Domain   string
	Provider string
}

var configColumns = []string{
	"id", "customer_id", "name", "domain", "provider",
	"dkim_public_key", "access_token_expires_at", "created_at", "updated_at",
}

func (s *Service) List(ctx context.Context, customerID int, params ListParams) (Listing, error) {
	page, pageSize := normalizePaging(params.Page, params.PageSize)

	query := s.db.WithContext(ctx).
		Model(&db.EmailProviderConfiguration{}).
		Scopes(db.TenantScope(customerID))

	if search := strings.TrimSpace(params.Search); search != "" {
		pattern := "%" + escapeLike(search) + "%"
		query = query.Where(`(name ILIKE ? ESCAPE '\' OR domain ILIKE ? ESCAPE '\')`, pattern, pattern)
	}

	query = query.Session(&gorm.Session{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return Listing{}, err
	}

	rows := make([]db.EmailProviderConfiguration, 0, pageSize)

	if total > int64((page-1)*pageSize) {
		err := query.Select(configColumns).
			Order("created_at DESC, id DESC").
			Limit(pageSize).
			Offset((page - 1) * pageSize).
			Find(&rows).Error
		if err != nil {
			return Listing{}, err
		}
	}

	return Listing{Items: rows, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *Service) Get(ctx context.Context, customerID, id int) (db.EmailProviderConfiguration, error) {
	return s.find(s.db.WithContext(ctx), customerID, id)
}

func (s *Service) Create(ctx context.Context, customerID int, in Input) (db.EmailProviderConfiguration, error) {
	name, domain, provider, err := normalizeInput(in)
	if err != nil {
		return db.EmailProviderConfiguration{}, err
	}

	row := db.EmailProviderConfiguration{
		CustomerID: customerID,
		Name:       name,
		Domain:     domain,
		Provider:   provider,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}

		return s.cache(ctx, "", row)
	})

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return db.EmailProviderConfiguration{}, domainTaken(domain)
	}
	if err != nil {
		return db.EmailProviderConfiguration{}, err
	}

	return row, nil
}

func (s *Service) Update(ctx context.Context, customerID, id int, in Input) (db.EmailProviderConfiguration, error) {
	name, domain, provider, err := normalizeInput(in)
	if err != nil {
		return db.EmailProviderConfiguration{}, err
	}

	var row db.EmailProviderConfiguration

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := s.find(tx.Clauses(clause.Locking{Strength: "UPDATE"}), customerID, id)
		if err != nil {
			return err
		}

		updates := map[string]any{
			"name":       name,
			"domain":     domain,
			"provider":   provider,
			"updated_at": time.Now(),
		}

		if err := tx.Model(&db.EmailProviderConfiguration{}).Where("id = ?", current.ID).Updates(updates).Error; err != nil {
			return err
		}

		row = current
		row.Name = name
		row.Domain = domain
		row.Provider = provider

		return s.cache(ctx, current.Domain, row)
	})

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return db.EmailProviderConfiguration{}, domainTaken(domain)
	}
	if err != nil {
		return db.EmailProviderConfiguration{}, err
	}

	return row, nil
}

func (s *Service) Delete(ctx context.Context, customerID, id int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := s.find(tx, customerID, id)
		if err != nil {
			return err
		}

		if err := tx.Delete(&db.EmailProviderConfiguration{}, row.ID).Error; err != nil {
			return err
		}

		return s.redis.DropDomain(ctx, row.Domain)
	})
}

func (s *Service) GenerateDKIM(ctx context.Context, customerID, id int) (db.EmailProviderConfiguration, error) {
	row, err := s.find(s.db.WithContext(ctx), customerID, id)
	if err != nil {
		return db.EmailProviderConfiguration{}, err
	}

	key, err := placeholderDKIM()
	if err != nil {
		return db.EmailProviderConfiguration{}, err
	}

	updates := map[string]any{"dkim_public_key": key, "updated_at": time.Now()}

	if err := s.db.WithContext(ctx).Model(&db.EmailProviderConfiguration{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return db.EmailProviderConfiguration{}, err
	}

	row.DKIMPublicKey = &key

	return row, nil
}

func (s *Service) GenerateAccessToken(ctx context.Context, customerID, id int) (string, time.Time, error) {
	row, err := s.find(s.db.WithContext(ctx), customerID, id)
	if err != nil {
		return "", time.Time{}, err
	}

	secret, err := s.tenantSecret(ctx, customerID)
	if err != nil {
		return "", time.Time{}, err
	}

	token, expiresAt, err := IssueAccessToken(secret, s.cfg.Issuer, row, config.ProviderTokenTTL)
	if err != nil {
		return "", time.Time{}, err
	}

	updates := map[string]any{
		"access_token_hash":       auth.HashToken(token),
		"access_token_expires_at": expiresAt,
		"updated_at":              time.Now(),
	}

	if err := s.db.WithContext(ctx).Model(&db.EmailProviderConfiguration{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}

func (s *Service) find(tx *gorm.DB, customerID, id int) (db.EmailProviderConfiguration, error) {
	var row db.EmailProviderConfiguration

	err := tx.Model(&db.EmailProviderConfiguration{}).
		Select(configColumns).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Take(&row).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.EmailProviderConfiguration{}, ErrNotFound
	}

	return row, err
}

func (s *Service) cache(ctx context.Context, previous string, row db.EmailProviderConfiguration) error {
	return s.redis.SyncDomain(ctx, previous, row.Domain, AuthorizedDomain{
		ConfigID:   row.ID,
		CustomerID: row.CustomerID,
	})
}

func (s *Service) tenantSecret(ctx context.Context, customerID int) (string, error) {
	var customer db.Customer

	err := s.db.WithContext(ctx).
		Model(&db.Customer{}).
		Select("id", "jwt_secret").
		Where("id = ?", customerID).
		Take(&customer).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrNotFound
	}

	return customer.JWTSecret, err
}

func normalizeInput(in Input) (string, string, string, error) {
	domain := NormalizeDomain(in.Domain)
	if !ValidDomain(domain) {
		return "", "", "", ErrInvalidDomain
	}

	provider := NormalizeProvider(in.Provider)
	if !ValidProvider(provider) {
		return "", "", "", ErrInvalidProvider
	}

	return NormalizeName(in.Name), domain, provider, nil
}

func placeholderDKIM() (string, error) {
	value, err := auth.NewToken()
	if err != nil {
		return "", err
	}

	return "v=DKIM1; k=rsa; p=" + value, nil
}
