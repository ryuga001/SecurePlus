package rulematcher_test

import (
	"testing"

	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	"dpdp-backend/internal/delivery/engine/rulematcher"
	"dpdp-backend/internal/delivery/utils"
)

func compiled(t *testing.T, rules ...dto.RuleRecord) dto.CompiledSet {
	t.Helper()

	set := dto.PolicySet{
		CustomerID: 1,
		Policies:   []dto.PolicyRecord{{PolicyID: 1, PolicyName: "Data Loss", Action: utils.ActionBlock}},
		Rules:      rules,
	}

	built, err := rulematcher.NewCompiler(stubAggregator{}, utils.MaxRules).Build(set)
	if err != nil {
		t.Fatalf("Build returned %v", err)
	}

	return *built
}

type stubAggregator struct{}

func (stubAggregator) Aggregate(_ []dto.PolicyRecord) dto.EffectiveRestrictions {
	return dto.EffectiveRestrictions{
		Domain:     dto.RestrictionSet{Blocked: map[string]dto.PolicyRef{}},
		Attachment: dto.RestrictionSet{Blocked: map[string]dto.PolicyRef{}},
	}
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

func TestKeywordMatcherIsCaseInsensitive(t *testing.T) {
	set := compiled(t, keywordRule(7, "Secrets", "confidential"))

	matches := rulematcher.NewKeywordMatcher().Match(dto.MatchInput{
		Parts: []dto.ContentPart{{Location: utils.LocationBody, Text: "This is CONFIDENTIAL"}},
	}, set.Rules)

	if len(matches) != 1 {
		t.Fatalf("matches = %v", matches)
	}
	if matches[0].RuleID != 7 || matches[0].Occurrences != 1 {
		t.Fatalf("match = %+v", matches[0])
	}
	if matches[0].ConfiguredValue != "confidential" || matches[0].RuleType != utils.MatcherKeyword {
		t.Fatalf("match = %+v", matches[0])
	}
	if matches[0].Action != utils.ActionQuarantine {
		t.Fatalf("action = %q", matches[0].Action)
	}
}

func TestKeywordMatcherCountsAcrossParts(t *testing.T) {
	set := compiled(t, keywordRule(7, "Secrets", "card"))

	matches := rulematcher.NewKeywordMatcher().Match(dto.MatchInput{
		Parts: []dto.ContentPart{
			{Location: utils.LocationSubject, Text: "card trouble"},
			{Location: utils.LocationBody, Text: "the card and the card"},
		},
	}, set.Rules)

	if len(matches) != 1 || matches[0].Occurrences != 3 {
		t.Fatalf("matches = %+v, want 3 occurrences", matches)
	}
	if len(matches[0].Locations) != 2 {
		t.Fatalf("locations = %v, want subject and body", matches[0].Locations)
	}
	if matches[0].Locations[0] != utils.LocationSubject || matches[0].Locations[1] != utils.LocationBody {
		t.Fatalf("locations = %v, want stable ordering", matches[0].Locations)
	}
}

func TestKeywordMatcherReportsEveryMatchingRule(t *testing.T) {
	set := compiled(t,
		keywordRule(1, "Cards", "card"),
		keywordRule(2, "Secrets", "secret"),
		keywordRule(3, "Unused", "nowhere"),
	)

	matches := rulematcher.NewKeywordMatcher().Match(dto.MatchInput{
		Parts: []dto.ContentPart{{Location: utils.LocationBody, Text: "card and secret"}},
	}, set.Rules)

	if len(matches) != 2 {
		t.Fatalf("matches = %+v, want two", matches)
	}
}

func TestKeywordMatcherReturnsNothingWithoutMatches(t *testing.T) {
	set := compiled(t, keywordRule(1, "Cards", "card"))

	matches := rulematcher.NewKeywordMatcher().Match(dto.MatchInput{
		Parts: []dto.ContentPart{{Location: utils.LocationBody, Text: "nothing here"}},
	}, set.Rules)

	if len(matches) != 0 {
		t.Fatalf("matches = %+v, want none", matches)
	}
}

func TestKeywordMatcherHandlesEmptyCompiledSet(t *testing.T) {
	matches := rulematcher.NewKeywordMatcher().Match(dto.MatchInput{
		Parts: []dto.ContentPart{{Location: utils.LocationBody, Text: "anything"}},
	}, dto.CompiledRules{})

	if matches != nil {
		t.Fatalf("matches = %+v, want nil", matches)
	}
}
