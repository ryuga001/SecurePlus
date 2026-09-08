//go:build integration

package delivery_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"gorm.io/gorm"

	providerrepo "dpdp-backend/internal/admin/repositories/emailprovider"
	providersvc "dpdp-backend/internal/admin/services/emailprovider"
	auditrepo "dpdp-backend/internal/audit/repositories/deliveryaudit"
	auditsvc "dpdp-backend/internal/audit/services/deliveryaudit"
	auditutils "dpdp-backend/internal/audit/utils"
	"dpdp-backend/internal/config"
	"dpdp-backend/internal/db"
	"dpdp-backend/internal/delivery"
	"dpdp-backend/internal/delivery/engine"
	"dpdp-backend/internal/delivery/receiver"
	"dpdp-backend/internal/delivery/relay"
	"dpdp-backend/tests/testsupport"
)

type harness struct {
	database   *gorm.DB
	mongo      *mongodriver.Client
	addr       string
	dispatcher *delivery.Dispatcher
	customer   db.Customer
}

func mailpitAddr(t *testing.T) string {
	t.Helper()

	addr := os.Getenv("TEST_MAILPIT_ADDR")
	if addr == "" {
		t.Skip("TEST_MAILPIT_ADDR is not set")
	}

	return addr
}

func setup(t *testing.T, mxOverride string) harness {
	t.Helper()

	database := testsupport.Postgres(t)
	rdb := testsupport.Redis(t)
	client := testsupport.Mongo(t)

	testsupport.Reset(t, database, rdb)
	testsupport.ResetMongo(t, client)

	t.Cleanup(func() {
		testsupport.Reset(t, database, rdb)
		testsupport.ResetMongo(t, client)
	})

	recorder := auditsvc.NewDeliveryAuditService(auditrepo.NewDeliveryAuditRepository(client, testsupport.MongoDatabase()))
	if err := recorder.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("index creation failed: %v", err)
	}

	repository := providerrepo.NewEmailProviderRepository(database)
	registry := providerrepo.NewRedisRepository(rdb)

	configurations := delivery.NewConfigurationStore(repository, receiver.ErrDomainUnknown)
	authorizer := receiver.NewAuthorizer(delivery.NewDomainLookup(registry), configurations)

	relayCfg := config.Relay{
		HELOHost:          "test.local",
		DialTimeout:       5 * time.Second,
		DNSTimeout:        2 * time.Second,
		MaxAttempts:       2,
		BackoffInitial:    50 * time.Millisecond,
		BackoffMultiplier: 2,
		BackoffMax:        200 * time.Millisecond,
		MXOverride:        mxOverride,
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	dispatcher := delivery.NewDispatcher(
		ctx,
		engine.NewEngine(engine.NewConfigCache(configurations, rdb)),
		relay.NewRelay(relayCfg),
		recorder,
	)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	addr := listener.Addr().String()
	listener.Close()

	server := receiver.NewServer(
		config.SMTPServer{
			Addr:          addr,
			MaxSize:       1024 * 1024,
			MaxRecipients: 10,
			MaxDeliveries: 4,
			ReadTimeout:   10 * time.Second,
			WriteTimeout:  10 * time.Second,
		},
		receiver.NewBackend(authorizer, recorder, dispatcher, 4, 1024*1024, 10),
		"test.local",
	)

	go server.ListenAndServe()

	t.Cleanup(func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		server.Shutdown(shutdownCtx)
	})

	waitForListener(t, addr)

	return harness{
		database:   database,
		mongo:      client,
		addr:       addr,
		dispatcher: dispatcher,
		customer:   testsupport.Customer(t, database),
	}
}

func waitForListener(t *testing.T, addr string) {
	t.Helper()

	for range 50 {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("smtp server did not start on %s", addr)
}

func (h harness) configuration(t *testing.T, domain string, withKey bool) db.EmailProviderConfiguration {
	t.Helper()

	service := providersvc.NewEmailProviderService(
		h.database,
		providerrepo.NewEmailProviderRepository(h.database),
		providerrepo.NewRedisRepository(testsupport.Redis(t)),
		config.Auth{Issuer: "dpdp"},
	)

	row, err := service.Create(context.Background(), h.customer.ID, providersvc.ConfigurationInput{
		Name:     "Corporate",
		Domain:   domain,
		Provider: "gmail",
	})
	if err != nil {
		t.Fatalf("configuration create failed: %v", err)
	}

	if withKey {
		if _, err := service.GenerateDKIM(context.Background(), h.customer.ID, row.ID); err != nil {
			t.Fatalf("dkim generation failed: %v", err)
		}
	}

	return row
}

func (h harness) send(t *testing.T, from string, recipients []string, body string) error {
	t.Helper()

	client, err := smtp.Dial(h.addr)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}

	defer client.Close()

	if err := client.Hello("sender.test"); err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return err
	}

	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}

	if _, err := writer.Write([]byte(body)); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	return client.Quit()
}

func (h harness) audit(t *testing.T, wantStatus string) bson.M {
	t.Helper()

	collection := h.mongo.Database(testsupport.MongoDatabase()).Collection(auditutils.DeliveryAuditCollection)

	for range 100 {
		var record bson.M

		err := collection.FindOne(context.Background(), bson.D{}).Decode(&record)
		if err == nil {
			if wantStatus == "" || record["status"] == wantStatus {
				return record
			}
		} else if !errors.Is(err, mongodriver.ErrNoDocuments) {
			t.Fatalf("audit lookup failed: %v", err)
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("no audit record reached status %q", wantStatus)

	return nil
}

func (h harness) auditCount(t *testing.T) int64 {
	t.Helper()

	total, err := h.mongo.Database(testsupport.MongoDatabase()).
		Collection(auditutils.DeliveryAuditCollection).
		CountDocuments(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}

	return total
}

func message(from, to, subject string) string {
	return fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMessage-ID: <%d@sender.test>\r\nDate: Mon, 08 Sep 2026 09:00:00 +0000\r\n\r\nhello\r\n",
		from, to, subject, time.Now().UnixNano(),
	)
}

func TestIntegrationDeliversAndRecordsSuccess(t *testing.T) {
	h := setup(t, mailpitAddr(t))
	h.configuration(t, "example.com", true)

	from := "alice@example.com"
	to := "bob@recipient.test"

	if err := h.send(t, from, []string{to}, message(from, to, "delivered")); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	record := h.audit(t, delivery.StatusSuccess)

	if record["from"] != from {
		t.Fatalf("from = %v", record["from"])
	}
	if record["correlation_id"] == "" {
		t.Fatal("correlation id is empty")
	}
	if !strings.Contains(record["message_id"].(string), "@sender.test") {
		t.Fatalf("message id = %v", record["message_id"])
	}

	dkim, ok := record["dkim"].(bson.M)
	if !ok || dkim["selector"] != delivery.DefaultDKIMSelector || dkim["signed"] != true {
		t.Fatalf("dkim = %v", record["dkim"])
	}

	recipients, ok := record["recipients"].(bson.A)
	if !ok || len(recipients) != 1 {
		t.Fatalf("recipients = %v", record["recipients"])
	}

	first := recipients[0].(bson.M)
	if first["status"] != delivery.StatusSuccess || first["smtp_code"].(int32) != 250 {
		t.Fatalf("recipient = %v", first)
	}
}

func TestIntegrationRejectsUnauthorizedSender(t *testing.T) {
	h := setup(t, mailpitAddr(t))
	h.configuration(t, "example.com", true)

	err := h.send(t, "mallory@stranger.test", []string{"bob@recipient.test"}, message("mallory@stranger.test", "bob@recipient.test", "nope"))
	if err == nil {
		t.Fatal("an unauthorized sender must be rejected")
	}

	if !strings.Contains(err.Error(), "550") {
		t.Fatalf("error = %v, want a 550 rejection", err)
	}

	if total := h.auditCount(t); total != 0 {
		t.Fatalf("audit records = %d, want none", total)
	}
}

func TestIntegrationDKIMFailurePreventsRelay(t *testing.T) {
	h := setup(t, mailpitAddr(t))
	h.configuration(t, "example.com", false)

	from := "alice@example.com"
	to := "bob@recipient.test"

	if err := h.send(t, from, []string{to}, message(from, to, "unsigned")); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	record := h.audit(t, delivery.StatusFailed)

	failure, ok := record["failure"].(bson.M)
	if !ok {
		t.Fatalf("failure = %v", record["failure"])
	}

	if failure["type"] != delivery.FailureProcessing && failure["type"] != delivery.FailureDKIM {
		t.Fatalf("failure type = %v", failure["type"])
	}

	attempts, ok := record["attempts"].(bson.A)
	if ok && len(attempts) != 0 {
		t.Fatalf("a signing failure must not produce relay attempts: %v", attempts)
	}
}

func TestIntegrationRelayFailureRecordsAttempts(t *testing.T) {
	h := setup(t, "127.0.0.1:1")
	h.configuration(t, "example.com", true)

	from := "alice@example.com"
	to := "bob@recipient.test"

	if err := h.send(t, from, []string{to}, message(from, to, "unreachable")); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	record := h.audit(t, delivery.StatusFailed)

	failure, ok := record["failure"].(bson.M)
	if !ok || failure["type"] != delivery.FailureRelay {
		t.Fatalf("failure = %v", record["failure"])
	}

	attempts, ok := record["attempts"].(bson.A)
	if !ok || len(attempts) != 2 {
		t.Fatalf("expected two attempts on one record, got %v", record["attempts"])
	}
}

func TestIntegrationSameMessageIDProducesTwoAudits(t *testing.T) {
	h := setup(t, mailpitAddr(t))
	h.configuration(t, "example.com", true)

	from := "alice@example.com"
	to := "bob@recipient.test"
	body := "From: " + from + "\r\nTo: " + to + "\r\nSubject: duplicate\r\nMessage-ID: <fixed@sender.test>\r\n\r\nhello\r\n"

	for range 2 {
		if err := h.send(t, from, []string{to}, body); err != nil {
			t.Fatalf("send failed: %v", err)
		}
	}

	h.dispatcher.Wait()

	if total := h.auditCount(t); total != 2 {
		t.Fatalf("audit records = %d, want 2 (no deduplication by Message-ID)", total)
	}
}
