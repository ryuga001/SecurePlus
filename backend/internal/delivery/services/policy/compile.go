package policy

import (
	"log/slog"
	"regexp"
	"slices"
	"strings"

	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/services/inspection"
	"dpdp-backend/internal/delivery/utils"
)

type Compiler struct {
	maxRules int
}

func NewCompiler(maxRules int) *Compiler {
	if maxRules <= 0 {
		maxRules = utils.MaxRules
	}

	return &Compiler{maxRules: maxRules}
}

func (c *Compiler) Build(set dto.PolicySet) (*dto.CompiledSet, error) {
	if len(set.Rules) > c.maxRules {
		return nil, utils.TooManyRules(len(set.Rules), c.maxRules)
	}

	compiled := dto.CompiledSet{
		CustomerID:   set.CustomerID,
		EmailUserID:  set.EmailUserID,
		PolicyCount:  len(set.Policies),
		Restrictions: Aggregate(set.Policies),
	}

	patterns := make([]string, 0, len(set.Rules))
	types := make([]string, 0, 2)

	for _, rule := range set.Rules {
		switch rule.RuleType {
		case utils.MatcherKeyword:
			if strings.TrimSpace(rule.RuleValue) == "" {
				continue
			}

			normalized := inspection.Normalize(rule.RuleValue)

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
				slog.Warn("regex rule skipped, pattern does not compile",
					"policy_id", rule.PolicyID,
					"rule_id", rule.RuleID,
					"rule_name", rule.RuleName,
					"error", err,
				)

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
		compiled.Rules.Automaton = inspection.BuildAutomaton(patterns)
	}

	compiled.RuleTypes = types

	return &compiled, nil
}
