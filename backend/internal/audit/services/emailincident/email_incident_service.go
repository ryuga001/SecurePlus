package emailincident

import (
	"context"
	"log/slog"

	dto "dpdp-backend/internal/audit/dto/emailincident"
	repo "dpdp-backend/internal/audit/repositories/emailincident"
)

type EmailIncidentService struct {
	repo *repo.EmailIncidentRepository
}

func NewEmailIncidentService(repo *repo.EmailIncidentRepository) *EmailIncidentService {
	return &EmailIncidentService{repo: repo}
}

func (s *EmailIncidentService) EnsureIndexes(ctx context.Context) error {
	return s.repo.EnsureIndexes(ctx)
}

func (s *EmailIncidentService) Record(ctx context.Context, record dto.Record) error {
	return s.repo.Upsert(ctx, record)
}

func (s *EmailIncidentService) UpdateAction(ctx context.Context, correlationID string, outcome dto.ActionOutcome) error {
	if err := s.repo.ApplyAction(ctx, correlationID, outcome); err != nil {
		slog.ErrorContext(ctx, "incident action outcome not recorded",
			"correlation_id", correlationID,
			"action", outcome.Action,
			"status", outcome.Status,
			"error", err,
		)

		return err
	}

	return nil
}
