package delivery

import (
	"context"
	"errors"

	providerrepo "dpdp-backend/internal/admin/repositories/emailprovider"
	"dpdp-backend/internal/db"
)

type providerStore interface {
	FindByDomain(ctx context.Context, domain string) (db.EmailProviderConfiguration, error)
	SigningConfig(ctx context.Context, customerID, id int) (db.EmailProviderConfiguration, error)
}

type domainRegistry interface {
	Domain(ctx context.Context, domain string) (providerrepo.AuthorizedDomain, error)
	SyncDomain(ctx context.Context, previous, domain string, value providerrepo.AuthorizedDomain) error
}

type DomainLookup struct {
	registry domainRegistry
}

func NewDomainLookup(registry domainRegistry) *DomainLookup {
	return &DomainLookup{registry: registry}
}

func (l *DomainLookup) Domain(ctx context.Context, domain string) (int, int, error) {
	value, err := l.registry.Domain(ctx, domain)
	if err != nil {
		return 0, 0, err
	}

	return value.CustomerID, value.ConfigID, nil
}

func (l *DomainLookup) Remember(ctx context.Context, domain string, customerID, configID int) error {
	return l.registry.SyncDomain(ctx, "", domain, providerrepo.AuthorizedDomain{
		ConfigID:   configID,
		CustomerID: customerID,
	})
}

type ConfigurationStore struct {
	repo    providerStore
	unknown error
}

func NewConfigurationStore(repo providerStore, unknown error) *ConfigurationStore {
	return &ConfigurationStore{repo: repo, unknown: unknown}
}

func (s *ConfigurationStore) FindByDomain(ctx context.Context, domain string) (int, int, error) {
	row, err := s.repo.FindByDomain(ctx, domain)
	if err != nil {
		if db.IsNotFound(err) {
			return 0, 0, s.unknown
		}

		return 0, 0, err
	}

	return row.CustomerID, row.ID, nil
}

func (s *ConfigurationStore) SigningConfig(ctx context.Context, customerID, configID int) (TenantConfig, error) {
	row, err := s.repo.SigningConfig(ctx, customerID, configID)
	if err != nil {
		if db.IsNotFound(err) {
			return TenantConfig{}, s.unknown
		}

		return TenantConfig{}, err
	}

	cfg := TenantConfig{
		CustomerID:   row.CustomerID,
		ConfigID:     row.ID,
		Domain:       row.Domain,
		DKIMSelector: DefaultDKIMSelector,
	}

	if row.DKIMPrivateKey != nil {
		cfg.DKIMPrivateKey = *row.DKIMPrivateKey
	}

	if cfg.DKIMPrivateKey == "" {
		return TenantConfig{}, errors.New("dkim private key is not configured for this domain")
	}

	return cfg, nil
}
