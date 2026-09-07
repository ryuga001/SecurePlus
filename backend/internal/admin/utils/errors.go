package utils

import (
	"errors"
	"fmt"
)

var (
	ErrUnauthenticated = errors.New(MsgUnauthenticated)
	ErrUnavailable     = errors.New(MsgUnavailable)

	ErrConfigurationNotFound = errors.New(MsgConfigurationNotFound)
	ErrDomainTaken           = errors.New(MsgDomainTaken)
	ErrInvalidDomain         = errors.New(MsgInvalidDomain)
	ErrInvalidProvider       = errors.New(MsgInvalidProvider)

	ErrPolicyNotFound             = errors.New(MsgPolicyNotFound)
	ErrInvalidAction              = errors.New(MsgInvalidAction)
	ErrInvalidRestrictionMode     = errors.New(MsgInvalidRestrictionMode)
	ErrRestrictionValuesNeeded    = errors.New(MsgRestrictionValuesNeeded)
	ErrInvalidRestrictionDomain   = errors.New(MsgInvalidRestrictionDomain)
	ErrInvalidRestrictionFileType = errors.New(MsgInvalidRestrictionFileType)
	ErrPolicyNameTaken            = errors.New(MsgPolicyNameTaken)
	ErrPolicyNameNeeded           = errors.New(MsgPolicyNameNeeded)
	ErrRulesNeeded                = errors.New(MsgRulesNeeded)
	ErrGroupsNeeded               = errors.New(MsgGroupsNeeded)

	ErrRuleNotFound    = errors.New(MsgRuleNotFound)
	ErrRuleNameTaken   = errors.New(MsgRuleNameTaken)
	ErrRuleNameNeeded  = errors.New(MsgRuleNameNeeded)
	ErrInvalidRuleType = errors.New(MsgInvalidRuleType)
	ErrRuleValueNeeded = errors.New(MsgRuleValueNeeded)
	ErrInvalidRegex    = errors.New(MsgInvalidRegex)

	ErrGroupNotFound    = errors.New(MsgGroupNotFound)
	ErrGroupNameTaken   = errors.New(MsgGroupNameTaken)
	ErrGroupNameNeeded  = errors.New(MsgGroupNameNeeded)
	ErrInvalidGroupType = errors.New(MsgInvalidGroupType)

	ErrEmailUserNotFound = errors.New(MsgEmailUserNotFound)
	ErrEmailTaken        = errors.New(MsgEmailTaken)
	ErrInvalidEmail      = errors.New(MsgInvalidEmail)
	ErrNameNeeded        = errors.New(MsgNameNeeded)

	ErrUnknownGroup     = errors.New(MsgUnknownGroup)
	ErrUnknownRule      = errors.New(MsgUnknownRule)
	ErrUnknownEmailUser = errors.New(MsgUnknownEmailUser)
	ErrMappingExists    = errors.New(MsgMappingExists)
	ErrMappingNotFound  = errors.New(MsgMappingNotFound)
	ErrTooManyItems     = errors.New(MsgTooManyItems)
)

func Taken(sentinel error, value string) error {
	return fmt.Errorf("%w (%s)", sentinel, value)
}

func InvalidRegex(cause error) error {
	return fmt.Errorf("%w (%s)", ErrInvalidRegex, cause)
}
