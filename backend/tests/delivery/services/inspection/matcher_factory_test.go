package inspection_test

import (
	"testing"

	"dpdp-backend/internal/delivery/services/inspection"
	"dpdp-backend/internal/delivery/utils"
)

func TestFactoryResolvesEveryRuleType(t *testing.T) {
	factory := inspection.DefaultMatcherFactory()

	for _, ruleType := range []string{utils.MatcherKeyword, utils.MatcherRegex} {
		matcher, ok := factory.For(ruleType)
		if !ok {
			t.Fatalf("no matcher registered for %q", ruleType)
		}
		if matcher.Type() != ruleType {
			t.Fatalf("matcher for %q reports %q", ruleType, matcher.Type())
		}
	}
}

func TestFactoryReportsUnknownRuleType(t *testing.T) {
	matcher, ok := inspection.DefaultMatcherFactory().For("URL")

	if ok || matcher != nil {
		t.Fatalf("For returned (%v, %v), want (nil, false)", matcher, ok)
	}
}
