package actiontrigger_test

import (
	"context"
	"testing"

	"dpdp-backend/internal/delivery/engine/actiontrigger"
	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

func match(action string) dto.RuleMatch {
	return dto.RuleMatch{PolicyID: 1, RuleID: 1, Action: action, Occurrences: 1}
}

func TestHighestPriorityActionWins(t *testing.T) {
	got := actiontrigger.NewActionResolver().Resolve([]dto.RuleMatch{
		match(utils.ActionAudit),
		match(utils.ActionQuarantine),
		match(utils.ActionBlock),
	})

	if got != utils.ActionBlock {
		t.Fatalf("action = %q, want BLOCK", got)
	}
}

func TestRedactBeatsAudit(t *testing.T) {
	got := actiontrigger.NewActionResolver().Resolve([]dto.RuleMatch{
		match(utils.ActionAudit),
		match(utils.ActionRedact),
	})

	if got != utils.ActionRedact {
		t.Fatalf("action = %q, want REDACT", got)
	}
}

func TestNoMatchesResolvesToNone(t *testing.T) {
	if got := actiontrigger.NewActionResolver().Resolve(nil); got != utils.ActionNone {
		t.Fatalf("action = %q, want NONE", got)
	}
}

func TestMatchOrderDoesNotChangeTheResult(t *testing.T) {
	orderings := [][]dto.RuleMatch{
		{match(utils.ActionBlock), match(utils.ActionAudit), match(utils.ActionQuarantine)},
		{match(utils.ActionAudit), match(utils.ActionBlock), match(utils.ActionQuarantine)},
		{match(utils.ActionQuarantine), match(utils.ActionAudit), match(utils.ActionBlock)},
		{match(utils.ActionAudit), match(utils.ActionQuarantine), match(utils.ActionBlock)},
	}

	for index, matches := range orderings {
		if got := actiontrigger.NewActionResolver().Resolve(matches); got != utils.ActionBlock {
			t.Fatalf("ordering %d resolved to %q, want BLOCK", index, got)
		}
	}
}

func TestUnknownActionRanksBelowEverything(t *testing.T) {
	got := actiontrigger.NewActionResolver().Resolve([]dto.RuleMatch{
		match("SOMETHING_ELSE"),
		match(utils.ActionAudit),
	})

	if got != utils.ActionAudit {
		t.Fatalf("action = %q, want AUDIT", got)
	}
}

func TestFactoryResolvesEveryAction(t *testing.T) {
	factory := actiontrigger.DefaultActionFactory(nil)

	for _, action := range []string{utils.ActionBlock, utils.ActionQuarantine, utils.ActionRedact, utils.ActionAudit} {
		executor, ok := factory.For(action)
		if !ok {
			t.Fatalf("no executor registered for %q", action)
		}
		if executor.Action() != action {
			t.Fatalf("executor for %q reports %q", action, executor.Action())
		}
	}
}

func TestFactoryHasNoExecutorForNone(t *testing.T) {
	if executor, ok := actiontrigger.DefaultActionFactory(nil).For(utils.ActionNone); ok || executor != nil {
		t.Fatalf("For(NONE) = (%v, %v), want (nil, false)", executor, ok)
	}
}

func TestFactoryReportsUnknownAction(t *testing.T) {
	if executor, ok := actiontrigger.DefaultActionFactory(nil).For("SHRED"); ok || executor != nil {
		t.Fatalf("For(SHRED) = (%v, %v), want (nil, false)", executor, ok)
	}
}

func TestStubExecutorsReportInvokedWithoutActing(t *testing.T) {
	factory := actiontrigger.DefaultActionFactory(nil)

	for _, action := range []string{utils.ActionBlock, utils.ActionQuarantine, utils.ActionRedact, utils.ActionAudit} {
		executor, _ := factory.For(action)

		result, err := executor.Execute(context.Background(), dto.ActionRequest{
			CorrelationID: "abc",
			CustomerID:    1,
			Action:        action,
		})
		if err != nil {
			t.Fatalf("%s executor returned %v", action, err)
		}

		if result.Action != action || result.Status != utils.ActionInvoked {
			t.Fatalf("%s result = %+v", action, result)
		}
	}
}
