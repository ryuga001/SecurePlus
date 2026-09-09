package utils

import (
	"errors"
	"fmt"
)

const (
	MsgDomainUnknown           = "sender domain is not configured"
	MsgSigningConfigMissing    = "signing configuration is not available"
	MsgNoDestination           = "no mail exchanger or address record for domain"
	MsgPrivateKeyMissing       = "dkim private key is not configured"
	MsgPrivateKeyInvalid       = "dkim private key could not be parsed"
	MsgAllRecipientsRestricted = "all recipients withheld by policy"
	MsgDeliveryPanicked        = "delivery panicked"
)

var (
	ErrDomainUnknown           = errors.New(MsgDomainUnknown)
	ErrSigningConfigMissing    = errors.New(MsgSigningConfigMissing)
	ErrNoDestination           = errors.New(MsgNoDestination)
	ErrPrivateKeyMissing       = errors.New(MsgPrivateKeyMissing)
	ErrPrivateKeyInvalid       = errors.New(MsgPrivateKeyInvalid)
	ErrAllRecipientsRestricted = errors.New(MsgAllRecipientsRestricted)
)

const (
	MsgTooManyRules      = "policy set exceeds the evaluable rule limit"
	MsgUnknownMatcher    = "no matcher is registered for this rule type"
	MsgUnknownAction     = "no executor is registered for this action"
	MsgMessageUnreadable = "message could not be parsed"
)

var (
	ErrTooManyRules      = errors.New(MsgTooManyRules)
	ErrUnknownMatcher    = errors.New(MsgUnknownMatcher)
	ErrUnknownAction     = errors.New(MsgUnknownAction)
	ErrMessageUnreadable = errors.New(MsgMessageUnreadable)
)

func TooManyRules(count, limit int) error {
	return fmt.Errorf("%w (%d rules, limit %d)", ErrTooManyRules, count, limit)
}

func UnknownMatcher(ruleType string) error {
	return fmt.Errorf("%w (%s)", ErrUnknownMatcher, ruleType)
}

func UnknownAction(action string) error {
	return fmt.Errorf("%w (%s)", ErrUnknownAction, action)
}
