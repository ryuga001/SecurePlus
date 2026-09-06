//go:build integration

package email

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/config"
	"dpdp-backend/internal/db"
	"dpdp-backend/internal/testsupport"
)

type harness struct {
	svc      *Service
	database *gorm.DB
	rdb      *redis.Client
	tenant   db.Customer
	other    db.Customer
}

func setup(t *testing.T) harness {
	t.Helper()

	gin.SetMode(gin.TestMode)

	database := testsupport.Postgres(t)
	rdb := testsupport.Redis(t)

	testsupport.Reset(t, database, rdb)
	t.Cleanup(func() { testsupport.Reset(t, database, rdb) })

	return harness{
		svc:      NewService(database, NewRedisService(rdb), config.Auth{Issuer: "dpdp"}),
		database: database,
		rdb:      rdb,
		tenant:   testsupport.Customer(t, database),
		other:    testsupport.Customer(t, database),
	}
}

func (h harness) create(t *testing.T, customerID int, name, domain string) db.EmailProviderConfiguration {
	t.Helper()

	row, err := h.svc.Create(context.Background(), customerID, Input{
		Name:     name,
		Domain:   domain,
		Provider: ProviderGmail,
	})
	if err != nil {
		t.Fatalf("create %q failed: %v", domain, err)
	}

	return row
}

func (h harness) key(t *testing.T, domain string) (AuthorizedDomain, bool) {
	t.Helper()

	raw, err := h.rdb.Get(context.Background(), authorizedDomainKey(domain)).Bytes()
	if errors.Is(err, redis.Nil) {
		return AuthorizedDomain{}, false
	}
	if err != nil {
		t.Fatalf("redis get failed: %v", err)
	}

	var value AuthorizedDomain
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("redis payload is not valid json: %v", err)
	}

	return value, true
}

func TestIntegrationCreateNormalisesAndPersists(t *testing.T) {
	h := setup(t)

	row := h.create(t, h.tenant.ID, "  Corporate   outbound ", "  HTTPS://WWW.Example.COM.  ")

	if row.ID == 0 {
		t.Fatal("expected a generated id")
	}
	if row.Name != "Corporate outbound" {
		t.Fatalf("name = %q", row.Name)
	}
	if row.Domain != "example.com" {
		t.Fatalf("domain = %q", row.Domain)
	}
	if row.CreatedAt.IsZero() || row.UpdatedAt.IsZero() {
		t.Fatal("timestamps must be populated")
	}
}

func TestIntegrationCreateRejectsInvalidInput(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	if _, err := h.svc.Create(ctx, h.tenant.ID, Input{Name: "x", Domain: "nope", Provider: ProviderGmail}); !errors.Is(err, ErrInvalidDomain) {
		t.Fatalf("expected ErrInvalidDomain, got %v", err)
	}

	if _, err := h.svc.Create(ctx, h.tenant.ID, Input{Name: "x", Domain: "example.com", Provider: "smtp"}); !errors.Is(err, ErrInvalidProvider) {
		t.Fatalf("expected ErrInvalidProvider, got %v", err)
	}
}

func TestIntegrationDuplicateDomainSameTenant(t *testing.T) {
	h := setup(t)
	h.create(t, h.tenant.ID, "First", "example.com")

	_, err := h.svc.Create(context.Background(), h.tenant.ID, Input{Name: "Second", Domain: "example.com", Provider: ProviderGmail})
	if !errors.Is(err, ErrDomainTaken) {
		t.Fatalf("expected ErrDomainTaken, got %v", err)
	}
	if !strings.Contains(err.Error(), "example.com") {
		t.Fatalf("message must name the domain, got %q", err.Error())
	}
}

func TestIntegrationDuplicateDomainAcrossTenants(t *testing.T) {
	h := setup(t)
	h.create(t, h.tenant.ID, "First", "example.com")

	_, err := h.svc.Create(context.Background(), h.other.ID, Input{Name: "Second", Domain: "example.com", Provider: ProviderGmail})
	if !errors.Is(err, ErrDomainTaken) {
		t.Fatalf("domains must be globally unique, got %v", err)
	}
}

func TestIntegrationDuplicateDomainDiffersOnlyByCaseOrSpace(t *testing.T) {
	h := setup(t)
	original := h.create(t, h.tenant.ID, "First", "example.com")

	for _, candidate := range []string{"EXAMPLE.com", "  Example.COM  ", "www.example.com."} {
		if _, err := h.svc.Create(context.Background(), h.tenant.ID, Input{Name: "Second", Domain: candidate, Provider: ProviderGmail}); !errors.Is(err, ErrDomainTaken) {
			t.Fatalf("%q must collide with example.com, got %v", candidate, err)
		}
	}

	value, ok := h.key(t, "example.com")
	if !ok || value.ConfigID != original.ID {
		t.Fatalf("rejected duplicates must leave the original key intact, got %+v", value)
	}
}

func TestIntegrationSameNameDifferentDomainAllowed(t *testing.T) {
	h := setup(t)
	h.create(t, h.tenant.ID, "Corporate", "example.com")
	h.create(t, h.tenant.ID, "Corporate", "example.net")
}

func TestIntegrationCreateWritesRedisKeyWithoutTTL(t *testing.T) {
	h := setup(t)
	row := h.create(t, h.tenant.ID, "Corporate", "example.com")

	value, ok := h.key(t, "example.com")
	if !ok {
		t.Fatal("authorized domain key was not written")
	}
	if value.ConfigID != row.ID || value.CustomerID != h.tenant.ID {
		t.Fatalf("payload = %+v", value)
	}

	ttl, err := h.rdb.PTTL(context.Background(), authorizedDomainKey("example.com")).Result()
	if err != nil {
		t.Fatalf("pttl failed: %v", err)
	}
	if ttl != -1 {
		t.Fatalf("key must not expire, pttl = %v", ttl)
	}
}

func TestIntegrationUpdateRekeysRedis(t *testing.T) {
	h := setup(t)
	row := h.create(t, h.tenant.ID, "Corporate", "example.com")

	updated, err := h.svc.Update(context.Background(), h.tenant.ID, row.ID, Input{
		Name:     "Corporate",
		Domain:   "mail.example.com",
		Provider: ProviderOutlook365,
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Domain != "mail.example.com" || updated.Provider != ProviderOutlook365 {
		t.Fatalf("update did not apply: %+v", updated)
	}

	if _, ok := h.key(t, "example.com"); ok {
		t.Fatal("old domain key must be evicted")
	}

	value, ok := h.key(t, "mail.example.com")
	if !ok || value.ConfigID != row.ID {
		t.Fatalf("new domain key missing or wrong: %+v", value)
	}
}

func TestIntegrationUpdateWithoutDomainChangeKeepsOneKey(t *testing.T) {
	h := setup(t)
	row := h.create(t, h.tenant.ID, "Corporate", "example.com")

	if _, err := h.svc.Update(context.Background(), h.tenant.ID, row.ID, Input{
		Name:     "Renamed",
		Domain:   "example.com",
		Provider: ProviderGmail,
	}); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	keys, err := h.rdb.Keys(context.Background(), "authorized_domain:*").Result()
	if err != nil {
		t.Fatalf("keys failed: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected exactly one key, got %v", keys)
	}
}

func TestIntegrationUpdateOntoExistingDomainConflicts(t *testing.T) {
	h := setup(t)
	h.create(t, h.tenant.ID, "First", "example.com")
	second := h.create(t, h.tenant.ID, "Second", "example.net")

	_, err := h.svc.Update(context.Background(), h.tenant.ID, second.ID, Input{
		Name:     "Second",
		Domain:   "example.com",
		Provider: ProviderGmail,
	})
	if !errors.Is(err, ErrDomainTaken) {
		t.Fatalf("expected ErrDomainTaken, got %v", err)
	}
}

func TestIntegrationUpdateOntoOwnDomainSucceeds(t *testing.T) {
	h := setup(t)
	row := h.create(t, h.tenant.ID, "First", "example.com")

	if _, err := h.svc.Update(context.Background(), h.tenant.ID, row.ID, Input{
		Name:     "First",
		Domain:   "EXAMPLE.com",
		Provider: ProviderGmail,
	}); err != nil {
		t.Fatalf("update onto its own domain failed: %v", err)
	}
}

func TestIntegrationDeleteEvictsOnlyItsOwnKey(t *testing.T) {
	h := setup(t)
	target := h.create(t, h.tenant.ID, "First", "example.com")
	h.create(t, h.tenant.ID, "Second", "example.net")

	if err := h.svc.Delete(context.Background(), h.tenant.ID, target.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if _, ok := h.key(t, "example.com"); ok {
		t.Fatal("deleted configuration key must be gone")
	}
	if _, ok := h.key(t, "example.net"); !ok {
		t.Fatal("sibling key must survive")
	}

	if err := h.svc.Delete(context.Background(), h.tenant.ID, target.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete = %v, want ErrNotFound", err)
	}
}

func TestIntegrationTenantIsolation(t *testing.T) {
	h := setup(t)
	row := h.create(t, h.tenant.ID, "Corporate", "example.com")
	ctx := context.Background()

	if _, err := h.svc.Get(ctx, h.other.ID, row.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get across tenants = %v, want ErrNotFound", err)
	}
	if _, err := h.svc.Update(ctx, h.other.ID, row.ID, Input{Name: "x", Domain: "other.com", Provider: ProviderGmail}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update across tenants = %v, want ErrNotFound", err)
	}
	if err := h.svc.Delete(ctx, h.other.ID, row.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete across tenants = %v, want ErrNotFound", err)
	}
	if _, err := h.svc.GenerateDKIM(ctx, h.other.ID, row.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("dkim across tenants = %v, want ErrNotFound", err)
	}
	if _, _, err := h.svc.GenerateAccessToken(ctx, h.other.ID, row.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("token across tenants = %v, want ErrNotFound", err)
	}

	if _, ok := h.key(t, "example.com"); !ok {
		t.Fatal("a rejected cross-tenant write must not touch redis")
	}
}

func TestIntegrationListIsTenantScoped(t *testing.T) {
	h := setup(t)
	h.create(t, h.tenant.ID, "Mine", "example.com")
	h.create(t, h.other.ID, "Theirs", "example.net")

	listing, err := h.svc.List(context.Background(), h.tenant.ID, ListParams{})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if listing.Total != 1 || len(listing.Items) != 1 || listing.Items[0].Name != "Mine" {
		t.Fatalf("list leaked another tenant: %+v", listing)
	}
}

func TestIntegrationListSearchDoesNotEscapeTenantScope(t *testing.T) {
	h := setup(t)
	h.create(t, h.tenant.ID, "Mine", "mine.example.com")
	h.create(t, h.other.ID, "Theirs", "shared-term.example.net")

	listing, err := h.svc.List(context.Background(), h.tenant.ID, ListParams{Search: "shared-term"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if listing.Total != 0 || len(listing.Items) != 0 {
		t.Fatalf("search matched another tenant's rows: %+v", listing.Items)
	}
}

func TestIntegrationListSearchTrimsAndIgnoresCase(t *testing.T) {
	h := setup(t)
	h.create(t, h.tenant.ID, "Marketing outbound", "marketing.example.com")
	h.create(t, h.tenant.ID, "Support", "support.example.net")

	for _, search := range []string{"marketing", "  MARKETING  ", "Marketing.Example"} {
		listing, err := h.svc.List(context.Background(), h.tenant.ID, ListParams{Search: search})
		if err != nil {
			t.Fatalf("list %q failed: %v", search, err)
		}
		if listing.Total != 1 {
			t.Fatalf("search %q returned %d rows", search, listing.Total)
		}
	}
}

func TestIntegrationListSearchTreatsWildcardAsLiteral(t *testing.T) {
	h := setup(t)
	h.create(t, h.tenant.ID, "Marketing", "marketing.example.com")

	listing, err := h.svc.List(context.Background(), h.tenant.ID, ListParams{Search: "%"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if listing.Total != 0 {
		t.Fatalf("a literal %% must not match everything, got %d rows", listing.Total)
	}
}

func TestIntegrationListPaginates(t *testing.T) {
	h := setup(t)

	for index := range 5 {
		h.create(t, h.tenant.ID, "Config "+strconv.Itoa(index), "example"+strconv.Itoa(index)+".com")
	}

	first, err := h.svc.List(context.Background(), h.tenant.ID, ListParams{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if first.Total != 5 || len(first.Items) != 2 || first.PageSize != 2 {
		t.Fatalf("first page = %+v", first)
	}

	third, err := h.svc.List(context.Background(), h.tenant.ID, ListParams{Page: 3, PageSize: 2})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(third.Items) != 1 {
		t.Fatalf("third page should hold the remainder, got %d", len(third.Items))
	}

	beyond, err := h.svc.List(context.Background(), h.tenant.ID, ListParams{Page: 9, PageSize: 2})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if beyond.Items == nil || len(beyond.Items) != 0 {
		t.Fatalf("a page past the end must be empty, not nil: %+v", beyond.Items)
	}

	clamped, err := h.svc.List(context.Background(), h.tenant.ID, ListParams{Page: 1, PageSize: 5000})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if clamped.PageSize != maxPageSize {
		t.Fatalf("page size = %d, want clamped to %d", clamped.PageSize, maxPageSize)
	}
}

func TestIntegrationGenerateDKIM(t *testing.T) {
	h := setup(t)
	row := h.create(t, h.tenant.ID, "Corporate", "example.com")

	first, err := h.svc.GenerateDKIM(context.Background(), h.tenant.ID, row.ID)
	if err != nil {
		t.Fatalf("dkim failed: %v", err)
	}
	if first.DKIMPublicKey == nil || !strings.HasPrefix(*first.DKIMPublicKey, "v=DKIM1;") {
		t.Fatalf("unexpected dkim value %v", first.DKIMPublicKey)
	}

	second, err := h.svc.GenerateDKIM(context.Background(), h.tenant.ID, row.ID)
	if err != nil {
		t.Fatalf("dkim regeneration failed: %v", err)
	}
	if *second.DKIMPublicKey == *first.DKIMPublicKey {
		t.Fatal("regeneration must produce a new key")
	}

	stored := h.stored(t, row.ID)
	if stored.DKIMPublicKey == nil || *stored.DKIMPublicKey != *second.DKIMPublicKey {
		t.Fatal("dkim key was not persisted")
	}
}

func TestIntegrationGenerateAccessTokenStoresOnlyHash(t *testing.T) {
	h := setup(t)
	row := h.create(t, h.tenant.ID, "Corporate", "example.com")

	token, expiresAt, err := h.svc.GenerateAccessToken(context.Background(), h.tenant.ID, row.ID)
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	stored := h.stored(t, row.ID)
	if stored.AccessTokenHash == nil {
		t.Fatal("hash was not persisted")
	}
	if *stored.AccessTokenHash == token {
		t.Fatal("plaintext token must never be stored")
	}
	if *stored.AccessTokenHash != auth.HashToken(token) {
		t.Fatal("stored value is not the sha256 hash of the issued token")
	}

	remaining := time.Until(expiresAt)
	if remaining < config.ProviderTokenTTL-time.Hour || remaining > config.ProviderTokenTTL {
		t.Fatalf("expiry %v is not one year out", remaining)
	}

	var customer db.Customer
	if err := h.database.Where("id = ?", h.tenant.ID).Take(&customer).Error; err != nil {
		t.Fatalf("secret lookup failed: %v", err)
	}

	claims, err := ParseAccessToken(token, customer.JWTSecret, "dpdp")
	if err != nil {
		t.Fatalf("issued token does not verify: %v", err)
	}
	if claims.Domain != "example.com" || claims.ConfigID != row.ID || claims.CustomerID != h.tenant.ID {
		t.Fatalf("claims = %+v", claims)
	}

	regenerated, _, err := h.svc.GenerateAccessToken(context.Background(), h.tenant.ID, row.ID)
	if err != nil {
		t.Fatalf("regeneration failed: %v", err)
	}
	if h.stored(t, row.ID).AccessTokenHash == nil || *h.stored(t, row.ID).AccessTokenHash != auth.HashToken(regenerated) {
		t.Fatal("regeneration must replace the stored hash")
	}
}

func TestIntegrationSecretsNeverLeaveTheDatabase(t *testing.T) {
	h := setup(t)
	row := h.create(t, h.tenant.ID, "Corporate", "example.com")

	if _, _, err := h.svc.GenerateAccessToken(context.Background(), h.tenant.ID, row.ID); err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	fetched, err := h.svc.Get(context.Background(), h.tenant.ID, row.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if fetched.AccessTokenHash != nil {
		t.Fatal("read paths must not select the token hash")
	}

	listing, err := h.svc.List(context.Background(), h.tenant.ID, ListParams{})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if listing.Items[0].AccessTokenHash != nil {
		t.Fatal("list must not select the token hash")
	}

	body, err := json.Marshal(toResponse(fetched))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(body), "access_token_hash") {
		t.Fatalf("response body exposes the hash: %s", body)
	}
}

func TestIntegrationHTTPDuplicateDomainReturns409(t *testing.T) {
	h := setup(t)
	h.create(t, h.tenant.ID, "First", "example.com")

	router := h.router(t)

	body := `{"name":"Second","domain":"EXAMPLE.com","provider":"gmail"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/email/configurations", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409, body %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response is not json: %v", err)
	}
	if payload.Error != "domain_taken" || !strings.Contains(payload.Message, "example.com") {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestIntegrationHTTPListReturnsItemsEnvelope(t *testing.T) {
	h := setup(t)
	h.create(t, h.tenant.ID, "Corporate", "example.com")

	router := h.router(t)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/email/configurations?search=corp&page=1&page_size=10", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", recorder.Code, recorder.Body.String())
	}

	var payload ListResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response is not json: %v", err)
	}
	if payload.Total != 1 || len(payload.Items) != 1 || payload.PageSize != 10 {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestIntegrationHTTPUnknownIDReturns404(t *testing.T) {
	h := setup(t)
	router := h.router(t)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/email/configurations/9999", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestIntegrationMigrationSeedsPrivileges(t *testing.T) {
	h := setup(t)

	var count int64
	if err := h.database.Table("privileges").Where("name LIKE ? AND type = ?", "admin.email.provider.%", "DASHBOARD").Count(&count).Error; err != nil {
		t.Fatalf("privilege lookup failed: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected 4 seeded privileges, got %d", count)
	}
}

func (h harness) stored(t *testing.T, id int) db.EmailProviderConfiguration {
	t.Helper()

	var row db.EmailProviderConfiguration
	if err := h.database.Where("id = ?", id).Take(&row).Error; err != nil {
		t.Fatalf("row lookup failed: %v", err)
	}

	return row
}

func (h harness) router(t *testing.T) *gin.Engine {
	t.Helper()

	role := testsupport.Role(t, h.database, h.tenant.ID,
		PrivilegeView, PrivilegeCreate, PrivilegeEdit, PrivilegeDelete)

	router := gin.New()
	group := router.Group("/api/v1")

	group.Use(func(c *gin.Context) {
		auth.SetClaims(c, &auth.Claims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: "1"},
			CustomerID:       h.tenant.ID,
			RoleID:           &role.ID,
		})
		c.Next()
	})

	NewHandler(h.svc).RegisterRoutes(group, func(string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})

	return router
}
