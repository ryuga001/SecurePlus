package utils

const DeliveryAuditCollection = "delivery_audits"

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
