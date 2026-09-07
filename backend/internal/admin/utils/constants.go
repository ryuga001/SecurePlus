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

	MsgPolicyNotFound   = "policy not found"
	MsgPolicyNameTaken  = "a policy with this name already exists"
	MsgPolicyNameNeeded = "policy name is required"
	MsgRulesNeeded      = "select at least one rule"
	MsgGroupsNeeded     = "select at least one group"

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
