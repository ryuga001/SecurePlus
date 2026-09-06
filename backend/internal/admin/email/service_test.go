package email

import (
	"context"
	"errors"
	"testing"
	"time"

	"dpdp-backend/internal/config"
	"dpdp-backend/internal/db"
)

type syncCall struct {
	previous string
	domain   string
	value    AuthorizedDomain
}

type fakeRegistry struct {
	syncs   []syncCall
	dropped []string
	err     error
}

func (f *fakeRegistry) SyncDomain(_ context.Context, previous, domain string, value AuthorizedDomain) error {
	f.syncs = append(f.syncs, syncCall{previous: previous, domain: domain, value: value})

	return f.err
}

func (f *fakeRegistry) DropDomain(_ context.Context, domain string) error {
	f.dropped = append(f.dropped, domain)

	return f.err
}

func row() db.EmailProviderConfiguration {
	return db.EmailProviderConfiguration{ID: 7, CustomerID: 3, Domain: "example.com", Name: "Corporate"}
}

func TestCacheSetsDomainOnCreate(t *testing.T) {
	registry := &fakeRegistry{}
	svc := NewService(nil, registry, config.Auth{})

	if err := svc.cache(context.Background(), "", row()); err != nil {
		t.Fatalf("cache returned %v", err)
	}

	if len(registry.syncs) != 1 {
		t.Fatalf("expected one sync, got %d", len(registry.syncs))
	}
	if registry.syncs[0].previous != "" || registry.syncs[0].domain != "example.com" {
		t.Fatalf("unexpected sync %+v", registry.syncs[0])
	}
	if len(registry.dropped) != 0 {
		t.Fatalf("create must not drop keys, dropped %v", registry.dropped)
	}
}

func TestCacheRekeysWhenDomainChanges(t *testing.T) {
	registry := &fakeRegistry{}
	svc := NewService(nil, registry, config.Auth{})

	if err := svc.cache(context.Background(), "old.com", row()); err != nil {
		t.Fatalf("cache returned %v", err)
	}

	if registry.syncs[0].previous != "old.com" {
		t.Fatalf("previous domain not forwarded: %+v", registry.syncs[0])
	}
}

func TestCacheCarriesConfigAndCustomer(t *testing.T) {
	registry := &fakeRegistry{}
	svc := NewService(nil, registry, config.Auth{})

	if err := svc.cache(context.Background(), "", row()); err != nil {
		t.Fatalf("cache returned %v", err)
	}

	want := AuthorizedDomain{ConfigID: 7, CustomerID: 3}
	if registry.syncs[0].value != want {
		t.Fatalf("payload = %+v, want %+v", registry.syncs[0].value, want)
	}
}

func TestCachePropagatesFailure(t *testing.T) {
	failure := errors.New("redis down")
	svc := NewService(nil, &fakeRegistry{err: failure}, config.Auth{})

	if err := svc.cache(context.Background(), "", row()); !errors.Is(err, failure) {
		t.Fatalf("cache error = %v, want %v", err, failure)
	}
}

func TestNormalizeInput(t *testing.T) {
	name, domain, provider, err := normalizeInput(Input{
		Name:     "  Corporate   outbound ",
		Domain:   "  WWW.Example.COM. ",
		Provider: " Gmail ",
	})
	if err != nil {
		t.Fatalf("normalizeInput returned %v", err)
	}

	if name != "Corporate outbound" || domain != "example.com" || provider != ProviderGmail {
		t.Fatalf("normalizeInput = (%q,%q,%q)", name, domain, provider)
	}
}

func TestNormalizeInputRejects(t *testing.T) {
	if _, _, _, err := normalizeInput(Input{Name: "x", Domain: "not a domain", Provider: ProviderGmail}); !errors.Is(err, ErrInvalidDomain) {
		t.Fatalf("expected ErrInvalidDomain, got %v", err)
	}

	if _, _, _, err := normalizeInput(Input{Name: "x", Domain: "example.com", Provider: "smtp"}); !errors.Is(err, ErrInvalidProvider) {
		t.Fatalf("expected ErrInvalidProvider, got %v", err)
	}
}

func TestIssueAccessTokenCarriesDomainClaim(t *testing.T) {
	secret := "tenant-secret"

	token, expiresAt, err := IssueAccessToken(secret, "dpdp", row(), config.ProviderTokenTTL)
	if err != nil {
		t.Fatalf("IssueAccessToken returned %v", err)
	}

	claims, err := ParseAccessToken(token, secret, "dpdp")
	if err != nil {
		t.Fatalf("ParseAccessToken returned %v", err)
	}

	if claims.Domain != "example.com" || claims.ConfigID != 7 || claims.CustomerID != 3 || claims.Typ != TokenType {
		t.Fatalf("unexpected claims %+v", claims)
	}

	remaining := time.Until(expiresAt)
	if remaining < config.ProviderTokenTTL-time.Minute || remaining > config.ProviderTokenTTL {
		t.Fatalf("expiry %v is not one year out", remaining)
	}
}

func TestParseAccessTokenRejectsOtherSecret(t *testing.T) {
	token, _, err := IssueAccessToken("tenant-secret", "dpdp", row(), config.ProviderTokenTTL)
	if err != nil {
		t.Fatalf("IssueAccessToken returned %v", err)
	}

	if _, err := ParseAccessToken(token, "another-tenant-secret", "dpdp"); err == nil {
		t.Fatal("a token signed for one tenant must not verify for another")
	}
}

func TestIssueAccessTokenRejectsInvalidDomain(t *testing.T) {
	invalid := row()
	invalid.Domain = "not a domain"

	if _, _, err := IssueAccessToken("secret", "dpdp", invalid, config.ProviderTokenTTL); !errors.Is(err, ErrInvalidDomain) {
		t.Fatalf("expected ErrInvalidDomain, got %v", err)
	}
}
