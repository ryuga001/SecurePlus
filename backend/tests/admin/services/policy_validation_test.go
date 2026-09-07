package services_test

import (
	"errors"
	"slices"
	"testing"

	policysvc "dpdp-backend/internal/admin/services/policy"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
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

func TestNormalizePolicyInputDefaultsActionToAudit(t *testing.T) {
	input, err := policysvc.NormalizePolicyInput(policysvc.PolicyInput{
		PolicyName: "Outbound",
		GroupIDs:   []int{1},
		RuleIDs:    []int{2},
	})
	if err != nil {
		t.Fatalf("NormalizePolicyInput returned %v", err)
	}

	if input.Action != db.ActionAudit {
		t.Fatalf("action = %q", input.Action)
	}
	if input.DomainRestriction.Mode != db.RestrictionNone || len(input.DomainRestriction.Values) != 0 {
		t.Fatalf("domain restriction = %+v", input.DomainRestriction)
	}
	if input.AttachmentRestriction.Mode != db.RestrictionNone {
		t.Fatalf("attachment restriction = %+v", input.AttachmentRestriction)
	}
}

func TestNormalizePolicyInputUppercasesAction(t *testing.T) {
	input, err := policysvc.NormalizePolicyInput(policysvc.PolicyInput{
		PolicyName: "Outbound",
		Action:     " quarantine ",
		GroupIDs:   []int{1},
		RuleIDs:    []int{2},
	})
	if err != nil {
		t.Fatalf("NormalizePolicyInput returned %v", err)
	}

	if input.Action != db.ActionQuarantine {
		t.Fatalf("action = %q", input.Action)
	}
}

func TestNormalizePolicyInputRejectsUnknownAction(t *testing.T) {
	_, err := policysvc.NormalizePolicyInput(policysvc.PolicyInput{
		PolicyName: "Outbound",
		Action:     "DELETE",
		GroupIDs:   []int{1},
		RuleIDs:    []int{2},
	})

	if !errors.Is(err, utils.ErrInvalidAction) {
		t.Fatalf("error = %v, want ErrInvalidAction", err)
	}
}

func TestNormalizePolicyInputNormalisesRestrictions(t *testing.T) {
	input, err := policysvc.NormalizePolicyInput(policysvc.PolicyInput{
		PolicyName: "Outbound",
		GroupIDs:   []int{1},
		RuleIDs:    []int{2},
		DomainRestriction: db.Restriction{
			Mode:   " block ",
			Values: []string{"  HTTPS://WWW.Partner.COM. ", "partner.com", "vendor.io"},
		},
		AttachmentRestriction: db.Restriction{
			Mode:   "allow",
			Values: []string{".PDF", "pdf", " docx "},
		},
	})
	if err != nil {
		t.Fatalf("NormalizePolicyInput returned %v", err)
	}

	if input.DomainRestriction.Mode != db.RestrictionBlock {
		t.Fatalf("domain mode = %q", input.DomainRestriction.Mode)
	}
	if !slices.Equal(input.DomainRestriction.Values, []string{"partner.com", "vendor.io"}) {
		t.Fatalf("domain values = %v", input.DomainRestriction.Values)
	}
	if input.AttachmentRestriction.Mode != db.RestrictionAllow {
		t.Fatalf("attachment mode = %q", input.AttachmentRestriction.Mode)
	}
	if !slices.Equal(input.AttachmentRestriction.Values, []string{"docx", "pdf"}) {
		t.Fatalf("attachment values = %v", input.AttachmentRestriction.Values)
	}
}

func TestNormalizePolicyInputRestrictionValidation(t *testing.T) {
	base := policysvc.PolicyInput{PolicyName: "Outbound", GroupIDs: []int{1}, RuleIDs: []int{2}}

	withDomain := base
	withDomain.DomainRestriction = db.Restriction{Mode: "SOMETHING", Values: []string{"partner.com"}}
	if _, err := policysvc.NormalizePolicyInput(withDomain); !errors.Is(err, utils.ErrInvalidRestrictionMode) {
		t.Fatalf("bad mode error = %v", err)
	}

	emptyList := base
	emptyList.DomainRestriction = db.Restriction{Mode: db.RestrictionBlock}
	if _, err := policysvc.NormalizePolicyInput(emptyList); !errors.Is(err, utils.ErrRestrictionValuesNeeded) {
		t.Fatalf("empty list error = %v", err)
	}

	badDomain := base
	badDomain.DomainRestriction = db.Restriction{Mode: db.RestrictionAllow, Values: []string{"not a domain"}}
	if _, err := policysvc.NormalizePolicyInput(badDomain); !errors.Is(err, utils.ErrInvalidRestrictionDomain) {
		t.Fatalf("bad domain error = %v", err)
	}

	badExtension := base
	badExtension.AttachmentRestriction = db.Restriction{Mode: db.RestrictionBlock, Values: []string{"my file.pdf"}}
	if _, err := policysvc.NormalizePolicyInput(badExtension); !errors.Is(err, utils.ErrInvalidRestrictionFileType) {
		t.Fatalf("bad extension error = %v", err)
	}
}

func TestNormalizePolicyInputDropsValuesWhenModeIsNone(t *testing.T) {
	input, err := policysvc.NormalizePolicyInput(policysvc.PolicyInput{
		PolicyName:        "Outbound",
		GroupIDs:          []int{1},
		RuleIDs:           []int{2},
		DomainRestriction: db.Restriction{Mode: db.RestrictionNone, Values: []string{"partner.com"}},
	})
	if err != nil {
		t.Fatalf("NormalizePolicyInput returned %v", err)
	}

	if len(input.DomainRestriction.Values) != 0 {
		t.Fatalf("values = %v, want empty", input.DomainRestriction.Values)
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
