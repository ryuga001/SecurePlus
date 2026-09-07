package services_test

import (
	"errors"
	"strings"
	"testing"

	rulesvc "dpdp-backend/internal/admin/services/rule"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

func TestNormalizeRuleInputUppercasesType(t *testing.T) {
	input, err := rulesvc.NormalizeRuleInput(rulesvc.RuleInput{RuleName: "Card", Type: " regex ", Value: `\d{16}`})
	if err != nil {
		t.Fatalf("rulesvc.NormalizeRuleInput returned %v", err)
	}

	if input.Type != db.RuleTypeRegex {
		t.Fatalf("type = %q", input.Type)
	}
}

func TestNormalizeRuleInputRejectsUnknownType(t *testing.T) {
	_, err := rulesvc.NormalizeRuleInput(rulesvc.RuleInput{RuleName: "Card", Type: "glob", Value: "x"})
	if !errors.Is(err, utils.ErrInvalidRuleType) {
		t.Fatalf("error = %v, want ErrInvalidRuleType", err)
	}
}

func TestNormalizeRuleInputRequiresName(t *testing.T) {
	_, err := rulesvc.NormalizeRuleInput(rulesvc.RuleInput{RuleName: "  ", Type: db.RuleTypeKeyword, Value: "secret"})
	if !errors.Is(err, utils.ErrRuleNameNeeded) {
		t.Fatalf("error = %v, want ErrRuleNameNeeded", err)
	}
}

func TestNormalizeRuleInputRequiresValue(t *testing.T) {
	_, err := rulesvc.NormalizeRuleInput(rulesvc.RuleInput{RuleName: "Card", Type: db.RuleTypeKeyword, Value: "   "})
	if !errors.Is(err, utils.ErrRuleValueNeeded) {
		t.Fatalf("error = %v, want ErrRuleValueNeeded", err)
	}
}

func TestNormalizeRuleInputAcceptsValidRegex(t *testing.T) {
	for _, value := range []string{`\d{16}`, `(?i)confidential`, `^a|b$`, `[A-Z]{2,4}`} {
		if _, err := rulesvc.NormalizeRuleInput(rulesvc.RuleInput{RuleName: "Rule", Type: db.RuleTypeRegex, Value: value}); err != nil {
			t.Errorf("regex %q rejected: %v", value, err)
		}
	}
}

func TestNormalizeRuleInputRejectsInvalidRegex(t *testing.T) {
	for _, value := range []string{`[`, `(`, `a{2,1}`, `*`, `\`} {
		_, err := rulesvc.NormalizeRuleInput(rulesvc.RuleInput{RuleName: "Rule", Type: db.RuleTypeRegex, Value: value})
		if !errors.Is(err, utils.ErrInvalidRegex) {
			t.Errorf("regex %q accepted, error = %v", value, err)
		}
	}
}

func TestNormalizeRuleInputInvalidRegexCarriesCompileDetail(t *testing.T) {
	_, err := rulesvc.NormalizeRuleInput(rulesvc.RuleInput{RuleName: "Rule", Type: db.RuleTypeRegex, Value: "["})
	if err == nil || !strings.Contains(err.Error(), "error parsing regexp") {
		t.Fatalf("message = %v", err)
	}
}

func TestNormalizeRuleInputSkipsRegexCheckForKeyword(t *testing.T) {
	input, err := rulesvc.NormalizeRuleInput(rulesvc.RuleInput{RuleName: "Bracket", Type: db.RuleTypeKeyword, Value: "["})
	if err != nil {
		t.Fatalf("keyword rule rejected: %v", err)
	}

	if input.Value != "[" {
		t.Fatalf("value = %q", input.Value)
	}
}

func TestNormalizeRuleInputPreservesWhitespaceInRegex(t *testing.T) {
	value := `a  b\s{2}c`

	input, err := rulesvc.NormalizeRuleInput(rulesvc.RuleInput{RuleName: "Spacing", Type: db.RuleTypeRegex, Value: value})
	if err != nil {
		t.Fatalf("rulesvc.NormalizeRuleInput returned %v", err)
	}

	if input.Value != value {
		t.Fatalf("value = %q, want %q", input.Value, value)
	}
}
