//go:build integration

package handler_test

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"

	rulerepo "dpdp-backend/internal/admin/repositories/rule"
	rulesvc "dpdp-backend/internal/admin/services/rule"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

func TestIntegrationRuleCreateBothTypes(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	regex, err := h.rules.Create(ctx, h.tenant.ID, rulesvc.RuleInput{
		RuleName: "  Card   number ",
		Type:     "regex",
		Value:    `\d{16}`,
	})
	if err != nil {
		t.Fatalf("regex rule create failed: %v", err)
	}
	if regex.RuleName != "Card number" || regex.Type != db.RuleTypeRegex {
		t.Fatalf("rule = %+v", regex)
	}

	keyword, err := h.rules.Create(ctx, h.tenant.ID, rulesvc.RuleInput{
		RuleName: "Confidential",
		Type:     db.RuleTypeKeyword,
		Value:    "  confidential  ",
	})
	if err != nil {
		t.Fatalf("keyword rule create failed: %v", err)
	}
	if keyword.Value != "confidential" {
		t.Fatalf("keyword value = %q", keyword.Value)
	}
}

func TestIntegrationRuleRejectsInvalidInput(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	_, err := h.rules.Create(ctx, h.tenant.ID, rulesvc.RuleInput{RuleName: "Bad", Type: "glob", Value: "x"})
	if !errors.Is(err, utils.ErrInvalidRuleType) {
		t.Fatalf("error = %v, want ErrInvalidRuleType", err)
	}

	_, err = h.rules.Create(ctx, h.tenant.ID, rulesvc.RuleInput{RuleName: "Bad", Type: db.RuleTypeRegex, Value: "["})
	if !errors.Is(err, utils.ErrInvalidRegex) {
		t.Fatalf("error = %v, want ErrInvalidRegex", err)
	}

	if total := h.count(t, &db.Rule{}, "customer_id = ?", h.tenant.ID); total != 0 {
		t.Fatalf("invalid rules were persisted: %d rows", total)
	}
}

func TestIntegrationRuleDuplicateNameSameTenant(t *testing.T) {
	h := setup(t)
	h.rule(t, h.tenant.ID, "Card")

	_, err := h.rules.Create(context.Background(), h.tenant.ID, rulesvc.RuleInput{
		RuleName: "Card",
		Type:     db.RuleTypeKeyword,
		Value:    "other",
	})

	if !errors.Is(err, utils.ErrRuleNameTaken) {
		t.Fatalf("error = %v, want ErrRuleNameTaken", err)
	}
}

func TestIntegrationRuleSameNameAcrossTenants(t *testing.T) {
	h := setup(t)

	h.rule(t, h.tenant.ID, "Card")
	h.rule(t, h.other.ID, "Card")
}

func TestIntegrationRuleUpdateRevalidatesValue(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	row, err := h.rules.Create(ctx, h.tenant.ID, rulesvc.RuleInput{
		RuleName: "Bracket",
		Type:     db.RuleTypeKeyword,
		Value:    "[",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	_, err = h.rules.Update(ctx, h.tenant.ID, row.ID, rulesvc.RuleInput{
		RuleName: "Bracket",
		Type:     db.RuleTypeRegex,
		Value:    "[",
	})
	if !errors.Is(err, utils.ErrInvalidRegex) {
		t.Fatalf("promotion to regex must revalidate, error = %v", err)
	}

	updated, err := h.rules.Update(ctx, h.tenant.ID, row.ID, rulesvc.RuleInput{
		RuleName: "Bracket class",
		Type:     db.RuleTypeRegex,
		Value:    "[a-z]",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Type != db.RuleTypeRegex || updated.Value != "[a-z]" {
		t.Fatalf("rule = %+v", updated)
	}
}

func TestIntegrationRuleTenantIsolation(t *testing.T) {
	h := setup(t)
	row := h.rule(t, h.tenant.ID, "Card")
	ctx := context.Background()

	if _, err := h.rules.Get(ctx, h.other.ID, row.ID); !errors.Is(err, utils.ErrRuleNotFound) {
		t.Fatalf("get across tenants = %v", err)
	}

	_, err := h.rules.Update(ctx, h.other.ID, row.ID, rulesvc.RuleInput{
		RuleName: "Hijack",
		Type:     db.RuleTypeKeyword,
		Value:    "x",
	})
	if !errors.Is(err, utils.ErrRuleNotFound) {
		t.Fatalf("update across tenants = %v", err)
	}

	if err := h.rules.Delete(ctx, h.other.ID, row.ID); !errors.Is(err, utils.ErrRuleNotFound) {
		t.Fatalf("delete across tenants = %v", err)
	}
}

func TestIntegrationRuleListSearchAndPagination(t *testing.T) {
	h := setup(t)

	h.rule(t, h.tenant.ID, "Marketing card")
	h.rule(t, h.tenant.ID, "Support passport")
	h.rule(t, h.other.ID, "Marketing theirs")

	ctx := context.Background()

	scoped, err := h.rules.List(ctx, h.tenant.ID, rulerepo.ListParams{})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if scoped.Total != 2 {
		t.Fatalf("list total = %d, want 2", scoped.Total)
	}

	searched, err := h.rules.List(ctx, h.tenant.ID, rulerepo.ListParams{Search: "  MARKETING "})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if searched.Total != 1 {
		t.Fatalf("search total = %d, want 1", searched.Total)
	}

	wildcard, err := h.rules.List(ctx, h.tenant.ID, rulerepo.ListParams{Search: "%"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if wildcard.Total != 0 {
		t.Fatalf("literal %% matched %d rows", wildcard.Total)
	}

	paged, err := h.rules.List(ctx, h.tenant.ID, rulerepo.ListParams{Page: 2, PageSize: 1})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(paged.Items) != 1 || paged.Total != 2 {
		t.Fatalf("page = %+v", paged)
	}
}

func TestIntegrationHTTPRuleErrors(t *testing.T) {
	h := setup(t)
	router := h.router(t)

	created := do(t, router, http.MethodPost, "/api/v1/admin/rules",
		`{"rule_name":"Card","type":"REGEX","value":"\\d{16}"}`)
	expectStatus(t, created, http.StatusCreated)

	badRegex := do(t, router, http.MethodPost, "/api/v1/admin/rules",
		`{"rule_name":"Broken","type":"REGEX","value":"["}`)
	expectStatus(t, badRegex, http.StatusBadRequest)

	payload := decodeError(t, badRegex)
	if payload.Error != utils.CodeInvalidRegex || !strings.Contains(payload.Message, "error parsing regexp") {
		t.Fatalf("payload = %+v", payload)
	}

	badType := do(t, router, http.MethodPost, "/api/v1/admin/rules",
		`{"rule_name":"Broken","type":"GLOB","value":"x"}`)
	expectStatus(t, badType, http.StatusBadRequest)

	duplicate := do(t, router, http.MethodPost, "/api/v1/admin/rules",
		`{"rule_name":"Card","type":"KEYWORD","value":"card"}`)
	expectStatus(t, duplicate, http.StatusConflict)

	if code := decodeError(t, duplicate).Error; code != utils.CodeRuleNameTaken {
		t.Fatalf("duplicate code = %q", code)
	}
}

func TestIntegrationHTTPRuleDeleteReturns204(t *testing.T) {
	h := setup(t)
	router := h.router(t)

	row := h.rule(t, h.tenant.ID, "Card")

	recorder := do(t, router, http.MethodDelete, "/api/v1/admin/rules/"+strconv.Itoa(row.ID), "")
	expectStatus(t, recorder, http.StatusNoContent)
}

func TestIntegrationMigrationSeedsRulePrivileges(t *testing.T) {
	h := setup(t)

	var count int64
	err := h.database.Table("privileges").
		Where("name LIKE ? AND type = ?", "admin.rule.%", "DASHBOARD").
		Count(&count).Error
	if err != nil {
		t.Fatalf("privilege lookup failed: %v", err)
	}

	if count != 4 {
		t.Fatalf("rule privileges = %d, want 4", count)
	}
}
