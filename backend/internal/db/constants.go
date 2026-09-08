package db

const (
	SystemCustomerID = 1

	RoleTypeSuperAdmin = "super_admin"
	RoleTypeAdmin      = "admin"

	PrivilegeTypeDashboard = "DASHBOARD"
)

const (
	PolicyTypeEmail = "EMAIL"
	GroupTypeUser   = "USER"
	RuleTypeRegex   = "REGEX"
	RuleTypeKeyword = "KEYWORD"
)

const (
	ActionBlock      = "BLOCK"
	ActionAudit      = "AUDIT"
	ActionQuarantine = "QUARANTINE"
	ActionRedact     = "REDACT"
)

const (
	RestrictionNone  = "NONE"
	RestrictionBlock = "BLOCK"
	RestrictionAllow = "ALLOW"
)
