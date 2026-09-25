package utils

const (
	DefaultPage     = 1
	DefaultPageSize = 25
	MaxPageSize     = 100
	MaxBatchIDs     = 200
)

const (
	MsgConfigurationNotFound = "email provider configuration not found"
	MsgDomainTaken           = "this domain is already configured"
	MsgInvalidDomain         = "enter a valid domain, for example example.com"
	MsgInvalidProvider       = "provider must be outlook365 or gmail"

	MsgPolicyNotFound             = "policy not found"
	MsgInvalidAction              = "action must be BLOCK, AUDIT, QUARANTINE or REDACT"
	MsgInvalidRestrictionMode     = "restriction must be NONE, BLOCK or ALLOW"
	MsgRestrictionValuesNeeded    = "add at least one entry to the restriction list"
	MsgInvalidRestrictionDomain   = "restriction list contains an invalid domain"
	MsgInvalidRestrictionFileType = "restriction list contains an unknown file type"
	MsgPolicyNameTaken            = "a policy with this name already exists"
	MsgPolicyNameNeeded           = "policy name is required"
	MsgRulesNeeded                = "select at least one rule"
	MsgGroupsNeeded               = "select at least one group"

	MsgRuleNotFound    = "rule not found"
	MsgRuleNameTaken   = "a rule with this name already exists"
	MsgRuleNameNeeded  = "rule name is required"
	MsgInvalidRuleType = "rule type must be REGEX or KEYWORD"
	MsgRuleValueNeeded = "rule value is required"
	MsgInvalidRegex    = "value must be a valid regular expression"

	MsgGroupNotFound    = "group not found"
	MsgGroupNameTaken   = "a group with this name already exists"
	MsgGroupNameNeeded  = "group name is required"
	MsgInvalidGroupType = "group type must be USER"

	MsgEmailUserNotFound = "email user not found"
	MsgEmailTaken        = "this email is already registered"
	MsgInvalidEmail      = "enter a valid email address"
	MsgNameNeeded        = "first name and last name are required"

	MsgUnknownGroup     = "one or more selected groups are not available"
	MsgUnknownRule      = "one or more selected rules are not available"
	MsgUnknownEmailUser = "the selected user is not available"
	MsgMappingExists    = "this assignment already exists"
	MsgMappingNotFound  = "this assignment does not exist"
	MsgTooManyItems     = "too many items selected"

	MsgAlertNotFound           = "alert not found"
	MsgAlertNameTaken          = "an alert with this name already exists"
	MsgAlertNameNeeded         = "alert name must be between 2 and 100 characters"
	MsgPoliciesNeeded          = "select at least one policy"
	MsgUnknownPolicy           = "one or more selected policies are not available"
	MsgTargetsNeeded           = "add at least one recipient"
	MsgInvalidTarget           = "recipient list contains an invalid email address"
	MsgInvalidScheduleType     = "schedule must be REAL_TIME or CUSTOM"
	MsgInvalidNotificationType = "notification type must be EMAIL or SMS"
	MsgSMSNotSupported         = "SMS alerts are not supported yet, choose EMAIL"
	MsgSystemAlertImmutable    = "system alerts cannot be modified or deleted"

	MsgDiscoveryNameNeeded        = "name must be between 2 and 100 characters"
	MsgDiscoveryConfigNotFound    = "data discovery configuration not found"
	MsgDiscoveryConfigNameTaken   = "a data discovery configuration with this name already exists"
	MsgDiscoveryConfigInUse       = "this configuration is used by one or more discovery policies"
	MsgConfigurationTypeImmutable = "the configuration type cannot be changed after creation"
	MsgInvalidConfigurationType   = "configuration type must be MICROSOFT_ENTRA_ACCOUNT, AZURE_STORAGE_ACCOUNT, GOOGLE_SERVICE_ACCOUNT or AWS_IAM"
	MsgInvalidSourceType          = "source type must be SHARE_POINT, ONE_DRIVE, AZURE_BLOB, GOOGLE_DRIVE or AWS_S3"
	MsgIncompatibleSource         = "this source type is not available for the selected configuration"
	MsgConfigFieldNeeded          = "one or more required connection fields are missing"
	MsgSecretNeeded               = "the connection secret is required"
	MsgInvalidServiceAccountKey   = "the uploaded service account key is not valid or is missing required fields"
	MsgConnectionFailed           = "the connection test failed"
	MsgProviderUnavailable        = "the provider could not be reached, try again shortly"
	MsgUnknownConfiguration       = "the selected configuration is not available"
	MsgDiscoveryPolicyNotFound    = "data discovery policy not found"
	MsgDiscoveryPolicyNameTaken   = "a data discovery policy with this name already exists"
	MsgDiscoveryTargetsNeeded     = "add at least one target"
	MsgInvalidDiscoveryTarget     = "target list contains an entry that is not valid for this source type"
	MsgUnknownFileType            = "one or more selected file types are not available"
	MsgCredentialUnavailable      = "stored credential could not be opened"
	MsgDiscoveryScanNotFound      = "data discovery scan not found"
	MsgDiscoveryScanActive        = "a scan is already pending or running for this policy"
	MsgDiscoveryPolicyInactive    = "the policy must be active to start a scan"
	MsgSourceNotScannable         = "scanning is not available for this source type yet"

	MsgBrandingNotFound = "branding settings are not available"
	MsgInvalidTheme     = "choose either LIGHT or DARK"
	MsgInvalidLanguage  = "choose either ENGLISH, JAPANESE or SPANISH"
	MsgInvalidTimezone  = "enter a valid IANA timezone, for example Asia/Kolkata"
	MsgInvalidLogo      = "logo must be a PNG, JPEG or WebP image of 1 MB or less"

	MsgInvalidRequest    = "request body is invalid"
	MsgInvalidIdentifier = "invalid identifier"
	MsgUnauthenticated   = "unauthenticated"
	MsgUnavailable       = "service temporarily unavailable"
	MsgUnexpected        = "unexpected error"
)

const (
	CodeNotFound        = "not_found"
	CodeValidation      = "validation_failed"
	CodeUnauthenticated = "unauthenticated"
	CodeUnavailable     = "service_unavailable"
	CodeInternal        = "internal_error"

	CodeDomainTaken     = "domain_taken"
	CodeInvalidDomain   = "invalid_domain"
	CodeInvalidProvider = "invalid_provider"

	CodePolicyNameTaken = "policy_name_taken"
	CodeRuleNameTaken   = "rule_name_taken"
	CodeGroupNameTaken  = "group_name_taken"
	CodeEmailTaken      = "email_taken"

	CodeInvalidRuleType  = "invalid_rule_type"
	CodeInvalidRegex     = "invalid_regex"
	CodeInvalidGroupType = "invalid_group_type"
	CodeInvalidEmail     = "invalid_email"

	CodeGroupNotFound     = "group_not_found"
	CodeRuleNotFound      = "rule_not_found"
	CodeEmailUserNotFound = "email_user_not_found"
	CodeMappingExists     = "mapping_exists"

	CodeAlertNameTaken          = "alert_name_taken"
	CodePolicyNotFound          = "policy_not_found"
	CodeInvalidTarget           = "invalid_target"
	CodeInvalidScheduleType     = "invalid_schedule_type"
	CodeInvalidNotificationType = "invalid_notification_type"
	CodeUnsupportedNotification = "unsupported_notification_type"
	CodeSystemAlertImmutable    = "system_alert_immutable"

	CodeDiscoveryConfigNotFound    = "discovery_configuration_not_found"
	CodeDiscoveryConfigNameTaken   = "discovery_configuration_name_taken"
	CodeConfigurationInUse         = "configuration_in_use"
	CodeConfigurationTypeImmutable = "configuration_type_immutable"
	CodeInvalidConfigurationType   = "invalid_configuration_type"
	CodeInvalidSourceType          = "invalid_source_type"
	CodeIncompatibleSource         = "incompatible_source_type"
	CodeInvalidServiceAccountKey   = "invalid_service_account_key"
	CodeConnectionFailed           = "connection_failed"
	CodeProviderUnavailable        = "provider_unavailable"
	CodeDiscoveryPolicyNotFound    = "discovery_policy_not_found"
	CodeDiscoveryPolicyNameTaken   = "discovery_policy_name_taken"
	CodeInvalidDiscoveryTarget     = "invalid_discovery_target"
	CodeFileTypeNotFound           = "file_type_not_found"
	CodeDiscoveryScanNotFound      = "discovery_scan_not_found"
	CodeDiscoveryScanActive        = "discovery_scan_active"
	CodeDiscoveryPolicyInactive    = "discovery_policy_inactive"
	CodeSourceNotScannable         = "source_not_scannable"

	CodeInvalidTheme    = "invalid_theme"
	CodeInvalidLanguage = "invalid_language"
	CodeInvalidTimezone = "invalid_timezone"
	CodeInvalidLogo     = "invalid_logo"
)

const (
	PolicyTypeEmail = "EMAIL"
	GroupTypeUser   = "USER"
	RuleTypeRegex   = "REGEX"
	RuleTypeKeyword = "KEYWORD"
)

const (
	ProviderOutlook365 = "outlook365"
	ProviderGmail      = "gmail"
)
