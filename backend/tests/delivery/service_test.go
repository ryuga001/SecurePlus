package delivery_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	auditdto "dpdp-backend/internal/audit/dto/deliveryaudit"
	auditutils "dpdp-backend/internal/audit/utils"
	"dpdp-backend/internal/delivery"
	deliveryutils "dpdp-backend/internal/delivery/utils"
)

type fakeRecorder struct {
	mu          sync.Mutex
	completeErr error
	results     []auditdto.Result
}

func (r *fakeRecorder) Create(context.Context, auditdto.Record) error { return nil }

func (r *fakeRecorder) RecordAttempt(context.Context, string, auditdto.Attempt) error { return nil }

func (r *fakeRecorder) Complete(_ context.Context, _ string, result auditdto.Result) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.results = append(r.results, result)

	return r.completeErr
}

func (r *fakeRecorder) completed() []auditdto.Result {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]auditdto.Result(nil), r.results...)
}

type fakeProcessor struct {
	process func(context.Context, delivery.EmailMessage) (delivery.ProcessResult, error)
}

func (p fakeProcessor) Process(ctx context.Context, msg delivery.EmailMessage) (delivery.ProcessResult, error) {
	if p.process == nil {
		return delivery.ProcessResult{
			Message: msg,
			Config:  delivery.TenantConfig{Domain: "example.com", DKIMSelector: "dpdp"},
		}, nil
	}

	return p.process(ctx, msg)
}

type fakeSender struct {
	deliver func(context.Context, delivery.EmailMessage) ([]delivery.RecipientResult, error)
}

func (s fakeSender) Deliver(
	ctx context.Context,
	msg delivery.EmailMessage,
	_ delivery.TenantConfig,
	_ func(delivery.Attempt),
) ([]delivery.RecipientResult, error) {
	if s.deliver == nil {
		results := make([]delivery.RecipientResult, 0, len(msg.Recipients))

		for _, recipient := range msg.Recipients {
			results = append(results, delivery.RecipientResult{
				Email:    recipient,
				Status:   deliveryutils.StatusSuccess,
				SMTPCode: 250,
			})
		}

		return results, nil
	}

	return s.deliver(ctx, msg)
}

func serviceMessage() delivery.EmailMessage {
	return delivery.EmailMessage{
		CorrelationID: "corr-1",
		MessageID:     "<m@sender.test>",
		CustomerID:    7,
		ConfigID:      9,
		From:          "alice@example.com",
		SenderDomain:  "example.com",
		Recipients:    []string{"bob@recipient.test"},
		Raw:           []byte("From: alice@example.com\r\nSubject: s\r\n\r\nbody\r\n"),
	}
}

func onlyResult(t *testing.T, recorder *fakeRecorder) auditdto.Result {
	t.Helper()

	results := recorder.completed()
	if len(results) != 1 {
		t.Fatalf("audit completions = %d, want exactly one terminal record", len(results))
	}

	return results[0]
}

func TestProcessAcksAfterSuccessfulDelivery(t *testing.T) {
	recorder := &fakeRecorder{}
	service := delivery.NewDeliveryService(fakeProcessor{}, fakeSender{}, recorder)

	if err := service.Process(context.Background(), serviceMessage()); err != nil {
		t.Fatalf("Process returned %v, want nil", err)
	}

	result := onlyResult(t, recorder)

	if result.Status != auditutils.StatusSuccess {
		t.Fatalf("status = %q, want SUCCESS", result.Status)
	}
	if !result.DKIM.Signed {
		t.Fatal("a delivered message must record that it was DKIM signed")
	}
}

func TestProcessReportsNoErrorWhenDeliveryItselfFailed(t *testing.T) {
	recorder := &fakeRecorder{}
	relay := fakeSender{
		deliver: func(_ context.Context, msg delivery.EmailMessage) ([]delivery.RecipientResult, error) {
			return []delivery.RecipientResult{{
				Email:     msg.Recipients[0],
				Status:    deliveryutils.StatusFailed,
				SMTPCode:  550,
				Error:     "mailbox unavailable",
				Permanent: true,
			}}, nil
		},
	}

	service := delivery.NewDeliveryService(fakeProcessor{}, relay, recorder)

	if err := service.Process(context.Background(), serviceMessage()); err != nil {
		t.Fatalf("Process returned %v — a recorded delivery failure is terminal and must be acked, not retried", err)
	}

	result := onlyResult(t, recorder)

	if result.Status != auditutils.StatusFailed {
		t.Fatalf("status = %q, want FAILED", result.Status)
	}
	if result.Failure == nil || result.Failure.Type != auditutils.FailureRelay {
		t.Fatalf("failure = %+v, want a RELAY failure", result.Failure)
	}
}

func TestProcessReportsNoErrorWhenPolicyWithheldEveryRecipient(t *testing.T) {
	recorder := &fakeRecorder{}
	engine := fakeProcessor{
		process: func(_ context.Context, msg delivery.EmailMessage) (delivery.ProcessResult, error) {
			return delivery.ProcessResult{
				Withheld: []delivery.RecipientResult{{
					Email:     msg.Recipients[0],
					Status:    deliveryutils.StatusBlocked,
					Error:     "withheld by policy: Data Loss",
					Permanent: true,
				}},
			}, deliveryutils.ErrAllRecipientsRestricted
		},
	}

	service := delivery.NewDeliveryService(engine, fakeSender{}, recorder)

	if err := service.Process(context.Background(), serviceMessage()); err != nil {
		t.Fatalf("Process returned %v, a policy block is a terminal outcome", err)
	}

	result := onlyResult(t, recorder)

	if result.Failure == nil || result.Failure.Type != auditutils.FailureRule {
		t.Fatalf("failure = %+v, want a RULE failure", result.Failure)
	}
	if len(result.Recipients) != 1 || result.Recipients[0].Status != deliveryutils.StatusBlocked {
		t.Fatalf("recipients = %+v, want the withheld recipient recorded as BLOCKED", result.Recipients)
	}
}

func TestProcessReportsNoErrorWhenProcessingFailed(t *testing.T) {
	recorder := &fakeRecorder{}
	engine := fakeProcessor{
		process: func(context.Context, delivery.EmailMessage) (delivery.ProcessResult, error) {
			return delivery.ProcessResult{}, errors.New("policy store unreachable")
		},
	}

	service := delivery.NewDeliveryService(engine, fakeSender{}, recorder)

	if err := service.Process(context.Background(), serviceMessage()); err != nil {
		t.Fatalf("Process returned %v, the outcome was recorded so the entry must be acked", err)
	}

	result := onlyResult(t, recorder)

	if result.Failure == nil || result.Failure.Type != auditutils.FailureProcessing {
		t.Fatalf("failure = %+v, want a PROCESSING failure", result.Failure)
	}
}

func TestProcessReportsNoErrorWhenSigningFailed(t *testing.T) {
	recorder := &fakeRecorder{}
	relay := fakeSender{
		deliver: func(context.Context, delivery.EmailMessage) ([]delivery.RecipientResult, error) {
			return nil, errors.New("dkim private key could not be parsed")
		},
	}

	service := delivery.NewDeliveryService(fakeProcessor{}, relay, recorder)

	if err := service.Process(context.Background(), serviceMessage()); err != nil {
		t.Fatalf("Process returned %v", err)
	}

	result := onlyResult(t, recorder)

	if result.Failure == nil || result.Failure.Type != auditutils.FailureDKIM {
		t.Fatalf("failure = %+v, want a DKIM failure", result.Failure)
	}
	if result.DKIM.Signed {
		t.Fatal("a message that failed signing must not be recorded as signed")
	}
}

func TestProcessReturnsErrorOnlyWhenTheOutcomeCannotBeRecorded(t *testing.T) {
	recorder := &fakeRecorder{completeErr: errors.New("mongo down")}
	service := delivery.NewDeliveryService(fakeProcessor{}, fakeSender{}, recorder)

	if err := service.Process(context.Background(), serviceMessage()); err == nil {
		t.Fatal("an unrecorded outcome must be reported so the worker leaves the entry pending")
	}

	if len(recorder.completed()) != 1 {
		t.Fatalf("audit completions = %d, want one attempt", len(recorder.completed()))
	}
}

func TestProcessRecoversFromAPanicAndStillRecordsTheOutcome(t *testing.T) {
	recorder := &fakeRecorder{}
	engine := fakeProcessor{
		process: func(context.Context, delivery.EmailMessage) (delivery.ProcessResult, error) {
			panic("policy compiler exploded")
		},
	}

	service := delivery.NewDeliveryService(engine, fakeSender{}, recorder)

	if err := service.Process(context.Background(), serviceMessage()); err != nil {
		t.Fatalf("Process returned %v, a panic is recorded and acked rather than replayed forever", err)
	}

	result := onlyResult(t, recorder)

	if result.Status != auditutils.StatusFailed {
		t.Fatalf("status = %q, want FAILED", result.Status)
	}
	if result.Failure == nil || result.Failure.Type != auditutils.FailureUnknown {
		t.Fatalf("failure = %+v, want an UNKNOWN failure", result.Failure)
	}
}
