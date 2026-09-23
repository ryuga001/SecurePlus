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

	ErrAlertNotFound           = errors.New(MsgAlertNotFound)
	ErrAlertNameTaken          = errors.New(MsgAlertNameTaken)
	ErrAlertNameNeeded         = errors.New(MsgAlertNameNeeded)
	ErrPoliciesNeeded          = errors.New(MsgPoliciesNeeded)
	ErrUnknownPolicy           = errors.New(MsgUnknownPolicy)
	ErrTargetsNeeded           = errors.New(MsgTargetsNeeded)
	ErrInvalidTarget           = errors.New(MsgInvalidTarget)
	ErrInvalidScheduleType     = errors.New(MsgInvalidScheduleType)
	ErrInvalidNotificationType = errors.New(MsgInvalidNotificationType)
	ErrSMSNotSupported         = errors.New(MsgSMSNotSupported)
	ErrSystemAlertImmutable    = errors.New(MsgSystemAlertImmutable)

	ErrDiscoveryNameNeeded        = errors.New(MsgDiscoveryNameNeeded)
	ErrDiscoveryConfigNotFound    = errors.New(MsgDiscoveryConfigNotFound)
	ErrDiscoveryConfigNameTaken   = errors.New(MsgDiscoveryConfigNameTaken)
	ErrDiscoveryConfigInUse       = errors.New(MsgDiscoveryConfigInUse)
	ErrConfigurationTypeImmutable = errors.New(MsgConfigurationTypeImmutable)
	ErrInvalidConfigurationType   = errors.New(MsgInvalidConfigurationType)
	ErrInvalidSourceType          = errors.New(MsgInvalidSourceType)
	ErrIncompatibleSource         = errors.New(MsgIncompatibleSource)
	ErrConfigFieldNeeded          = errors.New(MsgConfigFieldNeeded)
	ErrSecretNeeded               = errors.New(MsgSecretNeeded)
	ErrInvalidServiceAccountKey   = errors.New(MsgInvalidServiceAccountKey)
	ErrConnectionFailed           = errors.New(MsgConnectionFailed)
	ErrProviderUnavailable        = errors.New(MsgProviderUnavailable)
	ErrUnknownConfiguration       = errors.New(MsgUnknownConfiguration)
	ErrDiscoveryPolicyNotFound    = errors.New(MsgDiscoveryPolicyNotFound)
	ErrDiscoveryPolicyNameTaken   = errors.New(MsgDiscoveryPolicyNameTaken)
	ErrDiscoveryTargetsNeeded     = errors.New(MsgDiscoveryTargetsNeeded)
	ErrInvalidDiscoveryTarget     = errors.New(MsgInvalidDiscoveryTarget)
	ErrUnknownFileType            = errors.New(MsgUnknownFileType)
	ErrCredentialUnavailable      = errors.New(MsgCredentialUnavailable)

	ErrBrandingNotFound = errors.New(MsgBrandingNotFound)
	ErrInvalidTheme     = errors.New(MsgInvalidTheme)
	ErrInvalidLanguage  = errors.New(MsgInvalidLanguage)
	ErrInvalidTimezone  = errors.New(MsgInvalidTimezone)
	ErrInvalidLogo      = errors.New(MsgInvalidLogo)
)

func Taken(sentinel error, value string) error {
	return fmt.Errorf("%w (%s)", sentinel, value)
}

func ConnectionFailed(reason string) error {
	return fmt.Errorf("%w (%s)", ErrConnectionFailed, reason)
}

func InvalidRegex(cause error) error {
	return fmt.Errorf("%w (%s)", ErrInvalidRegex, cause)
}
