package delivery

import (
	"context"
	"strings"
	"time"
)

type Processor interface {
	Process(ctx context.Context, msg EmailMessage) (ProcessResult, error)
}

type Sender interface {
	Deliver(ctx context.Context, msg EmailMessage, cfg TenantConfig, sink func(Attempt)) ([]RecipientResult, error)
}

type EmailMessage struct {
	CorrelationID string
	MessageID     string
	CustomerID    int
	ConfigID      int
	From          string
	SenderDomain  string
	Recipients    []string
	Raw           []byte
	Size          int64
	ReceivedAt    time.Time
}

type TenantConfig struct {
	CustomerID     int
	ConfigID       int
	Domain         string
	DKIMSelector   string
	DKIMPrivateKey string
}

type ProcessResult struct {
	Message  EmailMessage
	Config   TenantConfig
	Withheld []RecipientResult
}

type RecipientResult struct {
	Email     string
	Domain    string
	Status    string
	SMTPCode  int
	Error     string
	Permanent bool
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

func DomainOf(address string) string {
	at := strings.LastIndex(address, "@")
	if at < 0 || at == len(address)-1 {
		return ""
	}

	return strings.ToLower(strings.TrimSpace(address[at+1:]))
}

func GroupByDomain(recipients []string) map[string][]string {
	grouped := make(map[string][]string)

	for _, recipient := range recipients {
		domain := DomainOf(recipient)
		grouped[domain] = append(grouped[domain], recipient)
	}

	return grouped
}
