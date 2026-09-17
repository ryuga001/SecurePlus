package recording

import (
	"dpdp-backend/internal/delivery"
	evaluation "dpdp-backend/internal/delivery/dto/evaluation"
)

type GenerationInput struct {
	Message delivery.EmailMessage
	Result  evaluation.EvaluationResult
}
