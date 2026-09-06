package email

import (
	"errors"
	"fmt"

	"dpdp-backend/internal/admin/utils"
)

var (
	ErrNotFound        = errors.New(utils.MsgConfigurationNotFound)
	ErrDomainTaken     = errors.New(utils.MsgDomainTaken)
	ErrInvalidDomain   = errors.New(utils.MsgInvalidDomain)
	ErrInvalidProvider = errors.New(utils.MsgInvalidProvider)
	ErrUnauthenticated = errors.New(utils.MsgUnauthenticated)
	ErrUnavailable     = errors.New(utils.MsgUnavailable)
)

func domainTaken(domain string) error {
	return fmt.Errorf("%w (%s)", ErrDomainTaken, domain)
}
