package actiontrigger

import (
	"context"
	"log/slog"

	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

type BlockExecutor struct{}

func NewBlockExecutor() *BlockExecutor {
	return &BlockExecutor{}
}

func (e *BlockExecutor) Action() string {
	return utils.ActionBlock
}

func (e *BlockExecutor) Execute(ctx context.Context, request dto.ActionRequest) (dto.ActionResult, error) {
	return invoked(ctx, request, utils.ActionBlock)
}

type QuarantineExecutor struct{}

func NewQuarantineExecutor() *QuarantineExecutor {
	return &QuarantineExecutor{}
}

func (e *QuarantineExecutor) Action() string {
	return utils.ActionQuarantine
}

func (e *QuarantineExecutor) Execute(ctx context.Context, request dto.ActionRequest) (dto.ActionResult, error) {
	return invoked(ctx, request, utils.ActionQuarantine)
}

type RedactExecutor struct{}

func NewRedactExecutor() *RedactExecutor {
	return &RedactExecutor{}
}

func (e *RedactExecutor) Action() string {
	return utils.ActionRedact
}

func (e *RedactExecutor) Execute(ctx context.Context, request dto.ActionRequest) (dto.ActionResult, error) {
	return invoked(ctx, request, utils.ActionRedact)
}

type AuditExecutor struct{}

func NewAuditExecutor() *AuditExecutor {
	return &AuditExecutor{}
}

func (e *AuditExecutor) Action() string {
	return utils.ActionAudit
}

func (e *AuditExecutor) Execute(ctx context.Context, request dto.ActionRequest) (dto.ActionResult, error) {
	return invoked(ctx, request, utils.ActionAudit)
}

func invoked(ctx context.Context, request dto.ActionRequest, action string) (dto.ActionResult, error) {
	slog.InfoContext(ctx, "policy action invoked",
		"correlation_id", request.CorrelationID,
		"customer_id", request.CustomerID,
		"action", action,
		"trigger", request.Trigger,
	)

	return dto.ActionResult{Action: action, Status: utils.ActionInvoked}, nil
}
