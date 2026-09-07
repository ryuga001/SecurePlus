//go:build integration

package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"

	dto "dpdp-backend/internal/admin/dto/policy"
	policyrepo "dpdp-backend/internal/admin/repositories/policy"
	policysvc "dpdp-backend/internal/admin/services/policy"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

func TestIntegrationPolicyCreatePersists(t *testing.T) {
	h := setup(t)

	group := h.group(t, h.tenant.ID, "Finance")
	rule := h.rule(t, h.tenant.ID, "Card")

	detail, err := h.policies.Create(context.Background(), h.tenant.ID, policysvc.PolicyInput{
		PolicyName: "  Outbound   PII ",
		Active:     true,
		GroupIDs:   []int{group.ID, group.ID},
		RuleIDs:    []int{rule.ID},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if detail.Policy.PolicyName != "Outbound PII" {
		t.Fatalf("name = %q", detail.Policy.PolicyName)
	}
	if detail.Policy.Type != db.PolicyTypeEmail || !detail.Policy.Active {
		t.Fatalf("policy = %+v", detail.Policy)
	}
	if len(detail.Groups) != 1 || len(detail.Rules) != 1 {
		t.Fatalf("groups = %v rules = %v", detail.Groups, detail.Rules)
	}
	if detail.Groups[0].Name != "Finance" || detail.Rules[0].Name != "Card" {
		t.Fatalf("references = %+v %+v", detail.Groups[0], detail.Rules[0])
	}
}

func TestIntegrationPolicyRequiresNameRulesAndGroups(t *testing.T) {
	h := setup(t)

	group := h.group(t, h.tenant.ID, "Finance")
	rule := h.rule(t, h.tenant.ID, "Card")
	ctx := context.Background()

	_, err := h.policies.Create(ctx, h.tenant.ID, policysvc.PolicyInput{PolicyName: "   ", GroupIDs: []int{group.ID}, RuleIDs: []int{rule.ID}})
	if !errors.Is(err, utils.ErrPolicyNameNeeded) {
		t.Fatalf("blank name error = %v", err)
	}

	_, err = h.policies.Create(ctx, h.tenant.ID, policysvc.PolicyInput{PolicyName: "Outbound", GroupIDs: nil, RuleIDs: []int{rule.ID}})
	if !errors.Is(err, utils.ErrGroupsNeeded) {
		t.Fatalf("no groups error = %v", err)
	}

	_, err = h.policies.Create(ctx, h.tenant.ID, policysvc.PolicyInput{PolicyName: "Outbound", GroupIDs: []int{group.ID}, RuleIDs: nil})
	if !errors.Is(err, utils.ErrRulesNeeded) {
		t.Fatalf("no rules error = %v", err)
	}

	if total := h.count(t, &db.Policy{}, "customer_id = ?", h.tenant.ID); total != 0 {
		t.Fatalf("rejected creates wrote %d rows", total)
	}
}

func TestIntegrationPolicyDuplicateNameSameTenant(t *testing.T) {
	h := setup(t)
	h.policy(t, h.tenant.ID, "Outbound")

	group := h.group(t, h.tenant.ID, "Second group")
	rule := h.rule(t, h.tenant.ID, "Second rule")

	_, err := h.policies.Create(context.Background(), h.tenant.ID, policysvc.PolicyInput{
		PolicyName: "Outbound",
		Active:     true,
		GroupIDs:   []int{group.ID},
		RuleIDs:    []int{rule.ID},
	})

	if !errors.Is(err, utils.ErrPolicyNameTaken) {
		t.Fatalf("error = %v, want ErrPolicyNameTaken", err)
	}
	if !strings.Contains(err.Error(), "Outbound") {
		t.Fatalf("message must name the policy, got %q", err.Error())
	}
}

func TestIntegrationPolicySameNameAcrossTenants(t *testing.T) {
	h := setup(t)

	h.policy(t, h.tenant.ID, "Outbound")
	h.policy(t, h.other.ID, "Outbound")
}

func TestIntegrationPolicyUpdateReplacesMappings(t *testing.T) {
	h := setup(t)
	detail := h.policy(t, h.tenant.ID, "Outbound")

	newGroup := h.group(t, h.tenant.ID, "Legal")
	newRule := h.rule(t, h.tenant.ID, "Passport")

	updated, err := h.policies.Update(context.Background(), h.tenant.ID, detail.Policy.ID, policysvc.PolicyInput{
		PolicyName: "Outbound renamed",
		Active:     false,
		GroupIDs:   []int{newGroup.ID},
		RuleIDs:    []int{newRule.ID},
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	if updated.Policy.PolicyName != "Outbound renamed" || updated.Policy.Active {
		t.Fatalf("policy = %+v", updated.Policy)
	}
	if len(updated.Groups) != 1 || updated.Groups[0].ID != newGroup.ID {
		t.Fatalf("groups = %+v", updated.Groups)
	}
	if len(updated.Rules) != 1 || updated.Rules[0].ID != newRule.ID {
		t.Fatalf("rules = %+v", updated.Rules)
	}

	if total := h.count(t, &db.PolicyGroupMapping{}, "policy_id = ?", detail.Policy.ID); total != 1 {
		t.Fatalf("group mappings = %d, want 1", total)
	}
}

func TestIntegrationPolicyRejectsForeignReferences(t *testing.T) {
	h := setup(t)

	ownGroup := h.group(t, h.tenant.ID, "Finance")
	ownRule := h.rule(t, h.tenant.ID, "Card")
	foreignGroup := h.group(t, h.other.ID, "Their group")
	foreignRule := h.rule(t, h.other.ID, "Their rule")

	ctx := context.Background()

	_, err := h.policies.Create(ctx, h.tenant.ID, policysvc.PolicyInput{
		PolicyName: "Outbound",
		GroupIDs:   []int{foreignGroup.ID},
		RuleIDs:    []int{ownRule.ID},
	})
	if !errors.Is(err, utils.ErrUnknownGroup) {
		t.Fatalf("foreign group error = %v", err)
	}

	_, err = h.policies.Create(ctx, h.tenant.ID, policysvc.PolicyInput{
		PolicyName: "Outbound",
		GroupIDs:   []int{ownGroup.ID},
		RuleIDs:    []int{foreignRule.ID},
	})
	if !errors.Is(err, utils.ErrUnknownRule) {
		t.Fatalf("foreign rule error = %v", err)
	}

	_, err = h.policies.Create(ctx, h.tenant.ID, policysvc.PolicyInput{
		PolicyName: "Outbound",
		GroupIDs:   []int{ownGroup.ID},
		RuleIDs:    []int{999999},
	})
	if !errors.Is(err, utils.ErrUnknownRule) {
		t.Fatalf("unknown rule error = %v", err)
	}
}

func TestIntegrationPolicyCreateRollsBackOnBadReference(t *testing.T) {
	h := setup(t)

	group := h.group(t, h.tenant.ID, "Finance")
	rule := h.rule(t, h.tenant.ID, "Card")
	foreignRule := h.rule(t, h.other.ID, "Their rule")

	_, err := h.policies.Create(context.Background(), h.tenant.ID, policysvc.PolicyInput{
		PolicyName: "Outbound",
		GroupIDs:   []int{group.ID},
		RuleIDs:    []int{rule.ID, foreignRule.ID},
	})
	if !errors.Is(err, utils.ErrUnknownRule) {
		t.Fatalf("error = %v, want ErrUnknownRule", err)
	}

	if total := h.count(t, &db.Policy{}, "customer_id = ?", h.tenant.ID); total != 0 {
		t.Fatalf("policy rows = %d, want 0", total)
	}
	if total := h.count(t, &db.PolicyGroupMapping{}, "customer_id = ?", h.tenant.ID); total != 0 {
		t.Fatalf("group mappings = %d, want 0", total)
	}
	if total := h.count(t, &db.PolicyRuleMapping{}, "customer_id = ?", h.tenant.ID); total != 0 {
		t.Fatalf("rule mappings = %d, want 0", total)
	}
}

func TestIntegrationPolicyUpdateRollsBackOnBadReference(t *testing.T) {
	h := setup(t)
	detail := h.policy(t, h.tenant.ID, "Outbound")

	foreignGroup := h.group(t, h.other.ID, "Their group")

	_, err := h.policies.Update(context.Background(), h.tenant.ID, detail.Policy.ID, policysvc.PolicyInput{
		PolicyName: "Renamed",
		GroupIDs:   []int{foreignGroup.ID},
		RuleIDs:    []int{detail.Rules[0].ID},
	})
	if !errors.Is(err, utils.ErrUnknownGroup) {
		t.Fatalf("error = %v, want ErrUnknownGroup", err)
	}

	current, err := h.policies.Get(context.Background(), h.tenant.ID, detail.Policy.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if current.Policy.PolicyName != "Outbound" {
		t.Fatalf("name changed to %q despite the failure", current.Policy.PolicyName)
	}
	if len(current.Groups) != 1 || current.Groups[0].ID != detail.Groups[0].ID {
		t.Fatalf("groups changed: %+v", current.Groups)
	}
}

func TestIntegrationPolicySetActiveKeepsMappings(t *testing.T) {
	h := setup(t)
	detail := h.policy(t, h.tenant.ID, "Outbound")

	updated, err := h.policies.SetActive(context.Background(), h.tenant.ID, detail.Policy.ID, false)
	if err != nil {
		t.Fatalf("set active failed: %v", err)
	}

	if updated.Policy.Active {
		t.Fatal("policy must be inactive")
	}
	if len(updated.Groups) != 1 || len(updated.Rules) != 1 {
		t.Fatalf("mappings lost: %+v %+v", updated.Groups, updated.Rules)
	}

	back, err := h.policies.SetActive(context.Background(), h.tenant.ID, detail.Policy.ID, true)
	if err != nil {
		t.Fatalf("set active failed: %v", err)
	}
	if !back.Policy.Active {
		t.Fatal("policy must be active again")
	}
}

func TestIntegrationPolicyDeleteRemovesMappings(t *testing.T) {
	h := setup(t)
	detail := h.policy(t, h.tenant.ID, "Outbound")

	if err := h.policies.Delete(context.Background(), h.tenant.ID, detail.Policy.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if total := h.count(t, &db.PolicyGroupMapping{}, "policy_id = ?", detail.Policy.ID); total != 0 {
		t.Fatalf("group mappings = %d, want 0", total)
	}
	if total := h.count(t, &db.PolicyRuleMapping{}, "policy_id = ?", detail.Policy.ID); total != 0 {
		t.Fatalf("rule mappings = %d, want 0", total)
	}

	if err := h.policies.Delete(context.Background(), h.tenant.ID, detail.Policy.ID); !errors.Is(err, utils.ErrPolicyNotFound) {
		t.Fatalf("second delete = %v, want ErrPolicyNotFound", err)
	}
}

func TestIntegrationPolicyDeletingGroupDetachesIt(t *testing.T) {
	h := setup(t)
	detail := h.policy(t, h.tenant.ID, "Outbound")

	if err := h.groups.Delete(context.Background(), h.tenant.ID, detail.Groups[0].ID); err != nil {
		t.Fatalf("group delete failed: %v", err)
	}

	current, err := h.policies.Get(context.Background(), h.tenant.ID, detail.Policy.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(current.Groups) != 0 {
		t.Fatalf("groups = %+v, want empty", current.Groups)
	}
}

func TestIntegrationPolicyDeletingRuleDetachesIt(t *testing.T) {
	h := setup(t)
	detail := h.policy(t, h.tenant.ID, "Outbound")

	if err := h.rules.Delete(context.Background(), h.tenant.ID, detail.Rules[0].ID); err != nil {
		t.Fatalf("rule delete failed: %v", err)
	}

	current, err := h.policies.Get(context.Background(), h.tenant.ID, detail.Policy.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(current.Rules) != 0 {
		t.Fatalf("rules = %+v, want empty", current.Rules)
	}
}

func TestIntegrationPolicyTenantIsolation(t *testing.T) {
	h := setup(t)
	detail := h.policy(t, h.tenant.ID, "Outbound")

	ctx := context.Background()
	group := h.group(t, h.other.ID, "Their group")
	rule := h.rule(t, h.other.ID, "Their rule")

	if _, err := h.policies.Get(ctx, h.other.ID, detail.Policy.ID); !errors.Is(err, utils.ErrPolicyNotFound) {
		t.Fatalf("get across tenants = %v", err)
	}

	_, err := h.policies.Update(ctx, h.other.ID, detail.Policy.ID, policysvc.PolicyInput{
		PolicyName: "Hijack",
		GroupIDs:   []int{group.ID},
		RuleIDs:    []int{rule.ID},
	})
	if !errors.Is(err, utils.ErrPolicyNotFound) {
		t.Fatalf("update across tenants = %v", err)
	}

	if err := h.policies.Delete(ctx, h.other.ID, detail.Policy.ID); !errors.Is(err, utils.ErrPolicyNotFound) {
		t.Fatalf("delete across tenants = %v", err)
	}

	if _, err := h.policies.SetActive(ctx, h.other.ID, detail.Policy.ID, false); !errors.Is(err, utils.ErrPolicyNotFound) {
		t.Fatalf("set active across tenants = %v", err)
	}
}

func TestIntegrationPolicyListIsTenantScoped(t *testing.T) {
	h := setup(t)
	h.policy(t, h.tenant.ID, "Mine")
	h.policy(t, h.other.ID, "Theirs")

	listing, err := h.policies.List(context.Background(), h.tenant.ID, policyrepo.ListParams{})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if listing.Total != 1 || len(listing.Items) != 1 || listing.Items[0].Policy.PolicyName != "Mine" {
		t.Fatalf("list leaked another tenant: %+v", listing.Items)
	}
	if listing.Items[0].GroupCount != 1 || listing.Items[0].RuleCount != 1 {
		t.Fatalf("counts = %+v", listing.Items[0])
	}
}

func TestIntegrationPolicyListSearchDoesNotEscapeTenantScope(t *testing.T) {
	h := setup(t)
	h.policy(t, h.tenant.ID, "Mine")
	h.policy(t, h.other.ID, "shared-term")

	listing, err := h.policies.List(context.Background(), h.tenant.ID, policyrepo.ListParams{Search: "shared-term"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if listing.Total != 0 || len(listing.Items) != 0 {
		t.Fatalf("search matched another tenant: %+v", listing.Items)
	}
}

func TestIntegrationPolicyListFiltersAndPaginates(t *testing.T) {
	h := setup(t)

	for index := range 5 {
		h.policy(t, h.tenant.ID, "Policy "+strconv.Itoa(index))
	}

	first, err := h.policies.List(context.Background(), h.tenant.ID, policyrepo.ListParams{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if first.Total != 5 || len(first.Items) != 2 {
		t.Fatalf("first page = %+v", first)
	}

	third, err := h.policies.List(context.Background(), h.tenant.ID, policyrepo.ListParams{Page: 3, PageSize: 2})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(third.Items) != 1 {
		t.Fatalf("third page has %d rows", len(third.Items))
	}

	clamped, err := h.policies.List(context.Background(), h.tenant.ID, policyrepo.ListParams{Page: 1, PageSize: 5000})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if clamped.PageSize != utils.MaxPageSize {
		t.Fatalf("page size = %d", clamped.PageSize)
	}

	inactive := false
	filtered, err := h.policies.List(context.Background(), h.tenant.ID, policyrepo.ListParams{Active: &inactive})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if filtered.Total != 0 {
		t.Fatalf("inactive filter returned %d rows", filtered.Total)
	}
}

func TestIntegrationHTTPPolicyLifecycle(t *testing.T) {
	h := setup(t)
	router := h.router(t)

	group := h.group(t, h.tenant.ID, "Finance")
	rule := h.rule(t, h.tenant.ID, "Card")

	body := `{"policy_name":"Outbound PII","active":false,"group_ids":[` +
		strconv.Itoa(group.ID) + `],"rule_ids":[` + strconv.Itoa(rule.ID) + `]}`

	created := do(t, router, http.MethodPost, "/api/v1/admin/policies", body)
	expectStatus(t, created, http.StatusCreated)

	var payload dto.PolicyResponse
	if err := json.Unmarshal(created.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response is not json: %v", err)
	}
	if payload.Active {
		t.Fatal("active:false must round-trip")
	}
	if len(payload.Groups) != 1 || len(payload.Rules) != 1 {
		t.Fatalf("payload = %+v", payload)
	}

	path := "/api/v1/admin/policies/" + strconv.Itoa(payload.ID)

	fetched := do(t, router, http.MethodGet, path, "")
	expectStatus(t, fetched, http.StatusOK)

	toggled := do(t, router, http.MethodPatch, path+"/status", `{"active":true}`)
	expectStatus(t, toggled, http.StatusOK)

	listed := do(t, router, http.MethodGet, "/api/v1/admin/policies?page=1&page_size=10", "")
	expectStatus(t, listed, http.StatusOK)

	var listing utils.ListResponse[dto.PolicyListItem]
	if err := json.Unmarshal(listed.Body.Bytes(), &listing); err != nil {
		t.Fatalf("list response is not json: %v", err)
	}
	if listing.Total != 1 || listing.Items[0].GroupCount != 1 || listing.Items[0].RuleCount != 1 {
		t.Fatalf("listing = %+v", listing)
	}

	removed := do(t, router, http.MethodDelete, path, "")
	expectStatus(t, removed, http.StatusNoContent)
}

func TestIntegrationHTTPPolicyValidationErrors(t *testing.T) {
	h := setup(t)
	router := h.router(t)

	group := h.group(t, h.tenant.ID, "Finance")
	rule := h.rule(t, h.tenant.ID, "Card")

	noRules := do(t, router, http.MethodPost, "/api/v1/admin/policies",
		`{"policy_name":"Outbound","group_ids":[`+strconv.Itoa(group.ID)+`],"rule_ids":[]}`)
	expectStatus(t, noRules, http.StatusBadRequest)

	noGroups := do(t, router, http.MethodPost, "/api/v1/admin/policies",
		`{"policy_name":"Outbound","group_ids":[],"rule_ids":[`+strconv.Itoa(rule.ID)+`]}`)
	expectStatus(t, noGroups, http.StatusBadRequest)

	noName := do(t, router, http.MethodPost, "/api/v1/admin/policies",
		`{"policy_name":"","group_ids":[`+strconv.Itoa(group.ID)+`],"rule_ids":[`+strconv.Itoa(rule.ID)+`]}`)
	expectStatus(t, noName, http.StatusBadRequest)

	missing := do(t, router, http.MethodGet, "/api/v1/admin/policies/999999", "")
	expectStatus(t, missing, http.StatusNotFound)

	nonNumeric := do(t, router, http.MethodGet, "/api/v1/admin/policies/abc", "")
	expectStatus(t, nonNumeric, http.StatusBadRequest)
}

func TestIntegrationHTTPPolicyForeignGroupReturns400(t *testing.T) {
	h := setup(t)
	router := h.router(t)

	rule := h.rule(t, h.tenant.ID, "Card")
	foreign := h.group(t, h.other.ID, "Their group")

	recorder := do(t, router, http.MethodPost, "/api/v1/admin/policies",
		`{"policy_name":"Outbound","group_ids":[`+strconv.Itoa(foreign.ID)+`],"rule_ids":[`+strconv.Itoa(rule.ID)+`]}`)

	expectStatus(t, recorder, http.StatusBadRequest)

	if payload := decodeError(t, recorder); payload.Error != utils.CodeGroupNotFound {
		t.Fatalf("error code = %q", payload.Error)
	}
}

func TestIntegrationHTTPPolicyDuplicateNameReturns409(t *testing.T) {
	h := setup(t)
	router := h.router(t)

	detail := h.policy(t, h.tenant.ID, "Outbound")

	recorder := do(t, router, http.MethodPost, "/api/v1/admin/policies",
		`{"policy_name":"Outbound","group_ids":[`+strconv.Itoa(detail.Groups[0].ID)+`],"rule_ids":[`+strconv.Itoa(detail.Rules[0].ID)+`]}`)

	expectStatus(t, recorder, http.StatusConflict)

	payload := decodeError(t, recorder)
	if payload.Error != utils.CodePolicyNameTaken || !strings.Contains(payload.Message, "Outbound") {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestIntegrationMigrationSeedsPolicyPrivileges(t *testing.T) {
	h := setup(t)

	var count int64
	err := h.database.Table("privileges").
		Where("name LIKE ? AND type = ?", "admin.policy.%", "DASHBOARD").
		Count(&count).Error
	if err != nil {
		t.Fatalf("privilege lookup failed: %v", err)
	}

	if count != 4 {
		t.Fatalf("policy privileges = %d, want 4", count)
	}
}
