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
	ConfigurationTypeEntra        = "MICROSOFT_ENTRA_ACCOUNT"
	ConfigurationTypeAzureStorage = "AZURE_STORAGE_ACCOUNT"
	ConfigurationTypeGoogleSA     = "GOOGLE_SERVICE_ACCOUNT"
	ConfigurationTypeAWSIAM       = "AWS_IAM"

	SourceTypeSharePoint  = "SHARE_POINT"
	SourceTypeOneDrive    = "ONE_DRIVE"
	SourceTypeAzureBlob   = "AZURE_BLOB"
	SourceTypeGoogleDrive = "GOOGLE_DRIVE"
	SourceTypeAWSS3       = "AWS_S3"

	DiscoveryStatusActive   = "ACTIVE"
	DiscoveryStatusInactive = "INACTIVE"

	AzureAuthModeServicePrincipal = "SERVICE_PRINCIPAL"

	GoogleAccessModeDelegation  = "DOMAIN_WIDE_DELEGATION"
	GoogleAccessModeSharedDrive = "SHARED_DRIVE"
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
