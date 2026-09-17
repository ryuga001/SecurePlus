package smtp

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"

	gosmtp "github.com/emersion/go-smtp"

	"dpdp-backend/internal/delivery"
	deliveryutils "dpdp-backend/internal/delivery/utils"
)

var (
	errNotAuthorized  = &gosmtp.SMTPError{Code: 550, EnhancedCode: gosmtp.EnhancedCode{5, 7, 1}, Message: "sender domain not authorized"}
	errBadAddress     = &gosmtp.SMTPError{Code: 501, EnhancedCode: gosmtp.EnhancedCode{5, 1, 3}, Message: "malformed address"}
	errTooManyRcpt    = &gosmtp.SMTPError{Code: 452, EnhancedCode: gosmtp.EnhancedCode{4, 5, 3}, Message: "too many recipients"}
	errLookupDown     = &gosmtp.SMTPError{Code: 451, EnhancedCode: gosmtp.EnhancedCode{4, 3, 0}, Message: "sender domain lookup unavailable"}
	errAuditDown      = &gosmtp.SMTPError{Code: 451, EnhancedCode: gosmtp.EnhancedCode{4, 3, 0}, Message: "delivery audit unavailable"}
	errQueueDown      = &gosmtp.SMTPError{Code: 451, EnhancedCode: gosmtp.EnhancedCode{4, 3, 2}, Message: "requested action aborted: temporary failure"}
	errMessageTooBig  = &gosmtp.SMTPError{Code: 552, EnhancedCode: gosmtp.EnhancedCode{5, 3, 4}, Message: "message exceeds maximum size"}
	errNoValidSenders = &gosmtp.SMTPError{Code: 550, EnhancedCode: gosmtp.EnhancedCode{5, 7, 1}, Message: "sender is required"}
)

type Authorization struct {
	CustomerID int
	ConfigID   int
	Domain     string
}

type Backend struct {
	authorizer *Authorizer
	acceptor   *delivery.Acceptor
	maxSize    int64
	maxRcpt    int
}

func NewBackend(
	authorizer *Authorizer,
	acceptor *delivery.Acceptor,
	maxSize int64,
	maxRecipients int,
) *Backend {
	return &Backend{
		authorizer: authorizer,
		acceptor:   acceptor,
		maxSize:    maxSize,
		maxRcpt:    maxRecipients,
	}
}

func (b *Backend) NewSession(conn *gosmtp.Conn) (gosmtp.Session, error) {
	return &session{backend: b, ctx: context.Background()}, nil
}

type session struct {
	backend    *Backend
	ctx        context.Context
	from       string
	domain     string
	recipients []string
	auth       Authorization
}

func (s *session) Mail(from string, opts *gosmtp.MailOptions) error {
	address := strings.TrimSpace(from)
	if address == "" {
		return errNoValidSenders
	}

	domain := delivery.DomainOf(address)
	if domain == "" {
		return errBadAddress
	}

	authorization, err := s.backend.authorizer.Authorize(s.ctx, domain)
	if err != nil {
		if errors.Is(err, deliveryutils.ErrDomainUnknown) {
			return errNotAuthorized
		}

		slog.ErrorContext(s.ctx, "sender domain lookup failed", "domain", domain, "error", err)

		return errLookupDown
	}

	s.from = address
	s.domain = domain
	s.auth = authorization

	return nil
}

func (s *session) Rcpt(to string, opts *gosmtp.RcptOptions) error {
	address := strings.TrimSpace(to)
	if delivery.DomainOf(address) == "" {
		return errBadAddress
	}

	if len(s.recipients) >= s.backend.maxRcpt {
		return errTooManyRcpt
	}

	s.recipients = append(s.recipients, address)

	return nil
}

func (s *session) Data(r io.Reader) error {
	if s.from == "" || len(s.recipients) == 0 {
		return errBadAddress
	}

	raw, err := io.ReadAll(io.LimitReader(r, s.backend.maxSize+1))
	if err != nil {
		return err
	}

	if int64(len(raw)) > s.backend.maxSize {
		return errMessageTooBig
	}

	_, err = s.backend.acceptor.Accept(s.ctx, delivery.Submission{
		CustomerID:   s.auth.CustomerID,
		ConfigID:     s.auth.ConfigID,
		From:         s.from,
		SenderDomain: s.domain,
		Recipients:   s.recipients,
		Raw:          raw,
	})

	switch {
	case errors.Is(err, delivery.ErrAuditUnavailable):
		return errAuditDown
	case err != nil:
		return errQueueDown
	}

	s.reset()

	return nil
}

func (s *session) Reset() {
	s.reset()
}

func (s *session) Logout() error {
	s.reset()

	return nil
}

func (s *session) reset() {

	s.from = ""
	s.domain = ""
	s.recipients = nil
	s.auth = Authorization{}
}

func MessageID(raw []byte) string {
	return delivery.MessageIDOf(raw)
}
