package inspection

import (
	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

type RegexMatcher struct {
	maxMatches int
}

func NewRegexMatcher() *RegexMatcher {
	return &RegexMatcher{maxMatches: utils.MaxMatchesPerRule}
}

func NewRegexMatcherWithLimit(maxMatches int) *RegexMatcher {
	if maxMatches <= 0 {
		maxMatches = utils.MaxMatchesPerRule
	}

	return &RegexMatcher{maxMatches: maxMatches}
}

func (m *RegexMatcher) Type() string {
	return utils.MatcherRegex
}

func (m *RegexMatcher) Match(input dto.MatchInput, compiled dto.CompiledRules) []dto.RuleMatch {
	if len(compiled.Regexes) == 0 {
		return nil
	}

	matches := make([]dto.RuleMatch, 0, len(compiled.Regexes))

	for _, expression := range compiled.Regexes {
		if expression.Pattern == nil {
			continue
		}

		occurrences := 0
		found := map[string]bool{}

		for _, part := range input.Parts {
			count := m.count(expression, part.Text)
			if count == 0 {
				continue
			}

			occurrences += count
			found[part.Location] = true
		}

		if occurrences == 0 {
			continue
		}

		matches = append(matches, dto.RuleMatch{
			PolicyID:        expression.Rule.PolicyID,
			PolicyName:      expression.Rule.PolicyName,
			Action:          expression.Rule.Action,
			RuleID:          expression.Rule.RuleID,
			RuleName:        expression.Rule.RuleName,
			RuleType:        utils.MatcherRegex,
			ConfiguredValue: expression.Rule.RuleValue,
			Occurrences:     occurrences,
			Locations:       ordered(found),
		})
	}

	return matches
}

func (m *RegexMatcher) count(expression dto.CompiledRegex, text string) int {
	if text == "" {
		return 0
	}

	spans := expression.Pattern.FindAllStringIndex(text, m.maxMatches)

	count := 0

	for _, span := range spans {
		if span[1] > span[0] {
			count++
		}
	}

	return count
}
