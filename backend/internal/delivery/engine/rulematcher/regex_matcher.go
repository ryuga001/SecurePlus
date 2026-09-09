package rulematcher

import (
	"log/slog"

	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

type RegexMatcher struct{}

func NewRegexMatcher() *RegexMatcher {
	return &RegexMatcher{}
}

func (m *RegexMatcher) Type() string {
	return utils.MatcherRegex
}

func (m *RegexMatcher) Match(input dto.MatchInput, compiled dto.CompiledRules) []dto.RuleMatch {
	if len(compiled.Regexes) == 0 {
		return nil
	}

	slog.Warn("regex rule evaluation is not implemented, rules were skipped",
		"rules", len(compiled.Regexes),
	)

	return nil
}
