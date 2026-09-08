package deliveryaudit

import (
	"context"
	"log/slog"
	"time"

	dto "dpdp-backend/internal/audit/dto/deliveryaudit"
	repo "dpdp-backend/internal/audit/repositories/deliveryaudit"
)

const (
	completeAttempts = 3
	completeBackoff  = time.Second
)

type DeliveryAuditService struct {
	repo *repo.DeliveryAuditRepository
}

func NewDeliveryAuditService(repo *repo.DeliveryAuditRepository) *DeliveryAuditService {
	return &DeliveryAuditService{repo: repo}
}

func (s *DeliveryAuditService) EnsureIndexes(ctx context.Context) error {
	return s.repo.EnsureIndexes(ctx)
}

func (s *DeliveryAuditService) Create(ctx context.Context, record dto.Record) error {
	return s.repo.Insert(ctx, record)
}

func (s *DeliveryAuditService) RecordAttempt(ctx context.Context, correlationID string, attempt dto.Attempt) error {
	return s.repo.PushAttempt(ctx, correlationID, attempt)
}

func (s *DeliveryAuditService) Complete(ctx context.Context, correlationID string, result dto.Result) error {
	var err error

	for attempt := 1; attempt <= completeAttempts; attempt++ {
		if err = s.repo.ApplyResult(ctx, correlationID, result); err == nil {
			return nil
		}

		if ctx.Err() != nil {
			break
		}

		if attempt < completeAttempts {
			time.Sleep(time.Duration(attempt) * completeBackoff)
		}
	}

	slog.ErrorContext(ctx, "delivery audit completion lost",
		"correlation_id", correlationID,
		"status", result.Status,
		"error", err,
	)

	return err
}
