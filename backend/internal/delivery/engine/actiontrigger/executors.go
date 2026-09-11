package actiontrigger

import (
	"context"
	"log/slog"

	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

type BlockExecutor struct {
	notifier dto.BlockNotifier
}

func NewBlockExecutor(notifier dto.BlockNotifier) *BlockExecutor {
	return &BlockExecutor{notifier: notifier}
}

func (e *BlockExecutor) Action() string {
	return utils.ActionBlock
}

func (e *BlockExecutor) Execute(ctx context.Context, request dto.ActionRequest) (dto.ActionResult, error) {
	result, err := invoked(ctx, request, utils.ActionBlock)
	if err != nil {
		return result, err
	}

	if e.notifier == nil {
		return result, nil
	}

	if err := e.notifier.Notify(context.WithoutCancel(ctx), request); err != nil {
		slog.ErrorContext(ctx, "block notice not sent",
			"correlation_id", request.CorrelationID,
			"customer_id", request.CustomerID,
			"error", err,
		)

		return dto.ActionResult{
			Action: utils.ActionBlock,
			Status: utils.ActionFailed,
			Error:  err.Error(),
		}, nil
	}

	return result, nil
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
