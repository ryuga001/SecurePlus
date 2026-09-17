package restriction

import (
	"dpdp-backend/internal/delivery"
	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

func EvaluateDomains(recipients []string, restrictions dto.EffectiveRestrictions) []dto.RestrictionViolation {
	violations := make([]dto.RestrictionViolation, 0)
	set := restrictions.Domain

	for _, recipient := range recipients {
		domain := delivery.DomainOf(recipient)

		if ref, blocked := set.Blocked[domain]; blocked {
			violations = append(violations, dto.RestrictionViolation{
				Kind:       utils.KindDomain,
				Mode:       utils.RestrictionBlock,
				Value:      domain,
				Recipient:  recipient,
				PolicyID:   ref.PolicyID,
				PolicyName: ref.PolicyName,
			})

			continue
		}

		if set.Allowed == nil {
			continue
		}

		if _, allowed := set.Allowed[domain]; allowed {
			continue
		}

		violations = append(violations, dto.RestrictionViolation{
			Kind:       utils.KindDomain,
			Mode:       utils.RestrictionAllow,
			Value:      domain,
			Recipient:  recipient,
			PolicyID:   anyRef(set.Allowed).PolicyID,
			PolicyName: anyRef(set.Allowed).PolicyName,
		})
	}

	return violations
}

func anyRef(set map[string]dto.PolicyRef) dto.PolicyRef {
	best := dto.PolicyRef{}

	for _, ref := range set {
		if best.PolicyID == 0 || ref.PolicyID < best.PolicyID {
			best = ref
		}
	}

	return best
}
