//go:build integration

package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	providerhandler "dpdp-backend/internal/admin/handler/emailprovider"
	userhandler "dpdp-backend/internal/admin/handler/emailuser"
	grouphandler "dpdp-backend/internal/admin/handler/group"
	policyhandler "dpdp-backend/internal/admin/handler/policy"
	rulehandler "dpdp-backend/internal/admin/handler/rule"
	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/middleware"
	"dpdp-backend/tests/testsupport"
)

func (h harness) guardedRouter(t *testing.T, privileges ...string) *gin.Engine {
	t.Helper()

	role := testsupport.Role(t, h.database, h.tenant.ID, privileges...)
	store := auth.NewStore(h.rdb)

	router, group := testsupport.Router(t, h.tenant.ID, &role.ID)

	guard := func(name string) gin.HandlerFunc {
		return middleware.RequirePrivilege(h.database, store, name)
	}

	policyhandler.NewPolicyHandler(h.policies).RegisterRoutes(group, guard)
	rulehandler.NewRuleHandler(h.rules).RegisterRoutes(group, guard)
	userhandler.NewEmailUserHandler(h.users).RegisterRoutes(group, guard)
	grouphandler.NewGroupHandler(h.groups).RegisterRoutes(group, guard)
	providerhandler.NewEmailProviderHandler(h.providers).RegisterRoutes(group, guard)

	return router
}

func expectForbidden(t *testing.T, recorder *httptest.ResponseRecorder, privilege string) {
	t.Helper()

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body %s", recorder.Code, recorder.Body.String())
	}

	payload := decodeError(t, recorder)
	if payload.Error != "forbidden" {
		t.Fatalf("error = %q, want forbidden", payload.Error)
	}
	if !strings.Contains(payload.Message, privilege) {
		t.Fatalf("message = %q, want it to name %q", payload.Message, privilege)
	}
}

func TestIntegrationAuthzDeniesEveryPolicyRouteWithoutPrivilege(t *testing.T) {
	h := setup(t)
	router := h.guardedRouter(t)

	cases := []struct {
		method    string
		path      string
		body      string
		privilege string
	}{
		{http.MethodGet, "/api/v1/admin/policies", "", policyhandler.PrivilegePolicyView},
		{http.MethodGet, "/api/v1/admin/policies/1", "", policyhandler.PrivilegePolicyView},
		{http.MethodPost, "/api/v1/admin/policies", `{"policy_name":"x","group_ids":[1],"rule_ids":[1]}`, policyhandler.PrivilegePolicyCreate},
		{http.MethodPut, "/api/v1/admin/policies/1", `{"policy_name":"x","group_ids":[1],"rule_ids":[1]}`, policyhandler.PrivilegePolicyEdit},
		{http.MethodPatch, "/api/v1/admin/policies/1/status", `{"active":true}`, policyhandler.PrivilegePolicyEdit},
		{http.MethodDelete, "/api/v1/admin/policies/1", "", policyhandler.PrivilegePolicyDelete},
	}

	for _, tc := range cases {
		recorder := do(t, router, tc.method, tc.path, tc.body)
		expectForbidden(t, recorder, tc.privilege)
	}
}

func TestIntegrationAuthzDeniesOtherResourcesWithoutPrivilege(t *testing.T) {
	h := setup(t)
	router := h.guardedRouter(t)

	cases := []struct {
		method    string
		path      string
		body      string
		privilege string
	}{
		{http.MethodGet, "/api/v1/admin/rules", "", rulehandler.PrivilegeRuleView},
		{http.MethodPost, "/api/v1/admin/rules", `{"rule_name":"x","type":"KEYWORD","value":"y"}`, rulehandler.PrivilegeRuleCreate},
		{http.MethodDelete, "/api/v1/admin/rules/1", "", rulehandler.PrivilegeRuleDelete},
		{http.MethodGet, "/api/v1/admin/email/users", "", userhandler.PrivilegeEmailUserView},
		{http.MethodPost, "/api/v1/admin/email/users", `{"email":"a@b.com","first_name":"A","last_name":"B"}`, userhandler.PrivilegeEmailUserCreate},
		{http.MethodGet, "/api/v1/admin/email/groups", "", grouphandler.PrivilegeGroupView},
		{http.MethodPost, "/api/v1/admin/email/groups/1/users", `{"email_user_id":1}`, grouphandler.PrivilegeGroupEdit},
		{http.MethodDelete, "/api/v1/admin/email/groups/1/users/1", "", grouphandler.PrivilegeGroupEdit},
	}

	for _, tc := range cases {
		recorder := do(t, router, tc.method, tc.path, tc.body)
		expectForbidden(t, recorder, tc.privilege)
	}
}

func TestIntegrationAuthzViewDoesNotImplyCreate(t *testing.T) {
	h := setup(t)
	router := h.guardedRouter(t, policyhandler.PrivilegePolicyView)

	listed := do(t, router, http.MethodGet, "/api/v1/admin/policies", "")
	expectStatus(t, listed, http.StatusOK)

	created := do(t, router, http.MethodPost, "/api/v1/admin/policies", `{"policy_name":"x","group_ids":[1],"rule_ids":[1]}`)
	expectForbidden(t, created, policyhandler.PrivilegePolicyCreate)
}

func TestIntegrationAuthzGrantedPrivilegeAllows(t *testing.T) {
	h := setup(t)
	router := h.guardedRouter(t, rulehandler.PrivilegeRuleView, rulehandler.PrivilegeRuleCreate)

	created := do(t, router, http.MethodPost, "/api/v1/admin/rules", `{"rule_name":"Card","type":"KEYWORD","value":"card"}`)
	expectStatus(t, created, http.StatusCreated)

	listed := do(t, router, http.MethodGet, "/api/v1/admin/rules", "")
	expectStatus(t, listed, http.StatusOK)
}

func TestIntegrationAuthzRoleWithoutRoleIDIsDenied(t *testing.T) {
	h := setup(t)

	store := auth.NewStore(h.rdb)
	router, group := testsupport.Router(t, h.tenant.ID, nil)

	rulehandler.NewRuleHandler(h.rules).RegisterRoutes(group, func(name string) gin.HandlerFunc {
		return middleware.RequirePrivilege(h.database, store, name)
	})

	recorder := do(t, router, http.MethodGet, "/api/v1/admin/rules", "")
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}

	if message := decodeError(t, recorder).Message; !strings.Contains(message, "no privileges assigned") {
		t.Fatalf("message = %q", message)
	}
}
