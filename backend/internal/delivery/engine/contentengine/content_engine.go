package contentengine

import dto "dpdp-backend/internal/delivery/engine/dto/evaluation"

type ContentEngine struct {
	factory dto.MatcherFactory
}

func NewContentEngine(factory dto.MatcherFactory) *ContentEngine {
	return &ContentEngine{factory: factory}
}

func (e *ContentEngine) Evaluate(input dto.MatchInput, compiled dto.CompiledSet) []dto.RuleMatch {
	matches := make([]dto.RuleMatch, 0)

	if len(input.Parts) == 0 {
		return matches
	}

	for _, ruleType := range compiled.RuleTypes {
		matcher, ok := e.factory.For(ruleType)
		if !ok {
			continue
		}

		matches = append(matches, matcher.Match(input, compiled.Rules)...)
	}

	return matches
}
