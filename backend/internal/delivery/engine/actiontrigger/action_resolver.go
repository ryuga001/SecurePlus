package actiontrigger

import (
	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

type ActionResolver struct{}

func NewActionResolver() *ActionResolver {
	return &ActionResolver{}
}

func (r *ActionResolver) Resolve(matches []dto.RuleMatch) string {
	effective := utils.ActionNone

	for _, match := range matches {
		if utils.ActionPriority(match.Action) > utils.ActionPriority(effective) {
			effective = match.Action
		}
	}

	return effective
}
