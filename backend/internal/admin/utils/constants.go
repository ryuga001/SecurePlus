package utils

const (
	MsgConfigurationNotFound = "email provider configuration not found"
	MsgDomainTaken           = "this domain is already configured"
	MsgInvalidDomain         = "enter a valid domain, for example example.com"
	MsgInvalidProvider       = "provider must be outlook365 or gmail"
	MsgInvalidRequest        = "request body is invalid"
	MsgInvalidIdentifier     = "invalid identifier"
	MsgUnauthenticated       = "unauthenticated"
	MsgUnavailable           = "service temporarily unavailable"
	MsgUnexpected            = "unexpected error"
)

const (
	CodeNotFound        = "not_found"
	CodeDomainTaken     = "domain_taken"
	CodeInvalidDomain   = "invalid_domain"
	CodeInvalidProvider = "invalid_provider"
	CodeValidation      = "validation_failed"
	CodeUnauthenticated = "unauthenticated"
	CodeUnavailable     = "service_unavailable"
	CodeInternal        = "internal_error"
)
