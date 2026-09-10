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

type ListParams struct {
	Search   string
	Trigger  string
	Action   string
	Decision string
	From     time.Time
	To       time.Time
	SortBy   string
	SortDesc bool
	Page     int
	PageSize int
}

type Listing struct {
	Items    []Incident
	Page     int
	PageSize int
	Total    int64
}

type ListQuery struct {
	Search      string `form:"search" binding:"max=200"`
	Trigger     string `form:"trigger" binding:"omitempty,oneof=RESTRICTION CONTENT"`
	Action      string `form:"action" binding:"omitempty,oneof=BLOCK QUARANTINE REDACT AUDIT"`
	CreatedFrom string `form:"createdFrom" binding:"max=40"`
	CreatedTo   string `form:"createdTo" binding:"max=40"`
	SortBy      string `form:"sort_by" binding:"max=40"`
	SortDir     string `form:"sort_dir" binding:"omitempty,oneof=asc desc"`
	Page        int    `form:"page" binding:"omitempty,min=1"`
	PageSize    int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type RecipientResponse struct {
	Email  string `json:"email"`
	Domain string `json:"domain"`
}

type WithheldResponse struct {
	Email      string `json:"email"`
	Domain     string `json:"domain"`
	PolicyID   int    `json:"policy_id"`
	PolicyName string `json:"policy_name"`
}

type ViolationResponse struct {
	Kind        string `json:"kind"`
	Mode        string `json:"mode"`
	Value       string `json:"value"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	PolicyID    int    `json:"policy_id"`
	PolicyName  string `json:"policy_name"`
}

type MatchResponse struct {
	PolicyID        int      `json:"policy_id"`
	PolicyName      string   `json:"policy_name"`
	RuleID          int      `json:"rule_id"`
	RuleName        string   `json:"rule_name"`
	RuleType        string   `json:"rule_type"`
	ConfiguredValue string   `json:"configured_value"`
	Occurrences     int      `json:"occurrences"`
	Locations       []string `json:"locations"`
}

type ListItem struct {
	CorrelationID   string    `json:"correlation_id"`
	MessageID       string    `json:"message_id"`
	From            string    `json:"from"`
	SenderDomain    string    `json:"sender_domain"`
	Recipients      []string  `json:"recipients"`
	RecipientCount  int       `json:"recipient_count"`
	Decision        string    `json:"decision"`
	Trigger         string    `json:"trigger"`
	EffectiveAction string    `json:"effective_action"`
	ActionStatus    string    `json:"action_status"`
	WithheldCount   int       `json:"withheld_count"`
	ViolationCount  int       `json:"violation_count"`
	MatchCount      int       `json:"match_count"`
	CreatedAt       time.Time `json:"created_at"`
}

type Response struct {
	CorrelationID         string              `json:"correlation_id"`
	MessageID             string              `json:"message_id"`
	ConfigID              int                 `json:"config_id"`
	From                  string              `json:"from"`
	SenderDomain          string              `json:"sender_domain"`
	Recipients            []RecipientResponse `json:"recipients"`
	EmailUserID           int                 `json:"email_user_id"`
	EvaluatedPolicyCount  int                 `json:"evaluated_policy_count"`
	TriggeredPolicyIDs    []int               `json:"triggered_policy_ids"`
	Decision              string              `json:"decision"`
	Trigger               string              `json:"trigger"`
	EffectiveAction       string              `json:"effective_action"`
	ActionInvoked         string              `json:"action_invoked"`
	ActionStatus          string              `json:"action_status"`
	ActionError           string              `json:"action_error"`
	WithheldRecipients    []WithheldResponse  `json:"withheld_recipients"`
	RestrictionViolations []ViolationResponse `json:"restriction_violations"`
	Matches               []MatchResponse     `json:"matches"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
}
