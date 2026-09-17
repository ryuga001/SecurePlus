package provider

import (
	"context"

	providerrepo "dpdp-backend/internal/admin/repositories/emailprovider"
	"dpdp-backend/internal/db"
	"dpdp-backend/internal/delivery"
	deliveryutils "dpdp-backend/internal/delivery/utils"
)

type DomainLookup struct {
	registry *providerrepo.RedisRepository
}

func NewDomainLookup(registry *providerrepo.RedisRepository) *DomainLookup {
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
	repo *providerrepo.EmailProviderRepository
}

func NewConfigurationStore(repo *providerrepo.EmailProviderRepository) *ConfigurationStore {
	return &ConfigurationStore{repo: repo}
}

func (s *ConfigurationStore) FindByDomain(ctx context.Context, domain string) (int, int, error) {
	row, err := s.repo.FindByDomain(ctx, domain)
	if err != nil {
		if db.IsNotFound(err) {
			return 0, 0, deliveryutils.ErrDomainUnknown
		}

		return 0, 0, err
	}

	return row.CustomerID, row.ID, nil
}

func (s *ConfigurationStore) SigningConfig(ctx context.Context, customerID, configID int) (delivery.TenantConfig, error) {
	row, err := s.repo.SigningConfig(ctx, customerID, configID)
	if err != nil {
		if db.IsNotFound(err) {
			return delivery.TenantConfig{}, deliveryutils.ErrDomainUnknown
		}

		return delivery.TenantConfig{}, err
	}

	cfg := delivery.TenantConfig{
		CustomerID:   row.CustomerID,
		ConfigID:     row.ID,
		Domain:       row.Domain,
		DKIMSelector: deliveryutils.DefaultDKIMSelector,
	}

	if row.DKIMPrivateKey != nil {
		cfg.DKIMPrivateKey = *row.DKIMPrivateKey
	}

	if cfg.DKIMPrivateKey == "" {
		return delivery.TenantConfig{}, deliveryutils.ErrPrivateKeyMissing
	}

	return cfg, nil
}
