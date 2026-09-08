package delivery

import (
	"strings"
	"time"
)

const DefaultDKIMSelector = "dpdp"

const (
	StatusProcessing = "PROCESSING"
	StatusSuccess    = "SUCCESS"
	StatusFailed     = "FAILED"
)

const (
	FailureRule       = "RULE"
	FailureProcessing = "PROCESSING"
	FailureDKIM       = "DKIM"
	FailureRelay      = "RELAY"
	FailureUnknown    = "UNKNOWN"
)

const (
	TLSNone       = "none"
	TLSVerified   = "verified"
	TLSUnverified = "unverified"
)

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

type Failure struct {
	Type     string
	Reason   string
	SMTPCode int
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
