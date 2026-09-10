package emailincident

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"

	"go.mongodb.org/mongo-driver/v2/mongo"

	adminutils "dpdp-backend/internal/admin/utils"
	dto "dpdp-backend/internal/audit/dto/emailincident"
	repo "dpdp-backend/internal/audit/repositories/emailincident"
	"dpdp-backend/internal/audit/utils"
)

const defaultSortField = "created_at"

var sortableFields = []string{"created_at", "effective_action", "trigger", "from", "sender_domain"}

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

func (s *EmailIncidentService) List(ctx context.Context, customerID int, params dto.ListParams) (dto.Listing, error) {
	params.Page, params.PageSize = adminutils.NormalizePaging(params.Page, params.PageSize)
	params.Search = strings.TrimSpace(params.Search)

	if !slices.Contains(sortableFields, params.SortBy) {
		params.SortBy = defaultSortField
		params.SortDesc = true
	}

	total, err := s.repo.Count(ctx, customerID, params)
	if err != nil {
		return dto.Listing{}, err
	}

	items, err := s.repo.Search(ctx, customerID, params)
	if err != nil {
		return dto.Listing{}, err
	}

	return dto.Listing{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}, nil
}

func (s *EmailIncidentService) Get(ctx context.Context, customerID int, correlationID string) (dto.Incident, error) {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return dto.Incident{}, utils.ErrEmailIncidentNotFound
	}

	incident, err := s.repo.FindByCorrelationID(ctx, customerID, correlationID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return dto.Incident{}, utils.ErrEmailIncidentNotFound
	}

	return incident, err
}
