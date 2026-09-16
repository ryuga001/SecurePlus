package dispatch

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	auditdto "dpdp-backend/internal/audit/dto/deliveryaudit"
	auditutils "dpdp-backend/internal/audit/utils"
	"dpdp-backend/internal/delivery/dto/delivery"
	deliveryutils "dpdp-backend/internal/delivery/utils"
)

type Dispatcher struct {
	engine   delivery.Processor
	relay    delivery.Sender
	recorder auditdto.Recorder
	ctx      context.Context
	wg       sync.WaitGroup
}

func NewDispatcher(ctx context.Context, engine delivery.Processor, relay delivery.Sender, recorder auditdto.Recorder) *Dispatcher {
	return &Dispatcher{engine: engine, relay: relay, recorder: recorder, ctx: ctx}
}

func (d *Dispatcher) Dispatch(msg delivery.EmailMessage, done func()) {
	d.wg.Add(1)

	go func() {
		defer d.wg.Done()

		if done != nil {
			defer done()
		}

		defer func() {
			if recovered := recover(); recovered != nil {
				slog.ErrorContext(d.ctx, "delivery panicked",
					"correlation_id", msg.CorrelationID,
					"panic", recovered,
				)

				d.complete(msg, auditdto.Result{
					Status:     auditutils.StatusFailed,
					Recipients: toAuditRecipients(failAll(msg.Recipients, "delivery panicked")),
					Failure:    &auditdto.Failure{Type: auditutils.FailureUnknown, Reason: "delivery panicked"},
				})
			}
		}()

		d.run(msg)
	}()
}

func (d *Dispatcher) Wait() {
	d.wg.Wait()
}

func (d *Dispatcher) run(msg delivery.EmailMessage) {
	outcome, err := d.engine.Process(d.ctx, msg)
	if err != nil {
		failureType := auditutils.FailureProcessing
		recipients := toAuditRecipients(failAll(msg.Recipients, err.Error()))

		if errors.Is(err, deliveryutils.ErrAllRecipientsRestricted) {
			failureType = auditutils.FailureRule
			recipients = toAuditRecipients(outcome.Withheld)
		}

		slog.ErrorContext(d.ctx, "processing failed",
			"correlation_id", msg.CorrelationID,
			"customer_id", msg.CustomerID,
			"failure", failureType,
			"error", err,
		)

		d.complete(msg, auditdto.Result{
			Status:     auditutils.StatusFailed,
			Recipients: recipients,
			Failure:    &auditdto.Failure{Type: failureType, Reason: err.Error()},
		})

		return
	}

	processed := outcome.Message
	cfg := outcome.Config

	dkim := auditdto.DKIM{Domain: cfg.Domain, Selector: cfg.DKIMSelector}

	sink := func(attempt delivery.Attempt) {
		if err := d.recorder.RecordAttempt(d.ctx, msg.CorrelationID, toAuditAttempt(attempt)); err != nil {
			slog.WarnContext(d.ctx, "delivery attempt not recorded",
				"correlation_id", msg.CorrelationID,
				"error", err,
			)
		}
	}

	results, err := d.relay.Deliver(d.ctx, processed, cfg, sink)
	if err != nil {
		slog.ErrorContext(d.ctx, "dkim signing failed",
			"correlation_id", msg.CorrelationID,
			"customer_id", msg.CustomerID,
			"error", err,
		)

		d.complete(msg, auditdto.Result{
			Status:     auditutils.StatusFailed,
			Recipients: toAuditRecipients(append(failAll(processed.Recipients, err.Error()), outcome.Withheld...)),
			Failure:    &auditdto.Failure{Type: auditutils.FailureDKIM, Reason: err.Error()},
			DKIM:       dkim,
		})

		return
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

	slog.InfoContext(d.ctx, "delivery finished",
		"correlation_id", msg.CorrelationID,
		"customer_id", msg.CustomerID,
		"message_id", msg.MessageID,
		"status", status,
	)

	d.complete(msg, auditdto.Result{
		Status:     status,
		Recipients: toAuditRecipients(results),
		Failure:    failure,
		DKIM:       dkim,
	})
}

func (d *Dispatcher) complete(msg delivery.EmailMessage, result auditdto.Result) {
	if err := d.recorder.Complete(context.WithoutCancel(d.ctx), msg.CorrelationID, result); err != nil {
		slog.ErrorContext(d.ctx, "delivery audit completion failed",
			"correlation_id", msg.CorrelationID,
			"error", err,
		)
	}
}

func failAll(recipients []string, reason string) []delivery.RecipientResult {
	results := make([]delivery.RecipientResult, 0, len(recipients))

	for _, recipient := range recipients {
		results = append(results, delivery.RecipientResult{
			Email:  recipient,
			Domain: delivery.DomainOf(recipient),
			Status: deliveryutils.StatusFailed,
			Error:  reason,
		})
	}

	return results
}

func toAuditRecipients(results []delivery.RecipientResult) []auditdto.Recipient {
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

func toAuditAttempt(attempt delivery.Attempt) auditdto.Attempt {
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
