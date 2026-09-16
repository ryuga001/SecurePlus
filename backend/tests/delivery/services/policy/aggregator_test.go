package policy_test

import (
	"testing"

	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/services/policy"
	"dpdp-backend/internal/delivery/utils"
)

func policyRecord(id int, name, mode string, values ...string) dto.PolicyRecord {
	return dto.PolicyRecord{
		PolicyID:              id,
		PolicyName:            name,
		DomainRestriction:     dto.Restriction{Mode: mode, Values: values},
		AttachmentRestriction: dto.Restriction{Mode: utils.RestrictionNone, Values: []string{}},
	}
}

func keys(set map[string]dto.PolicyRef) []string {
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}

	return values
}

func TestBlocklistsAreUnioned(t *testing.T) {
	effective := policy.Aggregate([]dto.PolicyRecord{
		policyRecord(1, "A", utils.RestrictionBlock, "a.test"),
		policyRecord(2, "B", utils.RestrictionBlock, "b.test"),
		policyRecord(3, "C", utils.RestrictionBlock, "c.test", "a.test"),
	})

	if len(effective.Domain.Blocked) != 3 {
		t.Fatalf("blocked = %v, want three distinct domains", keys(effective.Domain.Blocked))
	}
	if effective.Domain.Blocked["a.test"].PolicyID != 1 {
		t.Fatalf("a.test attributed to policy %d, want the first contributor", effective.Domain.Blocked["a.test"].PolicyID)
	}
}

func TestAllowlistsAreIntersected(t *testing.T) {
	effective := policy.Aggregate([]dto.PolicyRecord{
		policyRecord(1, "A", utils.RestrictionAllow, "company.com", "partner.com"),
		policyRecord(2, "B", utils.RestrictionAllow, "company.com", "vendor.com"),
	})

	if len(effective.Domain.Allowed) != 1 {
		t.Fatalf("allowed = %v, want only company.com", keys(effective.Domain.Allowed))
	}
	if _, ok := effective.Domain.Allowed["company.com"]; !ok {
		t.Fatalf("allowed = %v, want company.com", keys(effective.Domain.Allowed))
	}
}

func TestEmptyAllowlistDoesNotEmptyTheIntersection(t *testing.T) {
	effective := policy.Aggregate([]dto.PolicyRecord{
		policyRecord(1, "A", utils.RestrictionAllow, "company.com", "partner.com"),
		policyRecord(2, "B", utils.RestrictionNone),
		policyRecord(3, "C", utils.RestrictionAllow, "company.com", "vendor.com"),
	})

	if len(effective.Domain.Allowed) != 1 {
		t.Fatalf("allowed = %v, an abstaining policy must not empty the set", keys(effective.Domain.Allowed))
	}
}

func TestNoAllowlistMeansNoRestriction(t *testing.T) {
	effective := policy.Aggregate([]dto.PolicyRecord{
		policyRecord(1, "A", utils.RestrictionBlock, "a.test"),
		policyRecord(2, "B", utils.RestrictionNone),
	})

	if effective.Domain.Allowed != nil {
		t.Fatalf("allowed = %v, want nil so no allowlist restriction applies", keys(effective.Domain.Allowed))
	}
}

func TestAggregationOfNoPoliciesImposesNothing(t *testing.T) {
	effective := policy.Aggregate(nil)

	if len(effective.Domain.Blocked) != 0 || effective.Domain.Allowed != nil {
		t.Fatalf("effective = %+v, want empty", effective.Domain)
	}
}

func TestAttachmentRestrictionsAggregateIndependently(t *testing.T) {
	first := dto.PolicyRecord{
		PolicyID:              1,
		DomainRestriction:     dto.Restriction{Mode: utils.RestrictionBlock, Values: []string{"a.test"}},
		AttachmentRestriction: dto.Restriction{Mode: utils.RestrictionBlock, Values: []string{"exe"}},
	}
	second := dto.PolicyRecord{
		PolicyID:              2,
		DomainRestriction:     dto.Restriction{Mode: utils.RestrictionNone, Values: []string{}},
		AttachmentRestriction: dto.Restriction{Mode: utils.RestrictionBlock, Values: []string{"zip"}},
	}

	effective := policy.Aggregate([]dto.PolicyRecord{first, second})

	if len(effective.Attachment.Blocked) != 2 {
		t.Fatalf("blocked attachments = %v", keys(effective.Attachment.Blocked))
	}
	if len(effective.Domain.Blocked) != 1 {
		t.Fatalf("blocked domains = %v", keys(effective.Domain.Blocked))
	}
}
