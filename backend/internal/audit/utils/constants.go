package utils

const DeliveryAuditCollection = "delivery_audits"

const (
	StatusProcessing = "PROCESSING"
	StatusSuccess    = "SUCCESS"
	StatusFailed     = "FAILED"
	StatusBlocked    = "BLOCKED"
)

const (
	FailureRule       = "RULE"
	FailureProcessing = "PROCESSING"
	FailureDKIM       = "DKIM"
	FailureRelay      = "RELAY"
	FailureUnknown    = "UNKNOWN"
)

const EmailIncidentCollection = "email_incidents"

const (
	DecisionPass    = "PASS"
	DecisionFlagged = "FLAGGED"
)

const (
	ActionPending = "PENDING"
	ActionInvoked = "INVOKED"
	ActionFailed  = "FAILED"
)
