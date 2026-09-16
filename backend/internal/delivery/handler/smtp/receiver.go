package smtp

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	gosmtp "github.com/emersion/go-smtp"
	"github.com/google/uuid"

	auditdto "dpdp-backend/internal/audit/dto/deliveryaudit"
	auditutils "dpdp-backend/internal/audit/utils"
	"dpdp-backend/internal/delivery/dto/delivery"
	"dpdp-backend/internal/delivery/services/dispatch"
	deliveryutils "dpdp-backend/internal/delivery/utils"
)

var (
	errNotAuthorized  = &gosmtp.SMTPError{Code: 550, EnhancedCode: gosmtp.EnhancedCode{5, 7, 1}, Message: "sender domain not authorized"}
	errBadAddress     = &gosmtp.SMTPError{Code: 501, EnhancedCode: gosmtp.EnhancedCode{5, 1, 3}, Message: "malformed address"}
	errTooManyRcpt    = &gosmtp.SMTPError{Code: 452, EnhancedCode: gosmtp.EnhancedCode{4, 5, 3}, Message: "too many recipients"}
	errCapacityFull   = &gosmtp.SMTPError{Code: 451, EnhancedCode: gosmtp.EnhancedCode{4, 3, 2}, Message: "delivery capacity exhausted"}
	errLookupDown     = &gosmtp.SMTPError{Code: 451, EnhancedCode: gosmtp.EnhancedCode{4, 3, 0}, Message: "sender domain lookup unavailable"}
	errAuditDown      = &gosmtp.SMTPError{Code: 451, EnhancedCode: gosmtp.EnhancedCode{4, 3, 0}, Message: "delivery audit unavailable"}
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
	recorder   auditdto.Recorder
	dispatcher *dispatch.Dispatcher
	slots      chan struct{}
	maxSize    int64
	maxRcpt    int
}

func NewBackend(
	authorizer *Authorizer,
	recorder auditdto.Recorder,
	dispatcher *dispatch.Dispatcher,
	maxDeliveries int,
	maxSize int64,
	maxRecipients int,
) *Backend {
	return &Backend{
		authorizer: authorizer,
		recorder:   recorder,
		dispatcher: dispatcher,
		slots:      make(chan struct{}, maxDeliveries),
		maxSize:    maxSize,
		maxRcpt:    maxRecipients,
	}
}

func (b *Backend) NewSession(conn *gosmtp.Conn) (gosmtp.Session, error) {
	return &session{backend: b, ctx: context.Background()}, nil
}

func (b *Backend) acquire() bool {
	select {
	case b.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (b *Backend) release() {
	select {
	case <-b.slots:
	default:
	}
}

type session struct {
	backend    *Backend
	ctx        context.Context
	from       string
	domain     string
	recipients []string
	auth       Authorization
	holding    bool
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

	if len(s.recipients) == 1 {
		if !s.backend.acquire() {
			s.recipients = nil
			return errCapacityFull
		}

		s.holding = true
	}

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

	msg := delivery.EmailMessage{
		CorrelationID: uuid.NewString(),
		MessageID:     MessageID(raw),
		CustomerID:    s.auth.CustomerID,
		ConfigID:      s.auth.ConfigID,
		From:          s.from,
		SenderDomain:  s.domain,
		Recipients:    append([]string(nil), s.recipients...),
		Raw:           raw,
		Size:          int64(len(raw)),
		ReceivedAt:    time.Now().UTC(),
	}

	record := auditdto.Record{
		CorrelationID: msg.CorrelationID,
		MessageID:     msg.MessageID,
		CustomerID:    msg.CustomerID,
		ConfigID:      msg.ConfigID,
		From:          msg.From,
		SenderDomain:  msg.SenderDomain,
		Recipients:    pendingRecipients(msg.Recipients),
		Size:          msg.Size,
	}

	if err := s.backend.recorder.Create(s.ctx, record); err != nil {
		slog.ErrorContext(s.ctx, "delivery audit create failed",
			"correlation_id", msg.CorrelationID,
			"customer_id", msg.CustomerID,
			"error", err,
		)

		return errAuditDown
	}

	slog.InfoContext(s.ctx, "message accepted",
		"correlation_id", msg.CorrelationID,
		"customer_id", msg.CustomerID,
		"message_id", msg.MessageID,
		"recipients", len(msg.Recipients),
		"size", msg.Size,
	)

	s.holding = false
	s.backend.dispatcher.Dispatch(msg, s.backend.release)
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
	if s.holding {
		s.backend.release()
		s.holding = false
	}

	s.from = ""
	s.domain = ""
	s.recipients = nil
	s.auth = Authorization{}
}

func pendingRecipients(recipients []string) []auditdto.Recipient {
	entries := make([]auditdto.Recipient, 0, len(recipients))

	for _, recipient := range recipients {
		entries = append(entries, auditdto.Recipient{
			Email:  recipient,
			Domain: delivery.DomainOf(recipient),
			Status: auditutils.StatusProcessing,
		})
	}

	return entries
}

func MessageID(raw []byte) string {
	message, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(message.Header.Get("Message-ID"))
}
