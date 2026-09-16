package inspection

import (
	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

type KeywordMatcher struct{}

func NewKeywordMatcher() *KeywordMatcher {
	return &KeywordMatcher{}
}

func (m *KeywordMatcher) Type() string {
	return utils.MatcherKeyword
}

func (m *KeywordMatcher) Match(input dto.MatchInput, compiled dto.CompiledRules) []dto.RuleMatch {
	if compiled.Automaton == nil || len(compiled.Keywords) == 0 {
		return nil
	}

	occurrences := make([]int, len(compiled.Keywords))
	locations := make([]map[string]bool, len(compiled.Keywords))

	for _, part := range input.Parts {
		for _, hit := range compiled.Automaton.Find(Normalize(part.Text)) {
			if hit.Index < 0 || hit.Index >= len(compiled.Keywords) {
				continue
			}

			occurrences[hit.Index]++

			if locations[hit.Index] == nil {
				locations[hit.Index] = map[string]bool{}
			}

			locations[hit.Index][part.Location] = true
		}
	}

	matches := make([]dto.RuleMatch, 0)

	for index, count := range occurrences {
		if count == 0 {
			continue
		}

		keyword := compiled.Keywords[index]

		matches = append(matches, dto.RuleMatch{
			PolicyID:        keyword.Rule.PolicyID,
			PolicyName:      keyword.Rule.PolicyName,
			Action:          keyword.Rule.Action,
			RuleID:          keyword.Rule.RuleID,
			RuleName:        keyword.Rule.RuleName,
			RuleType:        utils.MatcherKeyword,
			ConfiguredValue: keyword.Rule.RuleValue,
			Occurrences:     count,
			Locations:       ordered(locations[index]),
		})
	}

	return matches
}

func ordered(found map[string]bool) []string {
	locations := make([]string, 0, len(found))

	for _, location := range []string{utils.LocationSubject, utils.LocationBody} {
		if found[location] {
			locations = append(locations, location)
		}
	}

	return locations
}
