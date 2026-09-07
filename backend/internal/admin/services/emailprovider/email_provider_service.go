package emailprovider

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	repo "dpdp-backend/internal/admin/repositories/emailprovider"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/config"
	"dpdp-backend/internal/db"
)

type ConfigurationListing struct {
	Items    []db.EmailProviderConfiguration
	Page     int
	PageSize int
	Total    int64
}

type EmailProviderService struct {
	db    *gorm.DB
	repo  *repo.EmailProviderRepository
	redis repo.DomainRegistry
	cfg   config.Auth
}

func NewEmailProviderService(
	database *gorm.DB,
	repo *repo.EmailProviderRepository,
	redis repo.DomainRegistry,
	cfg config.Auth,
) *EmailProviderService {
	return &EmailProviderService{db: database, repo: repo, redis: redis, cfg: cfg}
}

func (s *EmailProviderService) List(ctx context.Context, customerID int, params repo.ListParams) (ConfigurationListing, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	rows, total, err := s.repo.List(ctx, customerID, params)
	if err != nil {
		return ConfigurationListing{}, err
	}

	return ConfigurationListing{Items: rows, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *EmailProviderService) Get(ctx context.Context, customerID, id int) (db.EmailProviderConfiguration, error) {
	row, err := s.repo.Find(ctx, customerID, id)
	if err != nil {
		return db.EmailProviderConfiguration{}, configurationError(err, "")
	}

	return row, nil
}

func (s *EmailProviderService) Create(ctx context.Context, customerID int, in ConfigurationInput) (db.EmailProviderConfiguration, error) {
	input, err := NormalizeConfigurationInput(in)
	if err != nil {
		return db.EmailProviderConfiguration{}, err
	}

	row := db.EmailProviderConfiguration{
		CustomerID: customerID,
		Name:       input.Name,
		Domain:     input.Domain,
		Provider:   input.Provider,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.WithTx(tx).Insert(ctx, &row); err != nil {
			return err
		}

		return s.cache(ctx, "", row)
	})

	if err != nil {
		return db.EmailProviderConfiguration{}, configurationError(err, input.Domain)
	}

	return row, nil
}

func (s *EmailProviderService) Update(ctx context.Context, customerID, id int, in ConfigurationInput) (db.EmailProviderConfiguration, error) {
	input, err := NormalizeConfigurationInput(in)
	if err != nil {
		return db.EmailProviderConfiguration{}, err
	}

	var row db.EmailProviderConfiguration

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)

		current, err := repo.Lock(ctx, customerID, id)
		if err != nil {
			return err
		}

		updates := map[string]any{
			"name":       input.Name,
			"domain":     input.Domain,
			"provider":   input.Provider,
			"updated_at": time.Now(),
		}

		if _, err := repo.Update(ctx, customerID, id, updates); err != nil {
			return err
		}

		row = current
		row.Name = input.Name
		row.Domain = input.Domain
		row.Provider = input.Provider

		return s.cache(ctx, current.Domain, row)
	})

	if err != nil {
		return db.EmailProviderConfiguration{}, configurationError(err, input.Domain)
	}

	return row, nil
}

func (s *EmailProviderService) Delete(ctx context.Context, customerID, id int) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)

		row, err := repo.Find(ctx, customerID, id)
		if err != nil {
			return err
		}

		if _, err := repo.Delete(ctx, customerID, id); err != nil {
			return err
		}

		return s.redis.DropDomain(ctx, row.Domain)
	})

	if err != nil {
		return configurationError(err, "")
	}

	return nil
}

func (s *EmailProviderService) GenerateDKIM(ctx context.Context, customerID, id int) (db.EmailProviderConfiguration, error) {
	row, err := s.repo.Find(ctx, customerID, id)
	if err != nil {
		return db.EmailProviderConfiguration{}, configurationError(err, "")
	}

	key, err := placeholderDKIM()
	if err != nil {
		return db.EmailProviderConfiguration{}, err
	}

	updates := map[string]any{"dkim_public_key": key, "updated_at": time.Now()}

	if _, err := s.repo.Update(ctx, customerID, id, updates); err != nil {
		return db.EmailProviderConfiguration{}, err
	}

	row.DKIMPublicKey = &key

	return row, nil
}

func (s *EmailProviderService) GenerateAccessToken(ctx context.Context, customerID, id int) (string, time.Time, error) {
	row, err := s.repo.Find(ctx, customerID, id)
	if err != nil {
		return "", time.Time{}, configurationError(err, "")
	}

	secret, err := s.repo.TenantSecret(ctx, customerID)
	if err != nil {
		return "", time.Time{}, configurationError(err, "")
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

	if _, err := s.repo.Update(ctx, customerID, id, updates); err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}

func (s *EmailProviderService) cache(ctx context.Context, previous string, row db.EmailProviderConfiguration) error {
	return s.redis.SyncDomain(ctx, previous, row.Domain, repo.AuthorizedDomain{
		ConfigID:   row.ID,
		CustomerID: row.CustomerID,
	})
}

func placeholderDKIM() (string, error) {
	value, err := auth.NewToken()
	if err != nil {
		return "", err
	}

	return "v=DKIM1; k=rsa; p=" + value, nil
}

func configurationError(err error, domain string) error {
	switch {
	case db.IsNotFound(err):
		return utils.ErrConfigurationNotFound
	case db.IsDuplicate(err):
		return utils.Taken(utils.ErrDomainTaken, domain)
	}

	return err
}

const (
	minDomainLength = 4
	maxDomainLength = 253
)

var domainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*\.[a-z]{2,63}$`)

type ConfigurationInput struct {
	Name     string
	Domain   string
	Provider string
}

func NormalizeDomain(domain string) string {
	value := strings.ToLower(strings.TrimSpace(domain))
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimPrefix(value, "www.")
	value = strings.TrimSuffix(value, ".")

	return value
}

func NormalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func ValidDomain(domain string) bool {
	if len(domain) < minDomainLength || len(domain) > maxDomainLength {
		return false
	}

	return domainPattern.MatchString(domain)
}

func ValidProvider(provider string) bool {
	return provider == utils.ProviderOutlook365 || provider == utils.ProviderGmail
}

func NormalizeConfigurationInput(in ConfigurationInput) (ConfigurationInput, error) {
	domain := NormalizeDomain(in.Domain)
	if !ValidDomain(domain) {
		return ConfigurationInput{}, utils.ErrInvalidDomain
	}

	provider := NormalizeProvider(in.Provider)
	if !ValidProvider(provider) {
		return ConfigurationInput{}, utils.ErrInvalidProvider
	}

	return ConfigurationInput{
		Name:     utils.NormalizeName(in.Name),
		Domain:   domain,
		Provider: provider,
	}, nil
}

const TokenType = "provider_access"

type AccessClaims struct {
	jwt.RegisteredClaims
	CustomerID int    `json:"cid"`
	ConfigID   int    `json:"config_id"`
	Domain     string `json:"domain"`
	Typ        string `json:"typ"`
}

func IssueAccessToken(secret, issuer string, row db.EmailProviderConfiguration, ttl time.Duration) (string, time.Time, error) {
	if !ValidDomain(row.Domain) {
		return "", time.Time{}, utils.ErrInvalidDomain
	}

	id, err := auth.NewSalt()
	if err != nil {
		return "", time.Time{}, err
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	claims := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        id,
			Issuer:    issuer,
			Subject:   strconv.Itoa(row.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		CustomerID: row.CustomerID,
		ConfigID:   row.ID,
		Domain:     row.Domain,
		Typ:        TokenType,
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

func ParseAccessToken(raw, secret, issuer string) (*AccessClaims, error) {
	claims := &AccessClaims{}

	parsed, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(issuer), jwt.WithExpirationRequired())

	if err != nil || !parsed.Valid {
		return nil, errors.New(TokenType + " token is invalid")
	}
	if claims.Typ != TokenType || claims.Domain == "" {
		return nil, errors.New(TokenType + " token is invalid")
	}

	return claims, nil
}
