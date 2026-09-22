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

const (
	ScheduleTypeRealTime = "REAL_TIME"
	ScheduleTypeCustom   = "CUSTOM"

	NotificationTypeEmail = "EMAIL"
	NotificationTypeSMS   = "SMS"

	AlertTypeSystem      = "SYSTEM"
	AlertTypeApplication = "APPLICATION"
)

const (
	ThemeLight = "LIGHT"
	ThemeDark  = "DARK"

	LanguageEnglish  = "ENGLISH"
	LanguageJapanese = "JAPANESE"
	LanguageSpanish  = "SPANISH"

	TimezoneUTC = "UTC"
)
