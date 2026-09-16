package adjudication

import (
	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

func ResolveAction(matches []dto.RuleMatch) string {
	effective := utils.ActionNone

	for _, match := range matches {
		if utils.ActionPriority(match.Action) > utils.ActionPriority(effective) {
			effective = match.Action
		}
	}

	return effective
}
