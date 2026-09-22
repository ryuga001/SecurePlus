package alert

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	repo "dpdp-backend/internal/admin/repositories/alert"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

const alertPolicyPreviewLimit = 3

type AlertDetail struct {
	Alert    db.Alert
	Policies []repo.AlertPolicyReference
}

type AlertSummary struct {
	Alert       db.Alert
	Policies    []repo.AlertPolicyReference
	PolicyCount int
	TargetCount int
}

type AlertListing struct {
	Items    []AlertSummary
	Page     int
	PageSize int
	Total    int64
}

type AlertInput struct {
	Name             string
	ScheduleType     string
	NotificationType string
	Target           []string
	PolicyIDs        []int
}

type AlertService struct {
	db   *gorm.DB
	repo *repo.AlertRepository
}

func NewAlertService(database *gorm.DB, repository *repo.AlertRepository) *AlertService {
	return &AlertService{db: database, repo: repository}
}

func (s *AlertService) List(ctx context.Context, customerID int, params repo.ListParams) (AlertListing, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	rows, total, err := s.repo.List(ctx, customerID, params)
	if err != nil {
		return AlertListing{}, err
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	policies, err := s.repo.PoliciesFor(ctx, customerID, ids)
	if err != nil {
		return AlertListing{}, err
	}

	counts := countByAlert(policies)
	previews := previewsByAlert(policies, alertPolicyPreviewLimit)

	items := make([]AlertSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, AlertSummary{
			Alert:       row,
			Policies:    previews[row.ID],
			PolicyCount: counts[row.ID],
			TargetCount: len(row.Target),
		})
	}

	return AlertListing{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *AlertService) Get(ctx context.Context, customerID int, id string) (AlertDetail, error) {
	if err := validID(id); err != nil {
		return AlertDetail{}, err
	}

	row, err := s.repo.Find(ctx, customerID, id)
	if err != nil {
		return AlertDetail{}, alertError(err, "")
	}

	return s.detail(ctx, s.repo, customerID, row)
}

func (s *AlertService) Create(ctx context.Context, customerID int, in AlertInput) (AlertDetail, error) {
	input, err := NormalizeAlertInput(in)
	if err != nil {
		return AlertDetail{}, err
	}

	row := db.Alert{
		ID:               uuid.NewString(),
		CustomerID:       customerID,
		Name:             input.Name,
		ScheduleType:     input.ScheduleType,
		NotificationType: input.NotificationType,
		Target:           db.StringList(input.Target),
		AlertType:        db.AlertTypeApplication,
	}

	var detail AlertDetail

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		if err := ensurePolicies(ctx, repository, customerID, input.PolicyIDs); err != nil {
			return err
		}
		if err := repository.Insert(ctx, &row); err != nil {
			return err
		}
		if err := repository.ReplacePolicies(ctx, customerID, row.ID, input.PolicyIDs); err != nil {
			return err
		}

		created, err := repository.Find(ctx, customerID, row.ID)
		if err != nil {
			return err
		}

		detail, err = s.detail(ctx, repository, customerID, created)

		return err
	})

	if err != nil {
		return AlertDetail{}, alertError(err, input.Name)
	}

	return detail, nil
}

func (s *AlertService) Update(ctx context.Context, customerID int, id string, in AlertInput) (AlertDetail, error) {
	if err := validID(id); err != nil {
		return AlertDetail{}, err
	}

	input, err := NormalizeAlertInput(in)
	if err != nil {
		return AlertDetail{}, err
	}

	var detail AlertDetail

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		current, err := repository.Lock(ctx, customerID, id)
		if err != nil {
			return err
		}

		if current.AlertType != db.AlertTypeApplication {
			return utils.ErrSystemAlertImmutable
		}

		if err := ensurePolicies(ctx, repository, customerID, input.PolicyIDs); err != nil {
			return err
		}

		updates := map[string]any{
			"name":              input.Name,
			"schedule_type":     input.ScheduleType,
			"notification_type": input.NotificationType,
			"target":            db.StringList(input.Target),
			"updated_at":        time.Now(),
		}

		if _, err := repository.Update(ctx, customerID, id, updates); err != nil {
			return err
		}

		if err := repository.ReplacePolicies(ctx, customerID, id, input.PolicyIDs); err != nil {
			return err
		}

		current.Name = input.Name
		current.ScheduleType = input.ScheduleType
		current.NotificationType = input.NotificationType
		current.Target = db.StringList(input.Target)

		detail, err = s.detail(ctx, repository, customerID, current)

		return err
	})

	if err != nil {
		return AlertDetail{}, alertError(err, input.Name)
	}

	return detail, nil
}

func (s *AlertService) Delete(ctx context.Context, customerID int, id string) error {
	if err := validID(id); err != nil {
		return err
	}

	row, err := s.repo.Find(ctx, customerID, id)
	if err != nil {
		return alertError(err, "")
	}

	if row.AlertType != db.AlertTypeApplication {
		return utils.ErrSystemAlertImmutable
	}

	affected, err := s.repo.Delete(ctx, customerID, id)
	if err != nil {
		return alertError(err, "")
	}
	if affected == 0 {
		return utils.ErrAlertNotFound
	}

	return nil
}

func (s *AlertService) RealTimeEmailAlerts(ctx context.Context, customerID int, policyIDs []int) ([]repo.AlertTarget, error) {
	return s.repo.MatchingPolicies(ctx, customerID, utils.NormalizeIDs(policyIDs))
}

func (s *AlertService) detail(
	ctx context.Context,
	repository *repo.AlertRepository,
	customerID int,
	row db.Alert,
) (AlertDetail, error) {
	policies, err := repository.PoliciesFor(ctx, customerID, []string{row.ID})
	if err != nil {
		return AlertDetail{}, err
	}

	return AlertDetail{Alert: row, Policies: policies}, nil
}

func ensurePolicies(ctx context.Context, repository *repo.AlertRepository, customerID int, ids []int) error {
	found, err := repository.LockOwnedPolicyIDs(ctx, customerID, ids)
	if err != nil {
		return err
	}
	if len(found) != len(ids) {
		return utils.ErrUnknownPolicy
	}

	return nil
}

func countByAlert(rows []repo.AlertPolicyReference) map[string]int {
	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.AlertID]++
	}

	return counts
}

func previewsByAlert(rows []repo.AlertPolicyReference, limit int) map[string][]repo.AlertPolicyReference {
	previews := make(map[string][]repo.AlertPolicyReference)

	for _, row := range rows {
		items := previews[row.AlertID]
		if len(items) >= limit {
			continue
		}
		previews[row.AlertID] = append(items, row)
	}

	return previews
}

func validID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return utils.ErrAlertNotFound
	}

	return nil
}

func alertError(err error, name string) error {
	switch {
	case db.IsNotFound(err):
		return utils.ErrAlertNotFound
	case db.IsDuplicate(err):
		return utils.Taken(utils.ErrAlertNameTaken, name)
	case db.IsMissingReference(err):
		return utils.ErrUnknownPolicy
	}

	return err
}

func NormalizeAlertInput(in AlertInput) (AlertInput, error) {
	name := utils.NormalizeName(in.Name)
	if length := utf8.RuneCountInString(name); length < 2 || length > 100 {
		return AlertInput{}, utils.ErrAlertNameNeeded
	}

	schedule := strings.ToUpper(strings.TrimSpace(in.ScheduleType))
	if schedule == "" {
		schedule = db.ScheduleTypeRealTime
	}
	if schedule != db.ScheduleTypeRealTime && schedule != db.ScheduleTypeCustom {
		return AlertInput{}, utils.ErrInvalidScheduleType
	}

	notification := strings.ToUpper(strings.TrimSpace(in.NotificationType))
	if notification == "" {
		notification = db.NotificationTypeEmail
	}
	if notification == db.NotificationTypeSMS {
		return AlertInput{}, utils.ErrSMSNotSupported
	}
	if notification != db.NotificationTypeEmail {
		return AlertInput{}, utils.ErrInvalidNotificationType
	}

	policyIDs := utils.NormalizeIDs(in.PolicyIDs)
	if len(policyIDs) == 0 {
		return AlertInput{}, utils.ErrPoliciesNeeded
	}
	if len(policyIDs) > utils.MaxBatchIDs {
		return AlertInput{}, utils.ErrTooManyItems
	}

	targets, err := normalizeTargets(in.Target)
	if err != nil {
		return AlertInput{}, err
	}

	return AlertInput{
		Name:             name,
		ScheduleType:     schedule,
		NotificationType: notification,
		Target:           targets,
		PolicyIDs:        policyIDs,
	}, nil
}

func normalizeTargets(raw []string) ([]string, error) {
	seen := make(map[string]bool, len(raw))
	values := make([]string, 0, len(raw))

	for _, entry := range raw {
		value := utils.NormalizeEmail(entry)
		if value == "" {
			return nil, utils.ErrInvalidTarget
		}
		if seen[value] {
			continue
		}

		seen[value] = true
		values = append(values, value)
	}

	if len(values) == 0 {
		return nil, utils.ErrTargetsNeeded
	}
	if len(values) > utils.MaxBatchIDs {
		return nil, utils.ErrTooManyItems
	}

	slices.Sort(values)

	return values, nil
}

func ParsePolicyIDs(raw string) []int {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	ids := make([]int, 0, len(parts))

	for _, part := range parts {
		parsed, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || parsed < 1 {
			continue
		}

		ids = append(ids, parsed)
	}

	return utils.NormalizeIDs(ids)
}
