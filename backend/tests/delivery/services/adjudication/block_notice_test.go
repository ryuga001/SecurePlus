package adjudication_test

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"dpdp-backend/internal/db"
	"dpdp-backend/internal/delivery/dto/delivery"
	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/services/adjudication"
	"dpdp-backend/internal/delivery/utils"
)

type stubTemplates struct {
	template db.EmailTemplate
	orgName  string
	err      error
	orgErr   error
	name     string
}

func (s *stubTemplates) Find(_ context.Context, _ int, name string) (db.EmailTemplate, error) {
	s.name = name

	if s.err != nil {
		return db.EmailTemplate{}, s.err
	}

	return s.template, nil
}

func (s *stubTemplates) OrgName(_ context.Context, _ int) (string, error) {
	if s.orgErr != nil {
		return "", s.orgErr
	}

	return s.orgName, nil
}

type stubConfigs struct {
	cfg delivery.TenantConfig
	err error
}

func (s stubConfigs) Resolve(_ context.Context, customerID, configID int) (delivery.TenantConfig, error) {
	if s.err != nil {
		return delivery.TenantConfig{}, s.err
	}

	cfg := s.cfg
	cfg.CustomerID = customerID
	cfg.ConfigID = configID

	return cfg, nil
}

type spyRelay struct {
	calls   int
	sent    delivery.EmailMessage
	cfg     delivery.TenantConfig
	results []delivery.RecipientResult
	err     error
}

func (s *spyRelay) Deliver(
	_ context.Context,
	msg delivery.EmailMessage,
	cfg delivery.TenantConfig,
	_ func(delivery.Attempt),
) ([]delivery.RecipientResult, error) {
	s.calls++
	s.sent = msg
	s.cfg = cfg

	if s.err != nil {
		return nil, s.err
	}

	if s.results != nil {
		return s.results, nil
	}

	return []delivery.RecipientResult{{Email: msg.Recipients[0], Status: utils.StatusSuccess}}, nil
}

func templates() *stubTemplates {
	return &stubTemplates{
		orgName: "Acme Security",
		template: db.EmailTemplate{
			Subject: "Your message was not delivered - {{org_name}}",
			Body:    "<p>{{org_name}} stopped it due to {{reason}} ({{policy_name}}) to {{recipients}}. Ref {{correlation_id}} subject {{subject}}</p>",
		},
	}
}

func request() dto.ActionRequest {
	return dto.ActionRequest{
		CorrelationID: "corr-1",
		CustomerID:    4,
		ConfigID:      1,
		Action:        utils.ActionBlock,
		Trigger:       utils.TriggerContent,
		Sender:        "alice@example.com",
		SenderDomain:  "example.com",
		MessageID:     "<original@sender.test>",
		Subject:       "Quarterly numbers",
		Recipients:    []string{"bob@partner.test"},
		PolicyNames:   []string{"Data Loss"},
	}
}

func service(t *testing.T, store *stubTemplates, relay *spyRelay) *adjudication.BlockNoticeService {
	t.Helper()

	return adjudication.NewBlockNoticeService(store, stubConfigs{cfg: delivery.TenantConfig{Domain: "example.com"}}, relay)
}

func decoded(t *testing.T, encoded string) string {
	t.Helper()

	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(encoded, "\r\n", ""))
	if err != nil {
		t.Fatalf("body is not valid base64: %v", err)
	}

	return string(raw)
}

func body(t *testing.T, raw []byte) string {
	t.Helper()

	_, encoded, found := strings.Cut(string(raw), "\r\n\r\n")
	if !found {
		t.Fatalf("message has no body:\n%s", raw)
	}

	return encoded
}

func TestNoticeIsRelayedBackToTheSender(t *testing.T) {
	relay := &spyRelay{}

	if err := service(t, templates(), relay).Notify(context.Background(), request()); err != nil {
		t.Fatalf("Notify returned %v", err)
	}

	if relay.calls != 1 {
		t.Fatalf("relay calls = %d, want 1", relay.calls)
	}

	if len(relay.sent.Recipients) != 1 || relay.sent.Recipients[0] != "alice@example.com" {
		t.Fatalf("recipients = %v, the notice must go to the original sender", relay.sent.Recipients)
	}
}

func TestNoticeComesFromNoReplyAtTheSenderDomain(t *testing.T) {
	relay := &spyRelay{}

	if err := service(t, templates(), relay).Notify(context.Background(), request()); err != nil {
		t.Fatalf("Notify returned %v", err)
	}

	if relay.sent.From != "no-reply@example.com" {
		t.Fatalf("from = %q, want no-reply@example.com", relay.sent.From)
	}

	if !strings.Contains(string(relay.sent.Raw), "<no-reply@example.com>") {
		t.Fatalf("From header missing the no-reply mailbox:\n%s", relay.sent.Raw)
	}
}

func TestNoticeUsesTheTenantSigningConfig(t *testing.T) {
	relay := &spyRelay{}

	if err := service(t, templates(), relay).Notify(context.Background(), request()); err != nil {
		t.Fatalf("Notify returned %v", err)
	}

	if relay.cfg.CustomerID != 4 || relay.cfg.ConfigID != 1 {
		t.Fatalf("config = %+v, want the sender tenant's config so the notice is signed", relay.cfg)
	}
}

func TestNoticeLoadsTheBlockTemplate(t *testing.T) {
	store := templates()

	if err := service(t, store, &spyRelay{}).Notify(context.Background(), request()); err != nil {
		t.Fatalf("Notify returned %v", err)
	}

	if store.name != utils.TemplatePolicyBlockNotice {
		t.Fatalf("template = %q, want %q", store.name, utils.TemplatePolicyBlockNotice)
	}
}

func TestNoticeRendersOrgNameAndPolicyIntoTheBody(t *testing.T) {
	relay := &spyRelay{}

	if err := service(t, templates(), relay).Notify(context.Background(), request()); err != nil {
		t.Fatalf("Notify returned %v", err)
	}

	rendered := decoded(t, body(t, relay.sent.Raw))

	for _, want := range []string{"Acme Security", "Data Loss", "bob@partner.test", "corr-1", "Quarterly numbers"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("body is missing %q:\n%s", want, rendered)
		}
	}

	if strings.Contains(rendered, "{{") {
		t.Fatalf("body has unrendered placeholders:\n%s", rendered)
	}
}

func TestNoticeSubjectCarriesTheOrgName(t *testing.T) {
	relay := &spyRelay{}

	if err := service(t, templates(), relay).Notify(context.Background(), request()); err != nil {
		t.Fatalf("Notify returned %v", err)
	}

	if !strings.Contains(string(relay.sent.Raw), "Subject: Your message was not delivered - Acme Security") {
		t.Fatalf("subject not rendered:\n%s", relay.sent.Raw)
	}
}

func TestNoticeNamesTheWithheldRecipientsWhenOnlySomeWereBlocked(t *testing.T) {
	relay := &spyRelay{}

	payload := request()
	payload.Trigger = utils.TriggerRestriction
	payload.Recipients = []string{"bob@partner.test", "carol@blocked.test"}
	payload.Blocked = []string{"carol@blocked.test"}

	if err := service(t, templates(), relay).Notify(context.Background(), payload); err != nil {
		t.Fatalf("Notify returned %v", err)
	}

	rendered := decoded(t, body(t, relay.sent.Raw))

	if !strings.Contains(rendered, "carol@blocked.test") {
		t.Fatalf("body must name the withheld recipient:\n%s", rendered)
	}
	if strings.Contains(rendered, "bob@partner.test") {
		t.Fatalf("body must not claim the delivered recipient was blocked:\n%s", rendered)
	}
}

func TestNoticeCarriesAutoSubmittedHeaders(t *testing.T) {
	relay := &spyRelay{}

	if err := service(t, templates(), relay).Notify(context.Background(), request()); err != nil {
		t.Fatalf("Notify returned %v", err)
	}

	raw := string(relay.sent.Raw)

	for _, header := range []string{"Auto-Submitted: auto-replied", "X-Auto-Response-Suppress: All"} {
		if !strings.Contains(raw, header) {
			t.Fatalf("missing %q, without it mail loops are possible:\n%s", header, raw)
		}
	}
}

func TestNoticeGetsItsOwnMessageID(t *testing.T) {
	relay := &spyRelay{}

	if err := service(t, templates(), relay).Notify(context.Background(), request()); err != nil {
		t.Fatalf("Notify returned %v", err)
	}

	if relay.sent.MessageID == "<original@sender.test>" {
		t.Fatal("the notice reused the blocked message's Message-ID")
	}
	if !strings.HasSuffix(relay.sent.MessageID, "@example.com>") {
		t.Fatalf("message id = %q", relay.sent.MessageID)
	}
}

func TestNoticeIsSkippedWhenTheSenderIsTheNoticeMailbox(t *testing.T) {
	relay := &spyRelay{}

	payload := request()
	payload.Sender = "no-reply@example.com"

	if err := service(t, templates(), relay).Notify(context.Background(), payload); err != nil {
		t.Fatalf("Notify returned %v", err)
	}

	if relay.calls != 0 {
		t.Fatal("a notice to the notice mailbox would loop and must be skipped")
	}
}

func TestNoticeFailsWithoutASender(t *testing.T) {
	payload := request()
	payload.Sender = "  "

	err := service(t, templates(), &spyRelay{}).Notify(context.Background(), payload)

	if !errors.Is(err, utils.ErrNoticeSenderMissing) {
		t.Fatalf("error = %v, want ErrNoticeSenderMissing", err)
	}
}

func TestNoticeSurfacesAMissingTemplate(t *testing.T) {
	store := templates()
	store.err = utils.ErrNoticeTemplateMissing

	err := service(t, store, &spyRelay{}).Notify(context.Background(), request())

	if !errors.Is(err, utils.ErrNoticeTemplateMissing) {
		t.Fatalf("error = %v, want ErrNoticeTemplateMissing", err)
	}
}

func TestNoticeSurfacesARelayRejection(t *testing.T) {
	relay := &spyRelay{results: []delivery.RecipientResult{{
		Email:    "alice@example.com",
		Status:   utils.StatusFailed,
		SMTPCode: 550,
		Error:    "no such user",
	}}}

	err := service(t, templates(), relay).Notify(context.Background(), request())

	if !errors.Is(err, utils.ErrNoticeNotDelivered) {
		t.Fatalf("error = %v, want ErrNoticeNotDelivered", err)
	}
	if !strings.Contains(err.Error(), "550") {
		t.Fatalf("error = %v, want the smtp code carried through", err)
	}
}

func TestNoticeSurfacesASigningFailure(t *testing.T) {
	failure := errors.New("dkim key missing")

	svc := adjudication.NewBlockNoticeService(templates(), stubConfigs{err: failure}, &spyRelay{})

	if err := svc.Notify(context.Background(), request()); !errors.Is(err, failure) {
		t.Fatalf("error = %v, want the config failure", err)
	}
}
