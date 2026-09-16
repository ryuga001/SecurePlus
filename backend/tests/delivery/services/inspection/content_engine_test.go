package inspection_test

import (
	"testing"

	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/services/inspection"
	"dpdp-backend/internal/delivery/utils"
)

type spyMatcher struct {
	kind    string
	calls   int
	matches []dto.RuleMatch
}

func (s *spyMatcher) Type() string { return s.kind }

func (s *spyMatcher) Match(_ dto.MatchInput, _ dto.CompiledRules) []dto.RuleMatch {
	s.calls++

	return s.matches
}

func input() dto.MatchInput {
	return dto.MatchInput{Parts: []dto.ContentPart{{Location: utils.LocationBody, Text: "anything"}}}
}

func TestEveryRegisteredRuleTypeIsInvoked(t *testing.T) {
	keyword := &spyMatcher{kind: utils.MatcherKeyword, matches: []dto.RuleMatch{{RuleID: 1}}}
	regex := &spyMatcher{kind: utils.MatcherRegex}

	engine := inspection.NewContentEngine(inspection.NewMatcherFactory(keyword, regex))

	matches := engine.Evaluate(input(), dto.CompiledSet{
		RuleTypes: []string{utils.MatcherKeyword, utils.MatcherRegex},
	})

	if keyword.calls != 1 || regex.calls != 1 {
		t.Fatalf("calls: keyword=%d regex=%d, both must be invoked", keyword.calls, regex.calls)
	}
	if len(matches) != 1 {
		t.Fatalf("matches = %+v", matches)
	}
}

func TestMatchesFromEveryMatcherAreCollected(t *testing.T) {
	engine := inspection.NewContentEngine(inspection.NewMatcherFactory(
		&spyMatcher{kind: utils.MatcherKeyword, matches: []dto.RuleMatch{{RuleID: 1}, {RuleID: 2}}},
		&spyMatcher{kind: utils.MatcherRegex, matches: []dto.RuleMatch{{RuleID: 3}}},
	))

	matches := engine.Evaluate(input(), dto.CompiledSet{
		RuleTypes: []string{utils.MatcherKeyword, utils.MatcherRegex},
	})

	if len(matches) != 3 {
		t.Fatalf("matches = %+v, want all three collected", matches)
	}
}

func TestUnregisteredRuleTypeIsSkippedWithoutPanicking(t *testing.T) {
	engine := inspection.NewContentEngine(inspection.NewMatcherFactory())

	matches := engine.Evaluate(input(), dto.CompiledSet{RuleTypes: []string{"URL"}})

	if len(matches) != 0 {
		t.Fatalf("matches = %+v, want none", matches)
	}
}

func TestNoContentPartsSkipsEveryMatcher(t *testing.T) {
	keyword := &spyMatcher{kind: utils.MatcherKeyword, matches: []dto.RuleMatch{{RuleID: 1}}}

	engine := inspection.NewContentEngine(inspection.NewMatcherFactory(keyword))

	matches := engine.Evaluate(dto.MatchInput{}, dto.CompiledSet{RuleTypes: []string{utils.MatcherKeyword}})

	if keyword.calls != 0 {
		t.Fatalf("matcher was invoked %d times for an empty message", keyword.calls)
	}
	if len(matches) != 0 {
		t.Fatalf("matches = %+v, want none", matches)
	}
}
