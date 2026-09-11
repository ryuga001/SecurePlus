package utils

import "time"

const DefaultDKIMSelector = "dpdp"

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

const (
	TLSNone       = "none"
	TLSVerified   = "verified"
	TLSUnverified = "unverified"
)

const (
	EvaluationEnabled = true
	FailClosed        = false

	CacheTTL          = time.Minute
	MaxRules          = 2000
	MaxMatchesPerRule = 1000
)

const (
	DecisionPass    = "PASS"
	DecisionFlagged = "FLAGGED"
)

const (
	TemplatePolicyBlockNotice = "policy_block_notice"

	NoticeMailbox    = "no-reply"
	NoticeLineLength = 76
	NoticeNoSubject  = "(no subject)"

	NoticeReasonRestriction = "a recipient or attachment restriction"
	NoticeReasonContent     = "a content rule"
)

const (
	ActionNone       = "NONE"
	ActionAudit      = "AUDIT"
	ActionRedact     = "REDACT"
	ActionQuarantine = "QUARANTINE"
	ActionBlock      = "BLOCK"
)

const (
	TriggerRestriction = "RESTRICTION"
	TriggerContent     = "CONTENT"
)

const (
	RestrictionNone  = "NONE"
	RestrictionBlock = "BLOCK"
	RestrictionAllow = "ALLOW"
)

const (
	KindDomain     = "DOMAIN"
	KindAttachment = "ATTACHMENT"
)

const (
	MatcherKeyword = "KEYWORD"
	MatcherRegex   = "REGEX"
)

const (
	LocationSubject = "SUBJECT"
	LocationBody    = "BODY"
)

const (
	ActionPending = "PENDING"
	ActionInvoked = "INVOKED"
	ActionFailed  = "FAILED"
)

var actionPriority = map[string]int{
	ActionNone:       0,
	ActionAudit:      1,
	ActionRedact:     2,
	ActionQuarantine: 3,
	ActionBlock:      4,
}

func ActionPriority(action string) int {
	return actionPriority[action]
}
