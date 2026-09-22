package alert

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

type ListParams struct {
	Search           string
	NotificationType string
	ScheduleType     string
	PolicyIDs        []int
	Page             int
	PageSize         int
}

type AlertPolicyReference struct {
	AlertID string `gorm:"column:alert_id"`
	ID      int    `gorm:"column:id"`
	Name    string `gorm:"column:name"`
}

type AlertTarget struct {
	ID     string        `gorm:"column:id"`
	Name   string        `gorm:"column:name"`
	Target db.StringList `gorm:"column:target"`
}

type AlertRepository struct {
	db *gorm.DB
}

func NewAlertRepository(database *gorm.DB) *AlertRepository {
	return &AlertRepository{db: database}
}

func (r *AlertRepository) WithTx(tx *gorm.DB) *AlertRepository {
	return &AlertRepository{db: tx}
}

func (r *AlertRepository) List(ctx context.Context, customerID int, params ListParams) ([]db.Alert, int64, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	query := r.db.WithContext(ctx).
		Model(&db.Alert{}).
		Scopes(db.TenantScope(customerID))

	if search := utils.NormalizeName(params.Search); search != "" {
		query = query.Where(`(name ILIKE ? ESCAPE '\')`, utils.LikePattern(search))
	}

	if params.NotificationType != "" {
		query = query.Where("notification_type = ?", params.NotificationType)
	}

	if params.ScheduleType != "" {
		query = query.Where("schedule_type = ?", params.ScheduleType)
	}

	if ids := utils.NormalizeIDs(params.PolicyIDs); len(ids) > 0 {
		query = query.Where(`EXISTS (
			SELECT 1 FROM alert_policy_mapping m
			WHERE m.alert_id = alerts.id
			  AND m.customer_id = alerts.customer_id
			  AND m.policy_id IN ?)`, ids)
	}

	query = query.Session(&gorm.Session{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]db.Alert, 0, pageSize)

	if total > int64(utils.Offset(page, pageSize)) {
		err := query.
			Order("created_at DESC, id DESC").
			Limit(pageSize).
			Offset(utils.Offset(page, pageSize)).
			Find(&rows).Error
		if err != nil {
			return nil, 0, err
		}
	}

	return rows, total, nil
}

func (r *AlertRepository) Find(ctx context.Context, customerID int, id string) (db.Alert, error) {
	var row db.Alert

	err := r.db.WithContext(ctx).
		Model(&db.Alert{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *AlertRepository) Lock(ctx context.Context, customerID int, id string) (db.Alert, error) {
	var row db.Alert

	err := r.db.WithContext(ctx).
		Model(&db.Alert{}).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *AlertRepository) Insert(ctx context.Context, row *db.Alert) error {
	return r.db.WithContext(ctx).Omit("CreatedAt", "UpdatedAt").Create(row).Error
}

func (r *AlertRepository) Update(ctx context.Context, customerID int, id string, updates map[string]any) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&db.Alert{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Updates(updates)

	return result.RowsAffected, result.Error
}

func (r *AlertRepository) Delete(ctx context.Context, customerID int, id string) (int64, error) {
	result := r.db.WithContext(ctx).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Delete(&db.Alert{})

	return result.RowsAffected, result.Error
}

func (r *AlertRepository) LockOwnedPolicyIDs(ctx context.Context, customerID int, ids []int) ([]int, error) {
	found := make([]int, 0, len(ids))

	err := r.db.WithContext(ctx).
		Model(&db.Policy{}).
		Clauses(clause.Locking{Strength: "KEY SHARE"}).
		Scopes(db.TenantScope(customerID)).
		Where("id IN ? AND type = ?", ids, db.PolicyTypeEmail).
		Pluck("id", &found).Error

	return found, err
}

func (r *AlertRepository) ReplacePolicies(ctx context.Context, customerID int, alertID string, ids []int) error {
	err := r.db.WithContext(ctx).
		Where("alert_id = ? AND customer_id = ?", alertID, customerID).
		Delete(&db.AlertPolicyMapping{}).Error
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	rows := make([]db.AlertPolicyMapping, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, db.AlertPolicyMapping{AlertID: alertID, PolicyID: id, CustomerID: customerID})
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&rows).Error
}

func (r *AlertRepository) PoliciesFor(ctx context.Context, customerID int, alertIDs []string) ([]AlertPolicyReference, error) {
	rows := make([]AlertPolicyReference, 0, len(alertIDs))

	if len(alertIDs) == 0 {
		return rows, nil
	}

	err := r.db.WithContext(ctx).
		Table("alert_policy_mapping AS m").
		Select("m.alert_id AS alert_id, p.id AS id, p.policy_name AS name").
		Joins("JOIN policies AS p ON p.id = m.policy_id AND p.customer_id = m.customer_id").
		Scopes(db.TenantScopeOn("m", customerID)).
		Where("m.alert_id IN ?", alertIDs).
		Order("p.policy_name ASC, p.id ASC").
		Scan(&rows).Error

	return rows, err
}

func (r *AlertRepository) MatchingPolicies(ctx context.Context, customerID int, policyIDs []int) ([]AlertTarget, error) {
	rows := make([]AlertTarget, 0)

	if len(policyIDs) == 0 {
		return rows, nil
	}

	err := r.db.WithContext(ctx).
		Table("alerts AS a").
		Select("DISTINCT a.id AS id, a.name AS name, a.target AS target").
		Joins("JOIN alert_policy_mapping AS m ON m.alert_id = a.id AND m.customer_id = a.customer_id").
		Scopes(db.TenantScopeOn("a", customerID)).
		Where("a.alert_type = ?", db.AlertTypeApplication).
		Where("a.schedule_type = ?", db.ScheduleTypeRealTime).
		Where("a.notification_type = ?", db.NotificationTypeEmail).
		Where("m.policy_id IN ?", policyIDs).
		Order("a.name ASC, a.id ASC").
		Scan(&rows).Error

	return rows, err
}
