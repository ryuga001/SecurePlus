package receiver

import (
	"context"
	deliveryutils "dpdp-backend/internal/delivery/utils"
	"errors"
	"log/slog"
)

type DomainCache interface {
	Domain(ctx context.Context, domain string) (int, int, error)
	Remember(ctx context.Context, domain string, customerID, configID int) error
}

type DomainStore interface {
	FindByDomain(ctx context.Context, domain string) (int, int, error)
}

type Authorizer struct {
	cache DomainCache
	store DomainStore
}

func NewAuthorizer(cache DomainCache, store DomainStore) *Authorizer {
	return &Authorizer{cache: cache, store: store}
}

func (a *Authorizer) Authorize(ctx context.Context, domain string) (Authorization, error) {
	customerID, configID, err := a.cache.Domain(ctx, domain)
	if err == nil {
		return Authorization{CustomerID: customerID, ConfigID: configID, Domain: domain}, nil
	}

	customerID, configID, storeErr := a.store.FindByDomain(ctx, domain)
	if storeErr != nil {
		if errors.Is(storeErr, deliveryutils.ErrDomainUnknown) {
			return Authorization{}, deliveryutils.ErrDomainUnknown
		}

		return Authorization{}, storeErr
	}

	if cacheErr := a.cache.Remember(ctx, domain, customerID, configID); cacheErr != nil {
		slog.WarnContext(ctx, "authorized domain cache refill failed", "domain", domain, "error", cacheErr)
	}

	return Authorization{CustomerID: customerID, ConfigID: configID, Domain: domain}, nil
}
