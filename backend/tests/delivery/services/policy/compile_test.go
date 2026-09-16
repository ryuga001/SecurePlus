package policy_test

import (
	"errors"
	"testing"

	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/services/policy"
	"dpdp-backend/internal/delivery/utils"
)

func compiled(t *testing.T, rules ...dto.RuleRecord) dto.CompiledSet {
	t.Helper()

	built, err := policy.NewCompiler(utils.MaxRules).Build(dto.PolicySet{
		CustomerID: 1,
		Policies:   []dto.PolicyRecord{{PolicyID: 1, PolicyName: "Data Loss", Action: utils.ActionBlock}},
		Rules:      rules,
	})
	if err != nil {
		t.Fatalf("Build returned %v", err)
	}

	return *built
}

func keywordRule(id int, name, value string) dto.RuleRecord {
	return dto.RuleRecord{
		PolicyID:   1,
		PolicyName: "Data Loss",
		Action:     utils.ActionQuarantine,
		RuleID:     id,
		RuleName:   name,
		RuleType:   utils.MatcherKeyword,
		RuleValue:  value,
	}
}

func TestCompilerRecordsRuleTypesPresent(t *testing.T) {
	set := compiled(t,
		keywordRule(1, "Cards", "card"),
		dto.RuleRecord{PolicyID: 1, RuleID: 2, RuleType: utils.MatcherRegex, RuleValue: `\d+`},
	)

	if len(set.RuleTypes) != 2 {
		t.Fatalf("rule types = %v, want keyword and regex", set.RuleTypes)
	}
}

func TestCompilerSkipsUncompilableRegex(t *testing.T) {
	set := compiled(t, dto.RuleRecord{PolicyID: 1, RuleID: 2, RuleType: utils.MatcherRegex, RuleValue: "("})

	if len(set.Rules.Regexes) != 0 {
		t.Fatalf("regexes = %v, want the invalid pattern skipped", set.Rules.Regexes)
	}
}

func TestCompilerSkipsBlankKeyword(t *testing.T) {
	set := compiled(t, keywordRule(1, "Blank", "   "))

	if len(set.Rules.Keywords) != 0 {
		t.Fatalf("keywords = %v, want the blank keyword skipped", set.Rules.Keywords)
	}
	if set.Rules.Automaton != nil {
		t.Fatal("no automaton should be built without keywords")
	}
}

func TestCompilerRejectsOversizedPolicySet(t *testing.T) {
	rules := make([]dto.RuleRecord, 0, 5)
	for index := range 5 {
		rules = append(rules, keywordRule(index+1, "Rule", "value"))
	}

	_, err := policy.NewCompiler(4).Build(dto.PolicySet{CustomerID: 1, Rules: rules})

	if !errors.Is(err, utils.ErrTooManyRules) {
		t.Fatalf("error = %v, want ErrTooManyRules", err)
	}
}

func TestCompilerCountsPolicies(t *testing.T) {
	set, err := policy.NewCompiler(utils.MaxRules).Build(dto.PolicySet{
		CustomerID: 1,
		Policies: []dto.PolicyRecord{
			{PolicyID: 1, PolicyName: "One"},
			{PolicyID: 2, PolicyName: "Two"},
		},
	})
	if err != nil {
		t.Fatalf("Build returned %v", err)
	}

	if set.PolicyCount != 2 {
		t.Fatalf("policy count = %d, want 2", set.PolicyCount)
	}
}
