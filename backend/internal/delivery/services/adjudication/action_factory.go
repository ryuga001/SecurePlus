package adjudication

import (
	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

type ActionFactory struct {
	executors map[string]dto.ActionExecutor
}

func NewActionFactory(executors ...dto.ActionExecutor) *ActionFactory {
	registry := make(map[string]dto.ActionExecutor, len(executors))

	for _, executor := range executors {
		registry[executor.Action()] = executor
	}

	return &ActionFactory{executors: registry}
}

func DefaultActionFactory(notifier *BlockNoticeService) *ActionFactory {
	return NewActionFactory(
		NewBlockExecutor(notifier),
		NewLogExecutor(utils.ActionQuarantine),
		NewLogExecutor(utils.ActionRedact),
		NewLogExecutor(utils.ActionAudit),
	)
}

func (f *ActionFactory) For(action string) (dto.ActionExecutor, bool) {
	if action == utils.ActionNone {
		return nil, false
	}

	executor, ok := f.executors[action]

	return executor, ok
}
