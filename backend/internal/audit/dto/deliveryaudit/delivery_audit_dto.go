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
