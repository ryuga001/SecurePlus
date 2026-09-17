package delivery

import (
	"context"
	"errors"
	"log/slog"
	"time"

	auditdto "dpdp-backend/internal/audit/dto/deliveryaudit"
	auditutils "dpdp-backend/internal/audit/utils"
	deliveryutils "dpdp-backend/internal/delivery/utils"
)

type DeliveryService struct {
	engine   Processor
	relay    Sender
	recorder auditdto.Recorder
}

func NewDeliveryService(engine Processor, relay Sender, recorder auditdto.Recorder) *DeliveryService {
	return &DeliveryService{engine: engine, relay: relay, recorder: recorder}
}

func (s *DeliveryService) Process(ctx context.Context, msg EmailMessage) (err error) {
	started := time.Now()

	defer func() {
		if recovered := recover(); recovered != nil {
			slog.ErrorContext(ctx, "delivery panicked",
				"correlation_id", msg.CorrelationID,
				"message_id", msg.MessageID,
				"panic", recovered,
			)

			err = s.complete(ctx, msg, auditdto.Result{
				Status:     auditutils.StatusFailed,
				Recipients: toAuditRecipients(failAll(msg.Recipients, "delivery panicked")),
				Failure:    &auditdto.Failure{Type: auditutils.FailureUnknown, Reason: "delivery panicked"},
			})
		}
	}()

	outcome, processErr := s.engine.Process(ctx, msg)
	if processErr != nil {
		failureType := auditutils.FailureProcessing
		recipients := toAuditRecipients(failAll(msg.Recipients, processErr.Error()))

		if errors.Is(processErr, deliveryutils.ErrAllRecipientsRestricted) {
			failureType = auditutils.FailureRule
			recipients = toAuditRecipients(outcome.Withheld)
		}

		slog.ErrorContext(ctx, "processing failed",
			"correlation_id", msg.CorrelationID,
			"customer_id", msg.CustomerID,
			"failure", failureType,
			"error", processErr,
		)

		return s.complete(ctx, msg, auditdto.Result{
			Status:     auditutils.StatusFailed,
			Recipients: recipients,
			Failure:    &auditdto.Failure{Type: failureType, Reason: processErr.Error()},
		})
	}

	processed := outcome.Message
	cfg := outcome.Config

	dkim := auditdto.DKIM{Domain: cfg.Domain, Selector: cfg.DKIMSelector}

	sink := func(attempt Attempt) {
		if err := s.recorder.RecordAttempt(ctx, msg.CorrelationID, toAuditAttempt(attempt)); err != nil {
			slog.WarnContext(ctx, "delivery attempt not recorded",
				"correlation_id", msg.CorrelationID,
				"error", err,
			)
		}
	}

	results, relayErr := s.relay.Deliver(ctx, processed, cfg, sink)
	if relayErr != nil {
		slog.ErrorContext(ctx, "dkim signing failed",
			"correlation_id", msg.CorrelationID,
			"customer_id", msg.CustomerID,
			"error", relayErr,
		)

		return s.complete(ctx, msg, auditdto.Result{
			Status:     auditutils.StatusFailed,
			Recipients: toAuditRecipients(append(failAll(processed.Recipients, relayErr.Error()), outcome.Withheld...)),
			Failure:    &auditdto.Failure{Type: auditutils.FailureDKIM, Reason: relayErr.Error()},
			DKIM:       dkim,
		})
	}

	dkim.Signed = true

	results = append(results, outcome.Withheld...)

	status := auditutils.StatusSuccess
	var failure *auditdto.Failure

	for _, result := range results {
		if result.Status == deliveryutils.StatusBlocked {
			continue
		}

		if result.Status != deliveryutils.StatusSuccess {
			status = auditutils.StatusFailed
			failure = &auditdto.Failure{
				Type:     auditutils.FailureRelay,
				Reason:   result.Error,
				SMTPCode: result.SMTPCode,
			}

			break
		}
	}

	slog.InfoContext(ctx, "delivery finished",
		"correlation_id", msg.CorrelationID,
		"customer_id", msg.CustomerID,
		"message_id", msg.MessageID,
		"status", status,
		"duration_ms", time.Since(started).Milliseconds(),
	)

	return s.complete(ctx, msg, auditdto.Result{
		Status:     status,
		Recipients: toAuditRecipients(results),
		Failure:    failure,
		DKIM:       dkim,
	})
}

func (s *DeliveryService) complete(ctx context.Context, msg EmailMessage, result auditdto.Result) error {
	if err := s.recorder.Complete(context.WithoutCancel(ctx), msg.CorrelationID, result); err != nil {
		slog.ErrorContext(ctx, "delivery audit completion failed",
			"correlation_id", msg.CorrelationID,
			"error", err,
		)

		return err
	}

	return nil
}

func failAll(recipients []string, reason string) []RecipientResult {
	results := make([]RecipientResult, 0, len(recipients))

	for _, recipient := range recipients {
		results = append(results, RecipientResult{
			Email:  recipient,
			Domain: DomainOf(recipient),
			Status: deliveryutils.StatusFailed,
			Error:  reason,
		})
	}

	return results
}

func toAuditRecipients(results []RecipientResult) []auditdto.Recipient {
	recipients := make([]auditdto.Recipient, 0, len(results))

	for _, result := range results {
		recipients = append(recipients, auditdto.Recipient{
			Email:    result.Email,
			Domain:   result.Domain,
			Status:   result.Status,
			SMTPCode: result.SMTPCode,
			Error:    result.Error,
		})
	}

	return recipients
}

func toAuditAttempt(attempt Attempt) auditdto.Attempt {
	return auditdto.Attempt{
		Number:     attempt.Number,
		StartedAt:  attempt.StartedAt,
		FinishedAt: attempt.FinishedAt,
		MXHost:     attempt.MXHost,
		TLS:        attempt.TLS,
		SMTPCode:   attempt.SMTPCode,
		Error:      attempt.Error,
	}
}
