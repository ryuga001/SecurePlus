package services_test

import (
	"errors"
	"testing"
	"time"

	providersvc "dpdp-backend/internal/admin/services/emailprovider"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/config"
	"dpdp-backend/internal/db"
)

func configuration() db.EmailProviderConfiguration {
	return db.EmailProviderConfiguration{ID: 7, CustomerID: 3, Domain: "example.com", Name: "Corporate"}
}

func TestIssueAccessTokenCarriesDomainClaim(t *testing.T) {
	secret := "tenant-secret"

	token, expiresAt, err := providersvc.IssueAccessToken(secret, "dpdp", configuration(), config.ProviderTokenTTL)
	if err != nil {
		t.Fatalf("IssueAccessToken returned %v", err)
	}

	claims, err := providersvc.ParseAccessToken(token, secret, "dpdp")
	if err != nil {
		t.Fatalf("ParseAccessToken returned %v", err)
	}

	if claims.Domain != "example.com" || claims.ConfigID != 7 || claims.CustomerID != 3 {
		t.Fatalf("unexpected claims %+v", claims)
	}
	if claims.Typ != providersvc.TokenType {
		t.Fatalf("token type = %q", claims.Typ)
	}

	remaining := time.Until(expiresAt)
	if remaining < config.ProviderTokenTTL-time.Minute || remaining > config.ProviderTokenTTL {
		t.Fatalf("expiry %v is not one year out", remaining)
	}
}

func TestParseAccessTokenRejectsOtherSecret(t *testing.T) {
	token, _, err := providersvc.IssueAccessToken("tenant-secret", "dpdp", configuration(), config.ProviderTokenTTL)
	if err != nil {
		t.Fatalf("IssueAccessToken returned %v", err)
	}

	if _, err := providersvc.ParseAccessToken(token, "another-tenant-secret", "dpdp"); err == nil {
		t.Fatal("a token signed for one tenant must not verify for another")
	}
}

func TestParseAccessTokenRejectsOtherIssuer(t *testing.T) {
	token, _, err := providersvc.IssueAccessToken("tenant-secret", "dpdp", configuration(), config.ProviderTokenTTL)
	if err != nil {
		t.Fatalf("IssueAccessToken returned %v", err)
	}

	if _, err := providersvc.ParseAccessToken(token, "tenant-secret", "someone-else"); err == nil {
		t.Fatal("a token must not verify for another issuer")
	}
}

func TestIssueAccessTokenRejectsInvalidDomain(t *testing.T) {
	invalid := configuration()
	invalid.Domain = "not a domain"

	_, _, err := providersvc.IssueAccessToken("secret", "dpdp", invalid, config.ProviderTokenTTL)
	if !errors.Is(err, utils.ErrInvalidDomain) {
		t.Fatalf("expected ErrInvalidDomain, got %v", err)
	}
}
