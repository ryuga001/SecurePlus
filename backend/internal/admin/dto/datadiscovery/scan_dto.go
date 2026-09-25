package datadiscovery

import (
	"time"

	"dpdp-backend/internal/db"
)

type ScanRequest struct {
	PolicyID int `json:"policy_id" binding:"required,min=1"`
}

type ScanListQuery struct {
	PolicyIDs string `form:"policy_id" binding:"max=1200"`
	Statuses  string `form:"status" binding:"max=100"`
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type FileResultQuery struct {
	Statuses     string `form:"status" binding:"max=100"`
	WithFindings bool   `form:"with_findings"`
	Page         int    `form:"page" binding:"omitempty,min=1"`
	PageSize     int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type ScanTargetItem struct {
	Position        int        `json:"position"`
	Target          string     `json:"target"`
	Status          string     `json:"status"`
	ErrorCode       string     `json:"error_code"`
	FilesDiscovered int64      `json:"files_discovered"`
	FilesSkipped    int64      `json:"files_skipped"`
	FilesSucceeded  int64      `json:"files_succeeded"`
	FilesFailed     int64      `json:"files_failed"`
	FindingsTotal   int64      `json:"findings_total"`
	StartedAt       *time.Time `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at"`
}

type ScanListItem struct {
	ID         int64           `json:"id"`
	PolicyID   int             `json:"policy_id"`
	PolicyName string          `json:"policy_name"`
	Status     string          `json:"status"`
	ErrorCode  string          `json:"error_code"`
	Counters   db.ScanCounters `json:"counters"`
	StartedAt  *time.Time      `json:"started_at"`
	FinishedAt *time.Time      `json:"finished_at"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type ScanResponse struct {
	ScanListItem
	Targets []ScanTargetItem `json:"targets"`
}

type FileResultItem struct {
	ID             int64            `json:"id"`
	TargetPosition int              `json:"target_position"`
	FileKey        string           `json:"file_key"`
	FileName       string           `json:"file_name"`
	Extension      string           `json:"extension"`
	MIMEType       string           `json:"mime_type"`
	SizeBytes      int64            `json:"size_bytes"`
	ModifiedAt     *time.Time       `json:"modified_at"`
	Status         string           `json:"status"`
	ErrorCode      string           `json:"error_code"`
	FindingsTotal  int64            `json:"findings_total"`
	Findings       []db.FileFinding `json:"findings"`
	BytesProcessed int64            `json:"bytes_processed"`
	DurationMS     int64            `json:"duration_ms"`
	ProcessedAt    time.Time        `json:"processed_at"`
}
