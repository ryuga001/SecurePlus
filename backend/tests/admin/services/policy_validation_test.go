package services_test

import (
	"errors"
	"slices"
	"testing"

	policysvc "dpdp-backend/internal/admin/services/policy"
	"dpdp-backend/internal/admin/utils"
)

func TestNormalizePolicyInputCollapsesName(t *testing.T) {
	input, err := policysvc.NormalizePolicyInput(policysvc.PolicyInput{
		PolicyName: "  Outbound   PII  ",
		GroupIDs:   []int{1},
		RuleIDs:    []int{2},
	})
	if err != nil {
		t.Fatalf("policysvc.NormalizePolicyInput returned %v", err)
	}

	if input.PolicyName != "Outbound PII" {
		t.Fatalf("policy name = %q", input.PolicyName)
	}
}

func TestNormalizePolicyInputRequiresName(t *testing.T) {
	_, err := policysvc.NormalizePolicyInput(policysvc.PolicyInput{
		PolicyName: "   ",
		GroupIDs:   []int{1},
		RuleIDs:    []int{2},
	})

	if !errors.Is(err, utils.ErrPolicyNameNeeded) {
		t.Fatalf("error = %v, want ErrPolicyNameNeeded", err)
	}
}

func TestNormalizePolicyInputRequiresGroups(t *testing.T) {
	_, err := policysvc.NormalizePolicyInput(policysvc.PolicyInput{
		PolicyName: "Outbound",
		GroupIDs:   nil,
		RuleIDs:    []int{2},
	})

	if !errors.Is(err, utils.ErrGroupsNeeded) {
		t.Fatalf("error = %v, want ErrGroupsNeeded", err)
	}
	if err.Error() != utils.MsgGroupsNeeded {
		t.Fatalf("message = %q", err.Error())
	}
}

func TestNormalizePolicyInputRequiresRules(t *testing.T) {
	_, err := policysvc.NormalizePolicyInput(policysvc.PolicyInput{
		PolicyName: "Outbound",
		GroupIDs:   []int{1},
		RuleIDs:    []int{},
	})

	if !errors.Is(err, utils.ErrRulesNeeded) {
		t.Fatalf("error = %v, want ErrRulesNeeded", err)
	}
	if err.Error() != utils.MsgRulesNeeded {
		t.Fatalf("message = %q", err.Error())
	}
}

func TestNormalizePolicyInputDedupesAndSortsIDs(t *testing.T) {
	input, err := policysvc.NormalizePolicyInput(policysvc.PolicyInput{
		PolicyName: "Outbound",
		GroupIDs:   []int{3, 1, 3, 0, -2, 1},
		RuleIDs:    []int{9, 9},
	})
	if err != nil {
		t.Fatalf("policysvc.NormalizePolicyInput returned %v", err)
	}

	if !slices.Equal(input.GroupIDs, []int{1, 3}) {
		t.Fatalf("group ids = %v", input.GroupIDs)
	}
	if !slices.Equal(input.RuleIDs, []int{9}) {
		t.Fatalf("rule ids = %v", input.RuleIDs)
	}
}

func TestNormalizePolicyInputRejectsOversizedBatches(t *testing.T) {
	ids := make([]int, 0, utils.MaxBatchIDs+1)
	for index := 1; index <= utils.MaxBatchIDs+1; index++ {
		ids = append(ids, index)
	}

	_, err := policysvc.NormalizePolicyInput(policysvc.PolicyInput{PolicyName: "Outbound", GroupIDs: ids, RuleIDs: []int{1}})
	if !errors.Is(err, utils.ErrTooManyItems) {
		t.Fatalf("error = %v, want ErrTooManyItems", err)
	}
}

func TestNormalizePolicyInputKeepsInactive(t *testing.T) {
	input, err := policysvc.NormalizePolicyInput(policysvc.PolicyInput{
		PolicyName: "Outbound",
		Active:     false,
		GroupIDs:   []int{1},
		RuleIDs:    []int{2},
	})
	if err != nil {
		t.Fatalf("policysvc.NormalizePolicyInput returned %v", err)
	}

	if input.Active {
		t.Fatal("active must survive as false")
	}
}
