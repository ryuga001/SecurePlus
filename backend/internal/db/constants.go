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
	ScanStatusPending   = "PENDING"
	ScanStatusRunning   = "RUNNING"
	ScanStatusPartial   = "PARTIAL"
	ScanStatusCompleted = "COMPLETED"
	ScanStatusFailed    = "FAILED"

	TargetStatusPending   = "PENDING"
	TargetStatusRunning   = "RUNNING"
	TargetStatusCompleted = "COMPLETED"
	TargetStatusPartial   = "PARTIAL"
	TargetStatusFailed    = "FAILED"

	FileStatusSucceeded = "SUCCEEDED"
	FileStatusFailed    = "FAILED"
)

const (
	ScanErrorPolicyNotFound        = "POLICY_NOT_FOUND"
	ScanErrorPolicyInactive        = "POLICY_INACTIVE"
	ScanErrorConfigurationInactive = "CONFIGURATION_INACTIVE"
	ScanErrorCredentialUnavailable = "CREDENTIAL_UNAVAILABLE"
	ScanErrorSourceUnsupported     = "SOURCE_UNSUPPORTED"
	ScanErrorSourceAuthFailed      = "SOURCE_AUTH_FAILED"
	ScanErrorSourceUnavailable     = "SOURCE_UNAVAILABLE"
	ScanErrorNoRules               = "NO_RULES"
	ScanErrorRulesTooLarge         = "RULES_TOO_LARGE"
	ScanErrorDatabaseUnavailable   = "DATABASE_UNAVAILABLE"
	ScanErrorInterrupted           = "INTERRUPTED"
	ScanErrorAllTargetsFailed      = "ALL_TARGETS_FAILED"
	ScanErrorInitializationFailed  = "INITIALIZATION_FAILED"

	TargetErrorListingFailed = "LISTING_FAILED"

	FileErrorFetchFailed      = "FETCH_FAILED"
	FileErrorParseFailed      = "PARSE_FAILED"
	FileErrorLimitExceeded    = "LIMIT_EXCEEDED"
	FileErrorTooLarge         = "FILE_TOO_LARGE"
	FileErrorTimeout          = "TIMEOUT"
	FileErrorEvaluationFailed = "EVALUATION_FAILED"
	FileErrorCancelled        = "CANCELLED"
	FileErrorPersistFailed    = "PERSIST_FAILED"
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
