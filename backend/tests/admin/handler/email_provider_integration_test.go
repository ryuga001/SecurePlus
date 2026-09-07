//go:build integration

package handler_test

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	providerrepo "dpdp-backend/internal/admin/repositories/emailprovider"
	providersvc "dpdp-backend/internal/admin/services/emailprovider"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/config"
	"dpdp-backend/internal/db"
)

func (h harness) configuration(t *testing.T, customerID int, name, domain string) db.EmailProviderConfiguration {
	t.Helper()

	row, err := h.providers.Create(context.Background(), customerID, providersvc.ConfigurationInput{
		Name:     name,
		Domain:   domain,
		Provider: utils.ProviderGmail,
	})
	if err != nil {
		t.Fatalf("configuration %q create failed: %v", domain, err)
	}

	return row
}

func (h harness) domainKey(t *testing.T, domain string) (providerrepo.AuthorizedDomain, bool) {
	t.Helper()

	raw, err := h.rdb.Get(context.Background(), providerrepo.AuthorizedDomainKey(domain)).Bytes()
	if errors.Is(err, redis.Nil) {
		return providerrepo.AuthorizedDomain{}, false
	}
	if err != nil {
		t.Fatalf("redis get failed: %v", err)
	}

	var value providerrepo.AuthorizedDomain
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("redis payload is not valid json: %v", err)
	}

	return value, true
}

func TestIntegrationConfigurationCreateNormalises(t *testing.T) {
	h := setup(t)

	row := h.configuration(t, h.tenant.ID, "  Corporate   outbound ", "  HTTPS://WWW.Example.COM.  ")

	if row.Name != "Corporate outbound" || row.Domain != "example.com" {
		t.Fatalf("row = %+v", row)
	}

	value, ok := h.domainKey(t, "example.com")
	if !ok || value.ConfigID != row.ID || value.CustomerID != h.tenant.ID {
		t.Fatalf("authorized domain payload = %+v (present %v)", value, ok)
	}

	ttl, err := h.rdb.PTTL(context.Background(), providerrepo.AuthorizedDomainKey("example.com")).Result()
	if err != nil {
		t.Fatalf("pttl failed: %v", err)
	}
	if ttl != -1 {
		t.Fatalf("key must not expire, pttl = %v", ttl)
	}
}

func TestIntegrationConfigurationDomainIsGloballyUnique(t *testing.T) {
	h := setup(t)
	original := h.configuration(t, h.tenant.ID, "First", "example.com")

	ctx := context.Background()

	for _, candidate := range []string{"EXAMPLE.com", "  Example.COM  ", "www.example.com."} {
		_, err := h.providers.Create(ctx, h.tenant.ID, providersvc.ConfigurationInput{
			Name:     "Second",
			Domain:   candidate,
			Provider: utils.ProviderGmail,
		})
		if !errors.Is(err, utils.ErrDomainTaken) {
			t.Fatalf("%q must collide, error = %v", candidate, err)
		}
	}

	_, err := h.providers.Create(ctx, h.other.ID, providersvc.ConfigurationInput{
		Name:     "Theirs",
		Domain:   "example.com",
		Provider: utils.ProviderGmail,
	})
	if !errors.Is(err, utils.ErrDomainTaken) {
		t.Fatalf("cross-tenant duplicate = %v, want ErrDomainTaken", err)
	}

	value, ok := h.domainKey(t, "example.com")
	if !ok || value.ConfigID != original.ID {
		t.Fatalf("rejected duplicates changed the key: %+v", value)
	}
}

func TestIntegrationConfigurationRejectsInvalidInput(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	_, err := h.providers.Create(ctx, h.tenant.ID, providersvc.ConfigurationInput{Name: "x", Domain: "nope", Provider: utils.ProviderGmail})
	if !errors.Is(err, utils.ErrInvalidDomain) {
		t.Fatalf("error = %v, want ErrInvalidDomain", err)
	}

	_, err = h.providers.Create(ctx, h.tenant.ID, providersvc.ConfigurationInput{Name: "x", Domain: "example.com", Provider: "smtp"})
	if !errors.Is(err, utils.ErrInvalidProvider) {
		t.Fatalf("error = %v, want ErrInvalidProvider", err)
	}
}

func TestIntegrationConfigurationUpdateRekeysRedis(t *testing.T) {
	h := setup(t)
	row := h.configuration(t, h.tenant.ID, "Corporate", "example.com")

	updated, err := h.providers.Update(context.Background(), h.tenant.ID, row.ID, providersvc.ConfigurationInput{
		Name:     "Corporate",
		Domain:   "mail.example.com",
		Provider: utils.ProviderOutlook365,
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Domain != "mail.example.com" || updated.Provider != utils.ProviderOutlook365 {
		t.Fatalf("row = %+v", updated)
	}

	if _, ok := h.domainKey(t, "example.com"); ok {
		t.Fatal("old domain key must be evicted")
	}
	if value, ok := h.domainKey(t, "mail.example.com"); !ok || value.ConfigID != row.ID {
		t.Fatalf("new key missing or wrong: %+v", value)
	}
}

func TestIntegrationConfigurationUpdateWithoutDomainChangeKeepsOneKey(t *testing.T) {
	h := setup(t)
	row := h.configuration(t, h.tenant.ID, "Corporate", "example.com")

	if _, err := h.providers.Update(context.Background(), h.tenant.ID, row.ID, providersvc.ConfigurationInput{
		Name:     "Renamed",
		Domain:   "EXAMPLE.com",
		Provider: utils.ProviderGmail,
	}); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	keys, err := h.rdb.Keys(context.Background(), "authorized_domain:*").Result()
	if err != nil {
		t.Fatalf("keys failed: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected one key, got %v", keys)
	}
}

func TestIntegrationConfigurationDeleteEvictsOnlyItsKey(t *testing.T) {
	h := setup(t)
	target := h.configuration(t, h.tenant.ID, "First", "example.com")
	h.configuration(t, h.tenant.ID, "Second", "example.net")

	if err := h.providers.Delete(context.Background(), h.tenant.ID, target.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if _, ok := h.domainKey(t, "example.com"); ok {
		t.Fatal("deleted configuration key must be gone")
	}
	if _, ok := h.domainKey(t, "example.net"); !ok {
		t.Fatal("sibling key must survive")
	}

	if err := h.providers.Delete(context.Background(), h.tenant.ID, target.ID); !errors.Is(err, utils.ErrConfigurationNotFound) {
		t.Fatalf("second delete = %v", err)
	}
}

func TestIntegrationConfigurationTenantIsolation(t *testing.T) {
	h := setup(t)
	row := h.configuration(t, h.tenant.ID, "Corporate", "example.com")
	ctx := context.Background()

	if _, err := h.providers.Get(ctx, h.other.ID, row.ID); !errors.Is(err, utils.ErrConfigurationNotFound) {
		t.Fatalf("get across tenants = %v", err)
	}
	if _, err := h.providers.GenerateDKIM(ctx, h.other.ID, row.ID); !errors.Is(err, utils.ErrConfigurationNotFound) {
		t.Fatalf("dkim across tenants = %v", err)
	}
	if _, _, err := h.providers.GenerateAccessToken(ctx, h.other.ID, row.ID); !errors.Is(err, utils.ErrConfigurationNotFound) {
		t.Fatalf("token across tenants = %v", err)
	}
	if err := h.providers.Delete(ctx, h.other.ID, row.ID); !errors.Is(err, utils.ErrConfigurationNotFound) {
		t.Fatalf("delete across tenants = %v", err)
	}

	if _, ok := h.domainKey(t, "example.com"); !ok {
		t.Fatal("a rejected cross-tenant write must not touch redis")
	}
}

func TestIntegrationConfigurationListSearchIsTenantScoped(t *testing.T) {
	h := setup(t)
	h.configuration(t, h.tenant.ID, "Mine", "mine.example.com")
	h.configuration(t, h.other.ID, "Theirs", "shared-term.example.net")

	listing, err := h.providers.List(context.Background(), h.tenant.ID, providerrepo.ListParams{Search: "shared-term"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if listing.Total != 0 {
		t.Fatalf("search matched another tenant: %+v", listing.Items)
	}
}

func TestIntegrationConfigurationGenerateDKIM(t *testing.T) {
	h := setup(t)
	row := h.configuration(t, h.tenant.ID, "Corporate", "example.com")

	first, err := h.providers.GenerateDKIM(context.Background(), h.tenant.ID, row.ID)
	if err != nil {
		t.Fatalf("dkim failed: %v", err)
	}
	if first.DKIMPublicKey == nil || !strings.HasPrefix(*first.DKIMPublicKey, "v=DKIM1; k=rsa; p=") {
		t.Fatalf("dkim record = %v", first.DKIMPublicKey)
	}
	if first.DKIMPrivateKey == nil || !strings.HasPrefix(*first.DKIMPrivateKey, "-----BEGIN PRIVATE KEY-----") {
		t.Fatalf("dkim private key = %v", first.DKIMPrivateKey)
	}

	encoded := strings.TrimPrefix(*first.DKIMPublicKey, "v=DKIM1; k=rsa; p=")

	der, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("public key is not base64: %v", err)
	}
	if _, err := x509.ParsePKIXPublicKey(der); err != nil {
		t.Fatalf("public key does not parse: %v", err)
	}

	block, _ := pem.Decode([]byte(*first.DKIMPrivateKey))
	if block == nil {
		t.Fatal("private key is not valid pem")
	}
	if _, err := x509.ParsePKCS8PrivateKey(block.Bytes); err != nil {
		t.Fatalf("private key does not parse: %v", err)
	}

	var stored db.EmailProviderConfiguration
	if err := h.database.Where("id = ?", row.ID).Take(&stored).Error; err != nil {
		t.Fatalf("row lookup failed: %v", err)
	}
	if stored.DKIMPrivateKey == nil || *stored.DKIMPrivateKey != *first.DKIMPrivateKey {
		t.Fatal("private key was not persisted")
	}

	fetched, err := h.providers.Get(context.Background(), h.tenant.ID, row.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if fetched.DKIMPrivateKey != nil {
		t.Fatal("read paths must not select the dkim private key")
	}

	second, err := h.providers.GenerateDKIM(context.Background(), h.tenant.ID, row.ID)
	if err != nil {
		t.Fatalf("dkim regeneration failed: %v", err)
	}
	if *second.DKIMPublicKey == *first.DKIMPublicKey || *second.DKIMPrivateKey == *first.DKIMPrivateKey {
		t.Fatal("regeneration must produce a new key pair")
	}
}

func TestIntegrationConfigurationAccessTokenStoresOnlyHash(t *testing.T) {
	h := setup(t)
	row := h.configuration(t, h.tenant.ID, "Corporate", "example.com")

	token, expiresAt, err := h.providers.GenerateAccessToken(context.Background(), h.tenant.ID, row.ID)
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	var stored db.EmailProviderConfiguration
	if err := h.database.Where("id = ?", row.ID).Take(&stored).Error; err != nil {
		t.Fatalf("row lookup failed: %v", err)
	}

	if stored.AccessTokenHash == nil || *stored.AccessTokenHash == token {
		t.Fatal("plaintext token must never be stored")
	}
	if *stored.AccessTokenHash != auth.HashToken(token) {
		t.Fatal("stored value is not the hash of the issued token")
	}

	remaining := time.Until(expiresAt)
	if remaining < config.ProviderTokenTTL-time.Hour || remaining > config.ProviderTokenTTL {
		t.Fatalf("expiry %v is not one year out", remaining)
	}

	var customer db.Customer
	if err := h.database.Where("id = ?", h.tenant.ID).Take(&customer).Error; err != nil {
		t.Fatalf("secret lookup failed: %v", err)
	}

	claims, err := providersvc.ParseAccessToken(token, customer.JWTSecret, "dpdp")
	if err != nil {
		t.Fatalf("issued token does not verify: %v", err)
	}
	if claims.Domain != "example.com" || claims.ConfigID != row.ID || claims.CustomerID != h.tenant.ID {
		t.Fatalf("claims = %+v", claims)
	}

	fetched, err := h.providers.Get(context.Background(), h.tenant.ID, row.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if fetched.AccessTokenHash != nil {
		t.Fatal("read paths must not select the token hash")
	}
}

func TestIntegrationHTTPConfigurationDuplicateReturns409(t *testing.T) {
	h := setup(t)
	router := h.router(t)

	h.configuration(t, h.tenant.ID, "First", "example.com")

	recorder := do(t, router, http.MethodPost, "/api/v1/admin/email/configurations",
		`{"name":"Second","domain":"EXAMPLE.com","provider":"gmail"}`)

	expectStatus(t, recorder, http.StatusConflict)

	payload := decodeError(t, recorder)
	if payload.Error != utils.CodeDomainTaken || !strings.Contains(payload.Message, "example.com") {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestIntegrationHTTPConfigurationListAndDelete(t *testing.T) {
	h := setup(t)
	router := h.router(t)

	row := h.configuration(t, h.tenant.ID, "Corporate", "example.com")

	listed := do(t, router, http.MethodGet, "/api/v1/admin/email/configurations?search=corp&page=1&page_size=10", "")
	expectStatus(t, listed, http.StatusOK)

	if !strings.Contains(listed.Body.String(), `"items"`) {
		t.Fatalf("list body = %s", listed.Body.String())
	}
	if strings.Contains(listed.Body.String(), "access_token_hash") {
		t.Fatal("list response exposes the token hash")
	}

	missing := do(t, router, http.MethodGet, "/api/v1/admin/email/configurations/999999", "")
	expectStatus(t, missing, http.StatusNotFound)

	removed := do(t, router, http.MethodDelete, "/api/v1/admin/email/configurations/"+strconv.Itoa(row.ID), "")
	expectStatus(t, removed, http.StatusNoContent)
}

func TestIntegrationMigrationSeedsProviderPrivileges(t *testing.T) {
	h := setup(t)

	var count int64
	err := h.database.Table("privileges").
		Where("name LIKE ? AND type = ?", "admin.email.provider.%", "DASHBOARD").
		Count(&count).Error
	if err != nil {
		t.Fatalf("privilege lookup failed: %v", err)
	}

	if count != 4 {
		t.Fatalf("provider privileges = %d, want 4", count)
	}
}
