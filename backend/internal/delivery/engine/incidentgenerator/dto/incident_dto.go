package dto

import (
	"context"

	auditdto "dpdp-backend/internal/audit/dto/emailincident"
	evaluation "dpdp-backend/internal/delivery/engine/dto/evaluation"
)

type GenerationInput struct {
	Message evaluation.MessageContext
	Result  evaluation.EvaluationResult
}

type AuditWriter interface {
	Record(ctx context.Context, record auditdto.Record) error
	UpdateAction(ctx context.Context, correlationID string, outcome auditdto.ActionOutcome) error
}
