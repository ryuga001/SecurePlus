//go:build integration

package delivery_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"net/textproto"

	"github.com/redis/go-redis/v9"
	"testing"
	"time"

	auditdto "dpdp-backend/internal/audit/dto/deliveryaudit"
	"dpdp-backend/internal/config"
	"dpdp-backend/internal/delivery"
	smtphandler "dpdp-backend/internal/delivery/handler/smtp"
	"dpdp-backend/tests/testsupport"
)

type stubRecorder struct {
	completeErr error
	completed   int
}

func (r *stubRecorder) Create(context.Context, auditdto.Record) error { return nil }

func (r *stubRecorder) RecordAttempt(context.Context, string, auditdto.Attempt) error { return nil }

func (r *stubRecorder) Complete(context.Context, string, auditdto.Result) error {
	r.completed++

	return r.completeErr
}

type stubProcessor struct{}

func (stubProcessor) Process(_ context.Context, msg delivery.EmailMessage) (delivery.ProcessResult, error) {
	return delivery.ProcessResult{Message: msg, Config: delivery.TenantConfig{Domain: "example.com"}}, nil
}

type stubSender struct{}

func (stubSender) Deliver(_ context.Context, msg delivery.EmailMessage, _ delivery.TenantConfig, _ func(delivery.Attempt)) ([]delivery.RecipientResult, error) {
	results := make([]delivery.RecipientResult, 0, len(msg.Recipients))

	for _, recipient := range msg.Recipients {
		results = append(results, delivery.RecipientResult{Email: recipient, Status: "SUCCESS", SMTPCode: 250})
	}

	return results, nil
}

type stubDomainCache struct{}

func (stubDomainCache) Domain(context.Context, string) (int, int, error) { return 1, 2, nil }

func (stubDomainCache) Remember(context.Context, string, int, int) error { return nil }

type stubDomainStore struct{}

func (stubDomainStore) FindByDomain(context.Context, string) (int, int, error) { return 1, 2, nil }

func deliveryConfig(t *testing.T) config.Delivery {
	t.Helper()

	return config.Delivery{
		Stream:        fmt.Sprintf("delivery:messages:test:%d", time.Now().UnixNano()),
		ConsumerGroup: "delivery-workers",
		Workers:       1,
		ClaimIdle:     200 * time.Millisecond,
		BatchSize:     10,
		BlockTime:     50 * time.Millisecond,
	}
}

func testQueue(t *testing.T) (*delivery.Queue, config.Delivery) {
	t.Helper()

	cfg := deliveryConfig(t)
	queue := delivery.NewQueue(testsupport.Redis(t), cfg)

	if err := queue.EnsureGroup(context.Background()); err != nil {
		t.Fatalf("EnsureGroup returned %v", err)
	}

	return queue, cfg
}

func queuedMessage() delivery.EmailMessage {
	return delivery.EmailMessage{
		CorrelationID: "corr-1",
		MessageID:     "<m@sender.test>",
		CustomerID:    7,
		ConfigID:      9,
		From:          "alice@example.com",
		SenderDomain:  "example.com",
		Recipients:    []string{"bob@recipient.test", "carol@recipient.test"},
		Raw:           []byte("From: alice@example.com\r\nSubject: s\r\n\r\nbody\r\n"),
		ReceivedAt:    time.Now().UTC().Truncate(time.Millisecond),
	}
}

func TestQueueEnsureGroupIsIdempotent(t *testing.T) {
	queue, _ := testQueue(t)

	if err := queue.EnsureGroup(context.Background()); err != nil {
		t.Fatalf("second EnsureGroup returned %v, an existing group must not be an error", err)
	}
}

func TestQueuePublishRoundTripsEveryFieldTheWorkerNeeds(t *testing.T) {
	queue, _ := testQueue(t)
	ctx := context.Background()

	sent := queuedMessage()

	if err := queue.Publish(ctx, sent); err != nil {
		t.Fatalf("Publish returned %v", err)
	}

	queued, err := queue.Consume(ctx, "consumer-1", 10, time.Second)
	if err != nil {
		t.Fatalf("Consume returned %v", err)
	}

	if len(queued) != 1 {
		t.Fatalf("consumed %d entries, want 1", len(queued))
	}

	got := queued[0].Message

	if queued[0].Entry == "" {
		t.Fatal("stream entry id must be populated for acknowledgement")
	}
	if got.CorrelationID != sent.CorrelationID {
		t.Fatalf("correlation id = %q, want %q", got.CorrelationID, sent.CorrelationID)
	}
	if got.MessageID != sent.MessageID {
		t.Fatalf("message id = %q, want %q", got.MessageID, sent.MessageID)
	}
	if got.CustomerID != sent.CustomerID || got.ConfigID != sent.ConfigID {
		t.Fatalf("tenant = (%d,%d), want (%d,%d)", got.CustomerID, got.ConfigID, sent.CustomerID, sent.ConfigID)
	}
	if got.From != sent.From || got.SenderDomain != sent.SenderDomain {
		t.Fatalf("sender = %q/%q", got.From, got.SenderDomain)
	}
	if len(got.Recipients) != 2 || got.Recipients[0] != sent.Recipients[0] {
		t.Fatalf("recipients = %v", got.Recipients)
	}
	if string(got.Raw) != string(sent.Raw) {
		t.Fatalf("raw body did not survive the stream")
	}
}

func TestQueuePublishFailsWhenRedisIsUnavailable(t *testing.T) {
	cfg := deliveryConfig(t)
	client := testsupport.Redis(t)
	queue := delivery.NewQueue(client, cfg)

	client.Close()

	if err := queue.Publish(context.Background(), queuedMessage()); err == nil {
		t.Fatal("Publish must report failure when Redis is unavailable")
	}
}

func TestQueueAckClearsPendingEntry(t *testing.T) {
	queue, _ := testQueue(t)
	ctx := context.Background()

	if err := queue.Publish(ctx, queuedMessage()); err != nil {
		t.Fatalf("Publish returned %v", err)
	}

	queued, err := queue.Consume(ctx, "consumer-1", 10, time.Second)
	if err != nil || len(queued) != 1 {
		t.Fatalf("Consume returned %v / %d entries", err, len(queued))
	}

	pending, err := queue.Pending(ctx)
	if err != nil {
		t.Fatalf("Pending returned %v", err)
	}
	if pending != 1 {
		t.Fatalf("pending = %d after read, want 1 — a read entry must stay pending until acked", pending)
	}

	if err := queue.Ack(ctx, queued[0].Entry); err != nil {
		t.Fatalf("Ack returned %v", err)
	}

	pending, err = queue.Pending(ctx)
	if err != nil {
		t.Fatalf("Pending returned %v", err)
	}
	if pending != 0 {
		t.Fatalf("pending = %d after ack, want 0", pending)
	}
}

func TestQueueReclaimTakesOverEntryAbandonedByADeadWorker(t *testing.T) {
	queue, cfg := testQueue(t)
	ctx := context.Background()

	if err := queue.Publish(ctx, queuedMessage()); err != nil {
		t.Fatalf("Publish returned %v", err)
	}

	if _, err := queue.Consume(ctx, "dead-worker", 10, time.Second); err != nil {
		t.Fatalf("Consume returned %v", err)
	}

	time.Sleep(cfg.ClaimIdle + 100*time.Millisecond)

	reclaimed, err := queue.Reclaim(ctx, "live-worker", cfg.ClaimIdle, 10)
	if err != nil {
		t.Fatalf("Reclaim returned %v", err)
	}

	if len(reclaimed) != 1 {
		t.Fatalf("reclaimed %d entries, want the one the dead worker never acked", len(reclaimed))
	}
	if reclaimed[0].Message.CorrelationID != "corr-1" {
		t.Fatalf("reclaimed the wrong message: %+v", reclaimed[0].Message)
	}
}

func TestWorkerAcksOnlyAfterProcessingSucceeds(t *testing.T) {
	queue, cfg := testQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recorder := &stubRecorder{}
	service := delivery.NewDeliveryService(stubProcessor{}, stubSender{}, recorder)

	pool := delivery.NewWorkerPool(queue, service, cfg)
	pool.Start(ctx)

	if err := queue.Publish(ctx, queuedMessage()); err != nil {
		t.Fatalf("Publish returned %v", err)
	}

	waitForProcessed(t, pool, 1)

	pending, err := queue.Pending(ctx)
	if err != nil {
		t.Fatalf("Pending returned %v", err)
	}
	if pending != 0 {
		t.Fatalf("pending = %d after successful processing, want 0", pending)
	}

	cancel()
	pool.Wait()

	if processed, failures := pool.Stats(); processed != 1 || failures != 0 {
		t.Fatalf("stats = processed %d failures %d, want 1/0", processed, failures)
	}
}

func TestWorkerLeavesEntryPendingWhenTheOutcomeCannotBeRecorded(t *testing.T) {
	queue, cfg := testQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recorder := &stubRecorder{completeErr: errors.New("mongo down")}
	service := delivery.NewDeliveryService(stubProcessor{}, stubSender{}, recorder)

	pool := delivery.NewWorkerPool(queue, service, cfg)
	pool.Start(ctx)

	if err := queue.Publish(ctx, queuedMessage()); err != nil {
		t.Fatalf("Publish returned %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, failures := pool.Stats(); failures > 0 {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	cancel()
	pool.Wait()

	pending, err := queue.Pending(context.Background())
	if err != nil {
		t.Fatalf("Pending returned %v", err)
	}

	if pending == 0 {
		t.Fatal("an unrecorded delivery must stay pending so another worker can reclaim it")
	}

	if processed, _ := pool.Stats(); processed != 0 {
		t.Fatalf("processed = %d, a failed delivery must not count as processed", processed)
	}
}

func TestWorkerStopsConsumingOnCancellation(t *testing.T) {
	queue, cfg := testQueue(t)
	ctx, cancel := context.WithCancel(context.Background())

	pool := delivery.NewWorkerPool(queue, delivery.NewDeliveryService(stubProcessor{}, stubSender{}, &stubRecorder{}), cfg)
	pool.Start(ctx)

	cancel()

	done := make(chan struct{})

	go func() {
		pool.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("workers did not stop within the shutdown budget")
	}
}

func TestSMTPRejectsWithTemporaryFailureWhenTheQueueIsUnavailable(t *testing.T) {
	cfg := deliveryConfig(t)
	client := testsupport.Redis(t)
	queue := delivery.NewQueue(client, cfg)

	if err := queue.EnsureGroup(context.Background()); err != nil {
		t.Fatalf("EnsureGroup returned %v", err)
	}

	client.Close()

	addr := freeAddr(t)

	server := smtphandler.NewServer(
		config.SMTPServer{
			Addr:          addr,
			MaxSize:       1024 * 1024,
			MaxRecipients: 10,
			ReadTimeout:   5 * time.Second,
			WriteTimeout:  5 * time.Second,
		},
		smtphandler.NewBackend(
			smtphandler.NewAuthorizer(stubDomainCache{}, stubDomainStore{}),
			delivery.NewAcceptor(&stubRecorder{}, queue),
			1024*1024,
			10,
		),
		"test.local",
	)

	go server.ListenAndServe()

	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		server.Shutdown(shutdownCtx)
	})

	waitForListener(t, addr)

	err := sendOne(addr, "alice@example.com", "bob@recipient.test")

	if err == nil {
		t.Fatal("DATA must not succeed when the message never reached Redis")
	}

	var protoErr *textproto.Error
	if !errors.As(err, &protoErr) || protoErr.Code != 451 {
		t.Fatalf("error = %v, want SMTP 451", err)
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	addr := listener.Addr().String()
	listener.Close()

	return addr
}

func sendOne(addr, from, to string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}

	defer client.Close()

	if err := client.Hello("sender.test"); err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}

	if _, err := writer.Write([]byte("From: " + from + "\r\nSubject: s\r\n\r\nbody\r\n")); err != nil {
		return err
	}

	return writer.Close()
}

func waitForProcessed(t *testing.T, pool *delivery.WorkerPool, want int64) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		if processed, _ := pool.Stats(); processed >= want {
			return
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("workers did not process %d messages within the deadline", want)
}

func smtpServerFor(t *testing.T, queue *delivery.Queue) string {
	t.Helper()

	addr := freeAddr(t)

	server := smtphandler.NewServer(
		config.SMTPServer{
			Addr:          addr,
			MaxSize:       1024 * 1024,
			MaxRecipients: 10,
			ReadTimeout:   5 * time.Second,
			WriteTimeout:  5 * time.Second,
		},
		smtphandler.NewBackend(
			smtphandler.NewAuthorizer(stubDomainCache{}, stubDomainStore{}),
			delivery.NewAcceptor(&stubRecorder{}, queue),
			1024*1024,
			10,
		),
		"test.local",
	)

	go server.ListenAndServe()

	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		server.Shutdown(shutdownCtx)
	})

	waitForListener(t, addr)

	return addr
}

func TestSMTPAcceptsOnlyAfterTheMessageReachesRedis(t *testing.T) {
	queue, _ := testQueue(t)
	addr := smtpServerFor(t, queue)

	if err := sendOne(addr, "alice@example.com", "bob@recipient.test"); err != nil {
		t.Fatalf("DATA returned %v, want 250 once Redis accepted the message", err)
	}

	queued, err := queue.Consume(context.Background(), "assert", 10, time.Second)
	if err != nil {
		t.Fatalf("Consume returned %v", err)
	}

	if len(queued) != 1 {
		t.Fatalf("stream holds %d entries, want the accepted message", len(queued))
	}

	msg := queued[0].Message

	if msg.CorrelationID == "" {
		t.Fatal("the receiver must mint a correlation id before publishing")
	}
	if msg.From != "alice@example.com" {
		t.Fatalf("envelope from = %q", msg.From)
	}
	if len(msg.Recipients) != 1 || msg.Recipients[0] != "bob@recipient.test" {
		t.Fatalf("recipients = %v", msg.Recipients)
	}
	if msg.CustomerID != 1 || msg.ConfigID != 2 {
		t.Fatalf("tenant = (%d,%d), want the authorized sender's tenant", msg.CustomerID, msg.ConfigID)
	}
	if len(msg.Raw) == 0 {
		t.Fatal("the raw message body must travel on the stream")
	}
}

func TestSMTPPublishesADistinctCorrelationIdPerMessage(t *testing.T) {
	queue, _ := testQueue(t)
	addr := smtpServerFor(t, queue)

	for range 2 {
		if err := sendOne(addr, "alice@example.com", "bob@recipient.test"); err != nil {
			t.Fatalf("DATA returned %v", err)
		}
	}

	queued, err := queue.Consume(context.Background(), "assert", 10, time.Second)
	if err != nil {
		t.Fatalf("Consume returned %v", err)
	}

	if len(queued) != 2 {
		t.Fatalf("stream holds %d entries, want 2", len(queued))
	}

	if queued[0].Message.CorrelationID == queued[1].Message.CorrelationID {
		t.Fatal("each accepted message must carry its own correlation id")
	}
	if queued[0].Entry == queued[1].Entry {
		t.Fatal("each entry must have its own stream id")
	}
}

func TestWorkerPoolRecoversAMessageAbandonedByADeadWorker(t *testing.T) {
	queue, cfg := testQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := queue.Publish(ctx, queuedMessage()); err != nil {
		t.Fatalf("Publish returned %v", err)
	}

	if _, err := queue.Consume(ctx, "dead-worker", 10, time.Second); err != nil {
		t.Fatalf("Consume returned %v", err)
	}

	pending, err := queue.Pending(ctx)
	if err != nil || pending != 1 {
		t.Fatalf("pending = %d (err %v), want the entry stranded by the dead worker", pending, err)
	}

	recorder := &stubRecorder{}
	pool := delivery.NewWorkerPool(queue, delivery.NewDeliveryService(stubProcessor{}, stubSender{}, recorder), cfg)
	pool.Start(ctx)

	waitForProcessed(t, pool, 1)

	cancel()
	pool.Wait()

	pending, err = queue.Pending(context.Background())
	if err != nil {
		t.Fatalf("Pending returned %v", err)
	}

	if pending != 0 {
		t.Fatalf("pending = %d, the reclaimed entry must be acked once processed", pending)
	}

	if recorder.completed != 1 {
		t.Fatalf("audit completions = %d, want the reclaimed message recorded once", recorder.completed)
	}
}

func TestWorkerDiscardsAnUnreadableEntryInsteadOfLoopingForever(t *testing.T) {
	queue, cfg := testQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := testsupport.Redis(t)

	unreadable := &redis.XAddArgs{
		Stream: cfg.Stream,
		Values: map[string]any{"junk": "no correlation id"},
	}

	if err := client.XAdd(ctx, unreadable).Err(); err != nil {
		t.Fatalf("XAdd returned %v", err)
	}

	pool := delivery.NewWorkerPool(queue, delivery.NewDeliveryService(stubProcessor{}, stubSender{}, &stubRecorder{}), cfg)
	pool.Start(ctx)

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		pending, err := queue.Pending(ctx)
		if err == nil && pending == 0 {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	cancel()
	pool.Wait()

	pending, err := queue.Pending(context.Background())
	if err != nil {
		t.Fatalf("Pending returned %v", err)
	}

	if pending != 0 {
		t.Fatalf("pending = %d, an entry the worker cannot read must be acked rather than replayed", pending)
	}
}
