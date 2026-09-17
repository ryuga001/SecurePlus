package delivery

import (
	"context"
	"errors"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	auditdto "dpdp-backend/internal/audit/dto/deliveryaudit"
	auditutils "dpdp-backend/internal/audit/utils"
)

var (
	ErrAuditUnavailable = errors.New("delivery audit unavailable")
	ErrQueueUnavailable = errors.New("delivery queue unavailable")
)

type Submission struct {
	CustomerID   int
	ConfigID     int
	From         string
	SenderDomain string
	Recipients   []string
	Raw          []byte
}

type Acceptor struct {
	recorder auditdto.Recorder
	queue    *Queue
}

func NewAcceptor(recorder auditdto.Recorder, queue *Queue) *Acceptor {
	return &Acceptor{recorder: recorder, queue: queue}
}

func (a *Acceptor) Accept(ctx context.Context, submission Submission) (EmailMessage, error) {
	msg := EmailMessage{
		CorrelationID: uuid.NewString(),
		MessageID:     MessageIDOf(submission.Raw),
		CustomerID:    submission.CustomerID,
		ConfigID:      submission.ConfigID,
		From:          submission.From,
		SenderDomain:  submission.SenderDomain,
		Recipients:    append([]string(nil), submission.Recipients...),
		Raw:           submission.Raw,
		Size:          int64(len(submission.Raw)),
		ReceivedAt:    time.Now().UTC(),
	}

	record := auditdto.Record{
		CorrelationID: msg.CorrelationID,
		MessageID:     msg.MessageID,
		CustomerID:    msg.CustomerID,
		ConfigID:      msg.ConfigID,
		From:          msg.From,
		SenderDomain:  msg.SenderDomain,
		Recipients:    PendingRecipients(msg.Recipients),
		Size:          msg.Size,
	}

	if err := a.recorder.Create(ctx, record); err != nil {
		slog.ErrorContext(ctx, "delivery audit create failed",
			"correlation_id", msg.CorrelationID,
			"customer_id", msg.CustomerID,
			"error", err,
		)

		return msg, ErrAuditUnavailable
	}

	if err := a.queue.Publish(ctx, msg); err != nil {
		slog.ErrorContext(ctx, "delivery queue publish failed",
			"correlation_id", msg.CorrelationID,
			"customer_id", msg.CustomerID,
			"error", err,
		)

		a.recorder.Complete(ctx, msg.CorrelationID, auditdto.Result{
			Status:     auditutils.StatusFailed,
			Recipients: FailedRecipients(msg.Recipients, ErrQueueUnavailable.Error()),
			Failure: &auditdto.Failure{
				Type:   auditutils.FailureProcessing,
				Reason: ErrQueueUnavailable.Error(),
			},
		})

		return msg, ErrQueueUnavailable
	}

	slog.InfoContext(ctx, "message accepted",
		"correlation_id", msg.CorrelationID,
		"customer_id", msg.CustomerID,
		"message_id", msg.MessageID,
		"recipients", len(msg.Recipients),
		"size", msg.Size,
	)

	return msg, nil
}

func MessageIDOf(raw []byte) string {
	message, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(message.Header.Get("Message-ID"))
}

func PendingRecipients(recipients []string) []auditdto.Recipient {
	entries := make([]auditdto.Recipient, 0, len(recipients))

	for _, recipient := range recipients {
		entries = append(entries, auditdto.Recipient{
			Email:  recipient,
			Domain: DomainOf(recipient),
			Status: auditutils.StatusProcessing,
		})
	}

	return entries
}

func FailedRecipients(recipients []string, reason string) []auditdto.Recipient {
	entries := make([]auditdto.Recipient, 0, len(recipients))

	for _, recipient := range recipients {
		entries = append(entries, auditdto.Recipient{
			Email:  recipient,
			Domain: DomainOf(recipient),
			Status: auditutils.StatusFailed,
			Error:  reason,
		})
	}

	return entries
}
