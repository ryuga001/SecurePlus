package adjudication

import (
	"context"
	"log/slog"

	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

type BlockExecutor struct {
	notifier *BlockNoticeService
}

func NewBlockExecutor(notifier *BlockNoticeService) *BlockExecutor {
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

type logExecutor struct {
	action string
}

func NewLogExecutor(action string) logExecutor {
	return logExecutor{action: action}
}

func (e logExecutor) Action() string {
	return e.action
}

func (e logExecutor) Execute(ctx context.Context, request dto.ActionRequest) (dto.ActionResult, error) {
	return invoked(ctx, request, e.action)
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
