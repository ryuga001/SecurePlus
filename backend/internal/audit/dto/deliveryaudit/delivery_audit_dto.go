package deliveryaudit

import (
	"context"
	"time"
)

type Record struct {
	CorrelationID string
	MessageID     string
	CustomerID    int
	ConfigID      int
	From          string
	SenderDomain  string
	Recipients    []Recipient
	Size          int64
}

type Recipient struct {
	Email    string
	Domain   string
	Status   string
	SMTPCode int
	Error    string
}

type Attempt struct {
	Number     int
	StartedAt  time.Time
	FinishedAt time.Time
	MXHost     string
	TLS        string
	SMTPCode   int
	Error      string
}

type Failure struct {
	Type     string
	Reason   string
	SMTPCode int
}

type DKIM struct {
	Domain   string
	Selector string
	Signed   bool
}

type Result struct {
	Status     string
	Recipients []Recipient
	Failure    *Failure
	DKIM       DKIM
}

type Recorder interface {
	Create(ctx context.Context, record Record) error
	RecordAttempt(ctx context.Context, correlationID string, attempt Attempt) error
	Complete(ctx context.Context, correlationID string, result Result) error
}

type Audit struct {
	CorrelationID string
	MessageID     string
	CustomerID    int
	ConfigID      int
	From          string
	SenderDomain  string
	Recipients    []Recipient
	Status        string
	Failure       *Failure
	DKIM          DKIM
	Attempts      []Attempt
	Size          int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ListParams struct {
	Search   string
	Status   string
	Failure  string
	From     time.Time
	To       time.Time
	SortBy   string
	SortDesc bool
	Page     int
	PageSize int
}

type Listing struct {
	Items    []Audit
	Page     int
	PageSize int
	Total    int64
}

type ListQuery struct {
	Search      string `form:"search" binding:"max=200"`
	Status      string `form:"status" binding:"omitempty,oneof=PROCESSING SUCCESS FAILED"`
	Failure     string `form:"failure" binding:"omitempty,oneof=RULE PROCESSING DKIM RELAY UNKNOWN"`
	CreatedFrom string `form:"createdFrom" binding:"max=40"`
	CreatedTo   string `form:"createdTo" binding:"max=40"`
	SortBy      string `form:"sort_by" binding:"max=40"`
	SortDir     string `form:"sort_dir" binding:"omitempty,oneof=asc desc"`
	Page        int    `form:"page" binding:"omitempty,min=1"`
	PageSize    int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type RecipientResponse struct {
	Email    string `json:"email"`
	Domain   string `json:"domain"`
	Status   string `json:"status"`
	SMTPCode int    `json:"smtp_code"`
	Error    string `json:"error"`
}

type AttemptResponse struct {
	Number     int       `json:"number"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	MXHost     string    `json:"mx_host"`
	TLS        string    `json:"tls"`
	SMTPCode   int       `json:"smtp_code"`
	Error      string    `json:"error"`
}

type FailureResponse struct {
	Type     string `json:"type"`
	Reason   string `json:"reason"`
	SMTPCode int    `json:"smtp_code"`
}

type DKIMResponse struct {
	Domain   string `json:"domain"`
	Selector string `json:"selector"`
	Signed   bool   `json:"signed"`
}

type ListItem struct {
	CorrelationID  string    `json:"correlation_id"`
	MessageID      string    `json:"message_id"`
	From           string    `json:"from"`
	SenderDomain   string    `json:"sender_domain"`
	Recipients     []string  `json:"recipients"`
	RecipientCount int       `json:"recipient_count"`
	Status         string    `json:"status"`
	FailureType    string    `json:"failure_type"`
	AttemptCount   int       `json:"attempt_count"`
	Size           int64     `json:"size"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Response struct {
	CorrelationID string              `json:"correlation_id"`
	MessageID     string              `json:"message_id"`
	ConfigID      int                 `json:"config_id"`
	From          string              `json:"from"`
	SenderDomain  string              `json:"sender_domain"`
	Recipients    []RecipientResponse `json:"recipients"`
	Status        string              `json:"status"`
	Failure       *FailureResponse    `json:"failure"`
	DKIM          DKIMResponse        `json:"dkim"`
	Attempts      []AttemptResponse   `json:"attempts"`
	Size          int64               `json:"size"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}
