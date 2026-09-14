//go:build integration

package handler_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	providerhandler "dpdp-backend/internal/admin/handler/emailprovider"
	userhandler "dpdp-backend/internal/admin/handler/emailuser"
	grouphandler "dpdp-backend/internal/admin/handler/group"
	policyhandler "dpdp-backend/internal/admin/handler/policy"
	rulehandler "dpdp-backend/internal/admin/handler/rule"
	brandingrepo "dpdp-backend/internal/admin/repositories/branding"
	providerrepo "dpdp-backend/internal/admin/repositories/emailprovider"
	userrepo "dpdp-backend/internal/admin/repositories/emailuser"
	grouprepo "dpdp-backend/internal/admin/repositories/group"
	policyrepo "dpdp-backend/internal/admin/repositories/policy"
	rulerepo "dpdp-backend/internal/admin/repositories/rule"
	brandingsvc "dpdp-backend/internal/admin/services/branding"
	providersvc "dpdp-backend/internal/admin/services/emailprovider"
	usersvc "dpdp-backend/internal/admin/services/emailuser"
	groupsvc "dpdp-backend/internal/admin/services/group"
	policysvc "dpdp-backend/internal/admin/services/policy"
	rulesvc "dpdp-backend/internal/admin/services/rule"
	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/config"
	"dpdp-backend/internal/db"
	"dpdp-backend/tests/testsupport"
)

type harness struct {
	database *gorm.DB
	rdb      *redis.Client
	tenant   db.Customer
	other    db.Customer

	policies  *policysvc.PolicyService
	rules     *rulesvc.RuleService
	users     *usersvc.EmailUserService
	groups    *groupsvc.GroupService
	providers *providersvc.EmailProviderService
	branding  *brandingsvc.BrandingService
}

func setup(t *testing.T) harness {
	t.Helper()

	gin.SetMode(gin.TestMode)

	database := testsupport.Postgres(t)
	rdb := testsupport.Redis(t)

	testsupport.Reset(t, database, rdb)
	t.Cleanup(func() { testsupport.Reset(t, database, rdb) })

	return harness{
		database:  database,
		rdb:       rdb,
		tenant:    testsupport.Customer(t, database),
		other:     testsupport.Customer(t, database),
		policies:  policysvc.NewPolicyService(database, policyrepo.NewPolicyRepository(database)),
		rules:     rulesvc.NewRuleService(database, rulerepo.NewRuleRepository(database)),
		users:     usersvc.NewEmailUserService(database, userrepo.NewEmailUserRepository(database)),
		groups:    groupsvc.NewGroupService(database, grouprepo.NewGroupRepository(database)),
		providers: providersvc.NewEmailProviderService(database, providerrepo.NewEmailProviderRepository(database), providerrepo.NewRedisRepository(rdb), config.Auth{Issuer: "dpdp"}),
		branding: brandingsvc.NewBrandingService(
			brandingrepo.NewBrandingRepository(database),
			testsupport.OptionalStorage(t),
			auth.NewStore(rdb),
			24*time.Hour,
		),
	}
}

func (h harness) router(t *testing.T) *gin.Engine {
	t.Helper()

	router, group := testsupport.Router(t, h.tenant.ID, nil)

	pass := func(string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	}

	policyhandler.NewPolicyHandler(h.policies).RegisterRoutes(group, pass)
	rulehandler.NewRuleHandler(h.rules).RegisterRoutes(group, pass)
	userhandler.NewEmailUserHandler(h.users).RegisterRoutes(group, pass)
	grouphandler.NewGroupHandler(h.groups).RegisterRoutes(group, pass)
	providerhandler.NewEmailProviderHandler(h.providers).RegisterRoutes(group, pass)

	return router
}

func (h harness) count(t *testing.T, model any, query string, args ...any) int64 {
	t.Helper()

	var total int64
	if err := h.database.Model(model).Where(query, args...).Count(&total).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}

	return total
}

func (h harness) rule(t *testing.T, customerID int, name string) db.Rule {
	t.Helper()

	return testsupport.Rule(t, h.database, customerID, name, db.RuleTypeKeyword, "secret")
}

func (h harness) group(t *testing.T, customerID int, name string) db.Group {
	t.Helper()

	return testsupport.Group(t, h.database, customerID, name)
}

func (h harness) user(t *testing.T, customerID int, email string) db.EmailUser {
	t.Helper()

	return testsupport.EmailUser(t, h.database, customerID, email)
}

func (h harness) policy(t *testing.T, customerID int, name string) policysvc.PolicyDetail {
	t.Helper()

	group := h.group(t, customerID, name+" group")
	rule := h.rule(t, customerID, name+" rule")

	detail, err := h.policies.Create(context.Background(), customerID, policysvc.PolicyInput{
		PolicyName: name,
		Active:     true,
		GroupIDs:   []int{group.ID},
		RuleIDs:    []int{rule.ID},
	})
	if err != nil {
		t.Fatalf("policy %q create failed: %v", name, err)
	}

	return detail
}

type apiError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func do(t *testing.T, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}

	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

func decodeError(t *testing.T, recorder *httptest.ResponseRecorder) apiError {
	t.Helper()

	var payload apiError
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response is not json: %v (%s)", err, recorder.Body.String())
	}

	return payload
}

func expectStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()

	if recorder.Code != want {
		t.Fatalf("status = %d, want %d, body %s", recorder.Code, want, recorder.Body.String())
	}
}
