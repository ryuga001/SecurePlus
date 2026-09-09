package rulematcher

import (
	"regexp"
	"slices"
	"strings"

	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

type Compiler struct {
	aggregator dto.Aggregator
	maxRules   int
}

func NewCompiler(aggregator dto.Aggregator, maxRules int) *Compiler {
	if maxRules <= 0 {
		maxRules = utils.MaxRules
	}

	return &Compiler{aggregator: aggregator, maxRules: maxRules}
}

func (c *Compiler) Build(set dto.PolicySet) (*dto.CompiledSet, error) {
	if len(set.Rules) > c.maxRules {
		return nil, utils.TooManyRules(len(set.Rules), c.maxRules)
	}

	compiled := dto.CompiledSet{
		CustomerID:   set.CustomerID,
		EmailUserID:  set.EmailUserID,
		PolicyCount:  len(set.Policies),
		Restrictions: c.aggregator.Aggregate(set.Policies),
	}

	patterns := make([]string, 0, len(set.Rules))
	types := make([]string, 0, 2)

	for _, rule := range set.Rules {
		switch rule.RuleType {
		case utils.MatcherKeyword:
			if strings.TrimSpace(rule.RuleValue) == "" {
				continue
			}

			normalized := Normalize(rule.RuleValue)

			compiled.Rules.Keywords = append(compiled.Rules.Keywords, dto.CompiledKeyword{
				Rule:       rule,
				Normalized: normalized,
			})
			patterns = append(patterns, normalized)

			if !slices.Contains(types, utils.MatcherKeyword) {
				types = append(types, utils.MatcherKeyword)
			}
		case utils.MatcherRegex:
			pattern, err := regexp.Compile(rule.RuleValue)
			if err != nil {
				continue
			}

			compiled.Rules.Regexes = append(compiled.Rules.Regexes, dto.CompiledRegex{
				Rule:    rule,
				Pattern: pattern,
			})

			if !slices.Contains(types, utils.MatcherRegex) {
				types = append(types, utils.MatcherRegex)
			}
		}
	}

	if len(patterns) > 0 {
		compiled.Rules.Automaton = BuildAutomaton(patterns)
	}

	compiled.RuleTypes = types

	return &compiled, nil
}
