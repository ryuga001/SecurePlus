package inspection

import dto "dpdp-backend/internal/delivery/dto/evaluation"

type MatcherFactory struct {
	matchers map[string]dto.RuleMatcher
}

func NewMatcherFactory(matchers ...dto.RuleMatcher) *MatcherFactory {
	registry := make(map[string]dto.RuleMatcher, len(matchers))

	for _, matcher := range matchers {
		registry[matcher.Type()] = matcher
	}

	return &MatcherFactory{matchers: registry}
}

func DefaultMatcherFactory() *MatcherFactory {
	return NewMatcherFactory(NewKeywordMatcher(), NewRegexMatcher())
}

func (f *MatcherFactory) For(ruleType string) (dto.RuleMatcher, bool) {
	matcher, ok := f.matchers[ruleType]

	return matcher, ok
}
