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
	incidentrepo "dpdp-backend/internal/audit/repositories/emailincident"
	auditsvc "dpdp-backend/internal/audit/services/deliveryaudit"
	incidentsvc "dpdp-backend/internal/audit/services/emailincident"
	auditutils "dpdp-backend/internal/audit/utils"
	"dpdp-backend/internal/config"
	"dpdp-backend/internal/db"
	"dpdp-backend/internal/delivery"
	"dpdp-backend/internal/delivery/engine"
	"dpdp-backend/internal/delivery/engine/actiontrigger"
	"dpdp-backend/internal/delivery/engine/contentengine"
	engineeval "dpdp-backend/internal/delivery/engine/evaluation"
	"dpdp-backend/internal/delivery/engine/incidentgenerator"
	"dpdp-backend/internal/delivery/engine/parser"
	policysetrepo "dpdp-backend/internal/delivery/engine/policy/repositories/policyset"
	"dpdp-backend/internal/delivery/engine/policy/services/aggregator"
	policycache "dpdp-backend/internal/delivery/engine/policy/services/cache"
	"dpdp-backend/internal/delivery/engine/restrictionevalutor"
	"dpdp-backend/internal/delivery/engine/rulematcher"
	"dpdp-backend/internal/delivery/receiver"
	"dpdp-backend/internal/delivery/relay"
	deliveryutils "dpdp-backend/internal/delivery/utils"
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
		testsupport.ResetIncidents(t, client)
	})

	testsupport.ResetIncidents(t, client)

	recorder := auditsvc.NewDeliveryAuditService(auditrepo.NewDeliveryAuditRepository(client, testsupport.MongoDatabase()))
	if err := recorder.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("index creation failed: %v", err)
	}

	repository := providerrepo.NewEmailProviderRepository(database)
	registry := providerrepo.NewRedisRepository(rdb)

	configurations := delivery.NewConfigurationStore(repository)
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

	incidents := incidentsvc.NewEmailIncidentService(incidentrepo.NewEmailIncidentRepository(client, testsupport.MongoDatabase()))
	if err := incidents.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("incident index creation failed: %v", err)
	}

	enforcer := engineeval.NewEvaluationService(engineeval.Components{
		Parser:     parser.NewMessageParser(),
		Cache:      policycache.NewPolicyCacheService(policysetrepo.NewPolicySetRepository(database), rulematcher.NewCompiler(aggregator.NewAggregator(), deliveryutils.MaxRules), rdb, time.Second),
		Domain:     restrictionevalutor.NewDomainEvaluator(),
		Attachment: restrictionevalutor.NewAttachmentEvaluator(),
		Content:    contentengine.NewContentEngine(rulematcher.DefaultMatcherFactory()),
		Resolver:   actiontrigger.NewActionResolver(),
		Actions:    actiontrigger.DefaultActionFactory(nil),
		Incidents:  incidentgenerator.NewIncidentGenerator(incidents),
	})

	dispatcher := delivery.NewDispatcher(
		ctx,
		engine.NewEngine(engine.NewConfigCache(configurations, rdb), enforcer),
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

func asDocument(value any) (bson.M, bool) {
	switch typed := value.(type) {
	case bson.M:
		return typed, true
	case bson.D:
		fields := make(bson.M, len(typed))
		for _, element := range typed {
			fields[element.Key] = element.Value
		}

		return fields, true
	}

	return nil, false
}

func document(t *testing.T, value any, label string) bson.M {
	t.Helper()

	fields, ok := asDocument(value)
	if !ok {
		t.Fatalf("%s is not a document: %#v", label, value)
	}

	return fields
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

	record := h.audit(t, deliveryutils.StatusSuccess)

	if record["from"] != from {
		t.Fatalf("from = %v", record["from"])
	}
	if record["correlation_id"] == "" {
		t.Fatal("correlation id is empty")
	}
	if !strings.Contains(record["message_id"].(string), "@sender.test") {
		t.Fatalf("message id = %v", record["message_id"])
	}

	dkim, ok := asDocument(record["dkim"])
	if !ok || dkim["selector"] != deliveryutils.DefaultDKIMSelector || dkim["signed"] != true {
		t.Fatalf("dkim = %v", record["dkim"])
	}

	recipients, ok := record["recipients"].(bson.A)
	if !ok || len(recipients) != 1 {
		t.Fatalf("recipients = %v", record["recipients"])
	}

	first := document(t, recipients[0], "recipient")
	if first["status"] != deliveryutils.StatusSuccess || first["smtp_code"].(int32) != 250 {
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

	record := h.audit(t, deliveryutils.StatusFailed)

	failure, ok := asDocument(record["failure"])
	if !ok {
		t.Fatalf("failure = %v", record["failure"])
	}

	if failure["type"] != deliveryutils.FailureProcessing && failure["type"] != deliveryutils.FailureDKIM {
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

	record := h.audit(t, deliveryutils.StatusFailed)

	failure, ok := asDocument(record["failure"])
	if !ok || failure["type"] != deliveryutils.FailureRelay {
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

func (h harness) incident(t *testing.T, wantTrigger string) bson.M {
	t.Helper()

	collection := h.mongo.Database(testsupport.MongoDatabase()).Collection(auditutils.EmailIncidentCollection)

	for range 100 {
		var record bson.M

		err := collection.FindOne(context.Background(), bson.D{}).Decode(&record)
		if err == nil {
			if wantTrigger == "" || record["trigger"] == wantTrigger {
				return record
			}
		} else if !errors.Is(err, mongodriver.ErrNoDocuments) {
			t.Fatalf("incident lookup failed: %v", err)
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("no incident reached trigger %q", wantTrigger)

	return nil
}

func (h harness) incidentCount(t *testing.T) int64 {
	t.Helper()

	total, err := h.mongo.Database(testsupport.MongoDatabase()).
		Collection(auditutils.EmailIncidentCollection).
		CountDocuments(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("incident count failed: %v", err)
	}

	return total
}

func (h harness) policy(t *testing.T, sender, name, action string, domain db.Restriction, ruleIDs ...int) {
	t.Helper()

	user := testsupport.EmailUser(t, h.database, h.customer.ID, sender)
	group := testsupport.Group(t, h.database, h.customer.ID, name+" Group")
	row := testsupport.Policy(t, h.database, h.customer.ID, name, action, domain)

	testsupport.AddMember(t, h.database, h.customer.ID, group.ID, user.ID)
	testsupport.BindPolicy(t, h.database, h.customer.ID, row.ID, group.ID, ruleIDs...)
}

func TestIntegrationWithholdsOnlyTheRestrictedRecipient(t *testing.T) {
	h := setup(t, mailpitAddr(t))
	h.configuration(t, "example.com", true)

	from := "alice@example.com"
	survivor := "keep@allowed.test"
	blocked := "drop@blocked.test"

	h.policy(t, from, "No Competitors", db.ActionBlock,
		db.Restriction{Mode: db.RestrictionBlock, Values: []string{"blocked.test"}})

	body := message(from, survivor, "partially restricted")

	if err := h.send(t, from, []string{survivor, blocked}, body); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	record := h.audit(t, deliveryutils.StatusSuccess)

	recipients, ok := record["recipients"].(bson.A)
	if !ok || len(recipients) != 2 {
		t.Fatalf("recipients = %v, both must remain in the audit", record["recipients"])
	}

	statuses := map[string]string{}
	for _, entry := range recipients {
		row := document(t, entry, "recipient")
		statuses[row["email"].(string)] = row["status"].(string)
	}

	if statuses[survivor] != deliveryutils.StatusSuccess {
		t.Fatalf("%s = %q, want SUCCESS", survivor, statuses[survivor])
	}
	if statuses[blocked] != deliveryutils.StatusBlocked {
		t.Fatalf("%s = %q, want BLOCKED", blocked, statuses[blocked])
	}

	incident := h.incident(t, "RESTRICTION")

	if incident["effective_action"] != db.ActionBlock {
		t.Fatalf("effective action = %v", incident["effective_action"])
	}

	withheld, ok := incident["withheld_recipients"].(bson.A)
	if !ok || len(withheld) != 1 {
		t.Fatalf("withheld = %v, want exactly one", incident["withheld_recipients"])
	}
	if document(t, withheld[0], "withheld recipient")["email"] != blocked {
		t.Fatalf("withheld = %v", withheld[0])
	}

	matches, ok := incident["matches"].(bson.A)
	if ok && len(matches) != 0 {
		t.Fatalf("matches = %v, a restriction violation must skip content rules", matches)
	}
}

func TestIntegrationAllRecipientsRestrictedStopsDelivery(t *testing.T) {
	h := setup(t, mailpitAddr(t))
	h.configuration(t, "example.com", true)

	from := "alice@example.com"
	to := "drop@blocked.test"

	h.policy(t, from, "No Competitors", db.ActionBlock,
		db.Restriction{Mode: db.RestrictionBlock, Values: []string{"blocked.test"}})

	if err := h.send(t, from, []string{to}, message(from, to, "fully restricted")); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	record := h.audit(t, deliveryutils.StatusFailed)

	failure, ok := asDocument(record["failure"])
	if !ok || failure["type"] != auditutils.FailureRule {
		t.Fatalf("failure = %v, want RULE", record["failure"])
	}

	if attempts, ok := record["attempts"].(bson.A); ok && len(attempts) != 0 {
		t.Fatalf("attempts = %v, nothing should have been relayed", attempts)
	}

	h.incident(t, "RESTRICTION")
}

func TestIntegrationKeywordMatchRecordsIncidentAndStillDelivers(t *testing.T) {
	h := setup(t, mailpitAddr(t))
	h.configuration(t, "example.com", true)

	from := "alice@example.com"
	to := "bob@recipient.test"

	rule := testsupport.Rule(t, h.database, h.customer.ID, "Secrets", db.RuleTypeKeyword, "confidential")
	h.policy(t, from, "Data Loss", db.ActionQuarantine,
		db.Restriction{Mode: db.RestrictionNone, Values: []string{}}, rule.ID)

	body := "From: " + from + "\r\nTo: " + to + "\r\nSubject: notice\r\n" +
		"Message-ID: <content@sender.test>\r\n\r\nthis is CONFIDENTIAL material\r\n"

	if err := h.send(t, from, []string{to}, body); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	record := h.audit(t, deliveryutils.StatusSuccess)
	if record["status"] != deliveryutils.StatusSuccess {
		t.Fatalf("status = %v, content matches do not withhold this phase", record["status"])
	}

	incident := h.incident(t, "CONTENT")

	if incident["effective_action"] != db.ActionQuarantine {
		t.Fatalf("effective action = %v, want QUARANTINE", incident["effective_action"])
	}
	if incident["action_status"] != auditutils.ActionInvoked {
		t.Fatalf("action status = %v, want INVOKED", incident["action_status"])
	}

	matches, ok := incident["matches"].(bson.A)
	if !ok || len(matches) != 1 {
		t.Fatalf("matches = %v", incident["matches"])
	}

	match := document(t, matches[0], "match")

	if match["rule_type"] != db.RuleTypeKeyword || match["configured_value"] != "confidential" {
		t.Fatalf("match = %v", match)
	}
	if match["occurrences"].(int32) != 1 {
		t.Fatalf("occurrences = %v, want 1", match["occurrences"])
	}
	if _, present := match["matched_value"]; present {
		t.Fatal("the incident must never persist matched message content")
	}
}

func TestIntegrationCleanMessageProducesNoIncident(t *testing.T) {
	h := setup(t, mailpitAddr(t))
	h.configuration(t, "example.com", true)

	from := "alice@example.com"
	to := "bob@recipient.test"

	rule := testsupport.Rule(t, h.database, h.customer.ID, "Secrets", db.RuleTypeKeyword, "confidential")
	h.policy(t, from, "Data Loss", db.ActionQuarantine,
		db.Restriction{Mode: db.RestrictionNone, Values: []string{}}, rule.ID)

	if err := h.send(t, from, []string{to}, message(from, to, "nothing sensitive")); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	h.audit(t, deliveryutils.StatusSuccess)

	if total := h.incidentCount(t); total != 0 {
		t.Fatalf("incidents = %d, a passing message must produce none", total)
	}
}

func TestIntegrationRestrictionOnlyPolicyStillEnforces(t *testing.T) {
	h := setup(t, mailpitAddr(t))
	h.configuration(t, "example.com", true)

	from := "alice@example.com"
	to := "drop@blocked.test"

	h.policy(t, from, "No Competitors", db.ActionBlock,
		db.Restriction{Mode: db.RestrictionBlock, Values: []string{"blocked.test"}})

	if err := h.send(t, from, []string{to}, message(from, to, "ruleless policy")); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	h.audit(t, deliveryutils.StatusFailed)

	incident := h.incident(t, "RESTRICTION")

	if incident["effective_action"] != db.ActionBlock {
		t.Fatalf("a policy with no rules must still enforce its restrictions: %v", incident)
	}
}

func TestIntegrationUnregisteredSenderBypassesPolicies(t *testing.T) {
	h := setup(t, mailpitAddr(t))
	h.configuration(t, "example.com", true)

	from := "stranger@example.com"
	to := "drop@blocked.test"

	h.policy(t, "alice@example.com", "No Competitors", db.ActionBlock,
		db.Restriction{Mode: db.RestrictionBlock, Values: []string{"blocked.test"}})

	if err := h.send(t, from, []string{to}, message(from, to, "no policies apply")); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	h.audit(t, deliveryutils.StatusSuccess)

	if total := h.incidentCount(t); total != 0 {
		t.Fatalf("incidents = %d, an unregistered sender resolves to no policies", total)
	}
}
