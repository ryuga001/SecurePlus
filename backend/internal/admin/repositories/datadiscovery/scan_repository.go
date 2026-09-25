package datadiscovery

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

type ScanListParams struct {
	PolicyIDs []int
	Statuses  []string
	Page      int
	PageSize  int
}

type FileResultListParams struct {
	Statuses     []string
	WithFindings bool
	Page         int
	PageSize     int
}

var activeScanStatuses = []string{db.ScanStatusPending, db.ScanStatusRunning}

var fileResultColumns = []string{
	"target_position", "file_name", "extension", "mime_type", "size_bytes", "modified_at",
	"status", "error_code", "findings_total", "findings", "bytes_processed", "duration_ms",
	"processed_at",
}

type ScanRepository struct {
	db *gorm.DB
}

func NewScanRepository(database *gorm.DB) *ScanRepository {
	return &ScanRepository{db: database}
}

func (r *ScanRepository) WithTx(tx *gorm.DB) *ScanRepository {
	return &ScanRepository{db: tx}
}

func (r *ScanRepository) Insert(ctx context.Context, row *db.DataDiscoveryScan) error {
	return r.db.WithContext(ctx).
		Omit("CreatedAt", "UpdatedAt", "StartedAt", "FinishedAt").
		Create(row).Error
}

func (r *ScanRepository) Find(ctx context.Context, customerID int, id int64) (db.DataDiscoveryScan, error) {
	var row db.DataDiscoveryScan

	err := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryScan{}).
		Scopes(db.TenantScope(customerID)).
		Where("id = ?", id).
		Take(&row).Error

	return row, err
}

func (r *ScanRepository) List(
	ctx context.Context,
	customerID int,
	params ScanListParams,
) ([]db.DataDiscoveryScan, int64, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	query := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryScan{}).
		Scopes(db.TenantScope(customerID))

	if len(params.PolicyIDs) > 0 {
		query = query.Where("policy_id IN ?", params.PolicyIDs)
	}

	if len(params.Statuses) > 0 {
		query = query.Where("status IN ?", params.Statuses)
	}

	query = query.Session(&gorm.Session{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]db.DataDiscoveryScan, 0, pageSize)

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

func (r *ScanRepository) Targets(ctx context.Context, customerID int, scanID int64) ([]db.DataDiscoveryScanTarget, error) {
	rows := make([]db.DataDiscoveryScanTarget, 0)

	err := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryScanTarget{}).
		Scopes(db.TenantScope(customerID)).
		Where("scan_id = ?", scanID).
		Order("position ASC").
		Find(&rows).Error

	return rows, err
}

func (r *ScanRepository) ListFiles(
	ctx context.Context,
	customerID int,
	scanID int64,
	params FileResultListParams,
) ([]db.DataDiscoveryFileResult, int64, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	query := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryFileResult{}).
		Scopes(db.TenantScope(customerID)).
		Where("scan_id = ?", scanID)

	if len(params.Statuses) > 0 {
		query = query.Where("status IN ?", params.Statuses)
	}

	if params.WithFindings {
		query = query.Where("findings_total > 0")
	}

	query = query.Session(&gorm.Session{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]db.DataDiscoveryFileResult, 0, pageSize)

	if total > int64(utils.Offset(page, pageSize)) {
		err := query.
			Order("findings_total DESC, id ASC").
			Limit(pageSize).
			Offset(utils.Offset(page, pageSize)).
			Find(&rows).Error
		if err != nil {
			return nil, 0, err
		}
	}

	return rows, total, nil
}

func (r *ScanRepository) PolicyNamesFor(
	ctx context.Context,
	customerID int,
	policyIDs []int,
) ([]utils.ReferenceItem, error) {
	rows := make([]utils.ReferenceItem, 0, len(policyIDs))

	if len(policyIDs) == 0 {
		return rows, nil
	}

	err := r.db.WithContext(ctx).
		Table("data_discovery_policies AS p").
		Select("p.id AS id, p.name AS name").
		Scopes(db.TenantScopeOn("p", customerID)).
		Where("p.id IN ?", policyIDs).
		Scan(&rows).Error

	return rows, err
}

func (r *ScanRepository) Claim(ctx context.Context) (db.DataDiscoveryScan, bool, error) {
	var row db.DataDiscoveryScan

	result := r.db.WithContext(ctx).Raw(`
		UPDATE data_discovery_scans
		SET started_at = now(), updated_at = now()
		WHERE id = (
			SELECT id FROM data_discovery_scans
			WHERE status = ? AND started_at IS NULL
			ORDER BY id
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING *`, db.ScanStatusPending).Scan(&row)
	if result.Error != nil {
		return db.DataDiscoveryScan{}, false, result.Error
	}

	return row, result.RowsAffected > 0 && row.ID > 0, nil
}

func (r *ScanRepository) MarkRunning(ctx context.Context, scanID int64, totalTargets int) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryScan{}).
		Where("id = ? AND status = ?", scanID, db.ScanStatusPending).
		Updates(map[string]any{
			"status":        db.ScanStatusRunning,
			"total_targets": totalTargets,
			"updated_at":    time.Now(),
		})

	return result.RowsAffected, result.Error
}

func (r *ScanRepository) InsertTargets(ctx context.Context, rows []db.DataDiscoveryScanTarget) error {
	if len(rows) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).CreateInBatches(&rows, 200).Error
}

func (r *ScanRepository) Progress(ctx context.Context, scanID int64, counters db.ScanCounters) (int64, error) {
	updates := counters.Columns()
	updates["updated_at"] = time.Now()

	result := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryScan{}).
		Where("id = ? AND status = ?", scanID, db.ScanStatusRunning).
		Updates(updates)

	return result.RowsAffected, result.Error
}

func (r *ScanRepository) UpdateTarget(ctx context.Context, row db.DataDiscoveryScanTarget) error {
	return r.db.WithContext(ctx).
		Model(&db.DataDiscoveryScanTarget{}).
		Where("scan_id = ? AND position = ?", row.ScanID, row.Position).
		Updates(map[string]any{
			"status":           row.Status,
			"error_code":       row.ErrorCode,
			"files_discovered": row.FilesDiscovered,
			"files_skipped":    row.FilesSkipped,
			"files_succeeded":  row.FilesSucceeded,
			"files_failed":     row.FilesFailed,
			"findings_total":   row.FindingsTotal,
			"started_at":       row.StartedAt,
			"finished_at":      row.FinishedAt,
		}).Error
}

func (r *ScanRepository) UpsertFileResult(ctx context.Context, row *db.DataDiscoveryFileResult) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "scan_id"}, {Name: "file_key"}},
			DoUpdates: clause.AssignmentColumns(fileResultColumns),
		}).
		Omit("ID").
		Create(row).Error
}

func (r *ScanRepository) Finish(
	ctx context.Context,
	scanID int64,
	status string,
	errorCode *string,
	counters db.ScanCounters,
) (int64, error) {
	now := time.Now()

	updates := counters.Columns()
	updates["status"] = status
	updates["error_code"] = errorCode
	updates["finished_at"] = now
	updates["updated_at"] = now

	result := r.db.WithContext(ctx).
		Model(&db.DataDiscoveryScan{}).
		Where("id = ? AND status IN ?", scanID, activeScanStatuses).
		Updates(updates)

	return result.RowsAffected, result.Error
}

func (r *ScanRepository) CloseTargets(ctx context.Context, scanIDs []int64, errorCode string) error {
	if len(scanIDs) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		Model(&db.DataDiscoveryScanTarget{}).
		Where("scan_id IN ? AND status IN ?", scanIDs, []string{db.TargetStatusPending, db.TargetStatusRunning}).
		Updates(map[string]any{
			"status":      db.TargetStatusFailed,
			"error_code":  errorCode,
			"finished_at": time.Now(),
		}).Error
}

func (r *ScanRepository) FailStale(ctx context.Context, before time.Time) ([]int64, error) {
	ids := make([]int64, 0)

	err := r.db.WithContext(ctx).Raw(`
		UPDATE data_discovery_scans
		SET status = ?, error_code = ?, finished_at = now(), updated_at = now()
		WHERE status IN ? AND started_at IS NOT NULL AND updated_at < ?
		RETURNING id`,
		db.ScanStatusFailed, db.ScanErrorInterrupted, activeScanStatuses, before,
	).Scan(&ids).Error

	return ids, err
}
