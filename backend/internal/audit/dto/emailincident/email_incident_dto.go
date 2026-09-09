package emailincident

import (
	"context"
	"time"
)

type Recipient struct {
	Email  string
	Domain string
}

type WithheldRecipient struct {
	Email      string
	Domain     string
	PolicyID   int
	PolicyName string
}

type RestrictionViolation struct {
	Kind        string
	Mode        string
	Value       string
	Filename    string
	ContentType string
	PolicyID    int
	PolicyName  string
}

type Match struct {
	PolicyID        int
	PolicyName      string
	RuleID          int
	RuleName        string
	RuleType        string
	ConfiguredValue string
	Occurrences     int
	Locations       []string
}

type Record struct {
	CorrelationID         string
	MessageID             string
	CustomerID            int
	ConfigID              int
	From                  string
	SenderDomain          string
	Recipients            []Recipient
	EmailUserID           int
	EvaluatedPolicyCount  int
	TriggeredPolicyIDs    []int
	Decision              string
	Trigger               string
	EffectiveAction       string
	ActionInvoked         string
	ActionStatus          string
	WithheldRecipients    []WithheldRecipient
	RestrictionViolations []RestrictionViolation
	Matches               []Match
}

type ActionOutcome struct {
	Action string
	Status string
	Error  string
}

type Recorder interface {
	Record(ctx context.Context, record Record) error
	UpdateAction(ctx context.Context, correlationID string, outcome ActionOutcome) error
}

type Incident struct {
	Record
	ActionError string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
