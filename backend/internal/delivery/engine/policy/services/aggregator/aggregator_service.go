package aggregator

import (
	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

type Aggregator struct{}

func NewAggregator() *Aggregator {
	return &Aggregator{}
}

func (a *Aggregator) Aggregate(policies []dto.PolicyRecord) dto.EffectiveRestrictions {
	domain := make([]dto.Restriction, 0, len(policies))
	attachment := make([]dto.Restriction, 0, len(policies))
	refs := make([]dto.PolicyRef, 0, len(policies))

	for _, policy := range policies {
		domain = append(domain, policy.DomainRestriction)
		attachment = append(attachment, policy.AttachmentRestriction)
		refs = append(refs, dto.PolicyRef{PolicyID: policy.PolicyID, PolicyName: policy.PolicyName})
	}

	return dto.EffectiveRestrictions{
		Domain:     combine(domain, refs),
		Attachment: combine(attachment, refs),
	}
}

func combine(restrictions []dto.Restriction, refs []dto.PolicyRef) dto.RestrictionSet {
	set := dto.RestrictionSet{Blocked: map[string]dto.PolicyRef{}}

	for index, restriction := range restrictions {
		if restriction.Mode != utils.RestrictionBlock {
			continue
		}

		for _, value := range restriction.Values {
			if _, seen := set.Blocked[value]; !seen {
				set.Blocked[value] = refs[index]
			}
		}
	}

	for index, restriction := range restrictions {
		if restriction.Mode != utils.RestrictionAllow || len(restriction.Values) == 0 {
			continue
		}

		contributed := map[string]dto.PolicyRef{}
		for _, value := range restriction.Values {
			contributed[value] = refs[index]
		}

		if set.Allowed == nil {
			set.Allowed = contributed
			continue
		}

		set.Allowed = intersect(set.Allowed, contributed)
	}

	return set
}

func intersect(current, incoming map[string]dto.PolicyRef) map[string]dto.PolicyRef {
	result := make(map[string]dto.PolicyRef, len(current))

	for value, ref := range current {
		if _, ok := incoming[value]; ok {
			result[value] = ref
		}
	}

	return result
}
