package restrictionevalutor_test

import (
	"testing"

	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	"dpdp-backend/internal/delivery/engine/restrictionevalutor"
	"dpdp-backend/internal/delivery/utils"
)

func restrictions(blocked map[string]dto.PolicyRef, allowed map[string]dto.PolicyRef) dto.EffectiveRestrictions {
	if blocked == nil {
		blocked = map[string]dto.PolicyRef{}
	}

	return dto.EffectiveRestrictions{
		Domain:     dto.RestrictionSet{Blocked: blocked, Allowed: allowed},
		Attachment: dto.RestrictionSet{Blocked: map[string]dto.PolicyRef{}},
	}
}

func ref(id int, name string) dto.PolicyRef {
	return dto.PolicyRef{PolicyID: id, PolicyName: name}
}

func TestBlockedDomainIsAViolation(t *testing.T) {
	found := restrictionevalutor.NewDomainEvaluator().Evaluate(
		[]string{"bob@blocked.test"},
		restrictions(map[string]dto.PolicyRef{"blocked.test": ref(4, "No Competitors")}, nil),
	)

	if len(found) != 1 {
		t.Fatalf("violations = %+v", found)
	}
	if found[0].Recipient != "bob@blocked.test" || found[0].PolicyName != "No Competitors" {
		t.Fatalf("violation = %+v", found[0])
	}
	if found[0].Kind != utils.KindDomain || found[0].Mode != utils.RestrictionBlock {
		t.Fatalf("violation = %+v", found[0])
	}
}

func TestDomainOutsideAllowlistIsAViolation(t *testing.T) {
	found := restrictionevalutor.NewDomainEvaluator().Evaluate(
		[]string{"bob@other.test"},
		restrictions(nil, map[string]dto.PolicyRef{"allowed.test": ref(2, "Partners Only")}),
	)

	if len(found) != 1 || found[0].Mode != utils.RestrictionAllow {
		t.Fatalf("violations = %+v", found)
	}
}

func TestAllowedDomainPasses(t *testing.T) {
	found := restrictionevalutor.NewDomainEvaluator().Evaluate(
		[]string{"bob@allowed.test"},
		restrictions(nil, map[string]dto.PolicyRef{"allowed.test": ref(2, "Partners Only")}),
	)

	if len(found) != 0 {
		t.Fatalf("violations = %+v, want none", found)
	}
}

func TestBlocklistTakesPrecedenceOverAllowlist(t *testing.T) {
	found := restrictionevalutor.NewDomainEvaluator().Evaluate(
		[]string{"bob@both.test"},
		restrictions(
			map[string]dto.PolicyRef{"both.test": ref(1, "Blocker")},
			map[string]dto.PolicyRef{"both.test": ref(2, "Allower")},
		),
	)

	if len(found) != 1 || found[0].Mode != utils.RestrictionBlock {
		t.Fatalf("violations = %+v, blocklist must win", found)
	}
}

func TestNoRestrictionsMeansNoViolations(t *testing.T) {
	found := restrictionevalutor.NewDomainEvaluator().Evaluate(
		[]string{"bob@anything.test", "eve@elsewhere.test"},
		restrictions(nil, nil),
	)

	if len(found) != 0 {
		t.Fatalf("violations = %+v, want none", found)
	}
}

func TestOnlyOffendingRecipientsAreReported(t *testing.T) {
	found := restrictionevalutor.NewDomainEvaluator().Evaluate(
		[]string{"a@ok.test", "b@blocked.test", "c@ok.test"},
		restrictions(map[string]dto.PolicyRef{"blocked.test": ref(1, "Blocker")}, nil),
	)

	if len(found) != 1 || found[0].Recipient != "b@blocked.test" {
		t.Fatalf("violations = %+v, want only the blocked recipient", found)
	}
}

func TestEveryRecipientCanBeReported(t *testing.T) {
	found := restrictionevalutor.NewDomainEvaluator().Evaluate(
		[]string{"a@blocked.test", "b@blocked.test"},
		restrictions(map[string]dto.PolicyRef{"blocked.test": ref(1, "Blocker")}, nil),
	)

	if len(found) != 2 {
		t.Fatalf("violations = %+v, want both", found)
	}
}
