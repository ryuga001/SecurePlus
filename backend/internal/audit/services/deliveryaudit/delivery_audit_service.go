package deliveryaudit

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	adminutils "dpdp-backend/internal/admin/utils"
	dto "dpdp-backend/internal/audit/dto/deliveryaudit"
	repo "dpdp-backend/internal/audit/repositories/deliveryaudit"
	"dpdp-backend/internal/audit/utils"
)

const (
	completeAttempts = 3
	completeBackoff  = time.Second

	defaultSortField = "created_at"
)

var sortableFields = []string{"created_at", "updated_at", "status", "from", "sender_domain", "size"}

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

func (s *DeliveryAuditService) List(ctx context.Context, customerID int, params dto.ListParams) (dto.Listing, error) {
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

func (s *DeliveryAuditService) Get(ctx context.Context, customerID int, correlationID string) (dto.Audit, error) {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return dto.Audit{}, utils.ErrDeliveryAuditNotFound
	}

	audit, err := s.repo.FindByCorrelationID(ctx, customerID, correlationID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return dto.Audit{}, utils.ErrDeliveryAuditNotFound
	}

	return audit, err
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
