package screening_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	incidentdto "dpdp-backend/internal/audit/dto/emailincident"
	"dpdp-backend/internal/delivery"
	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/services/adjudication"
	"dpdp-backend/internal/delivery/services/inspection"
	"dpdp-backend/internal/delivery/services/recording"
	"dpdp-backend/internal/delivery/services/screening"
	"dpdp-backend/internal/delivery/utils"
)

func failingParser(err error) func([]byte) (dto.ParsedMessage, error) {
	return func([]byte) (dto.ParsedMessage, error) {
		return dto.ParsedMessage{
			Subject: "s",
			Parts:   []dto.ContentPart{{Location: utils.LocationSubject, Text: "s"}},
		}, err
	}
}

func parserWithAttachments(attachments ...dto.Attachment) func([]byte) (dto.ParsedMessage, error) {
	return func([]byte) (dto.ParsedMessage, error) {
		return dto.ParsedMessage{Attachments: attachments}, nil
	}
}

type stubCache struct {
	set *dto.CompiledSet
	err error
}

func (c stubCache) Load(_ context.Context, _ int, _ string) (*dto.CompiledSet, error) {
	if c.err != nil {
		return nil, c.err
	}

	return c.set, nil
}

type spyMatcher struct {
	calls   int
	matches []dto.RuleMatch
}

func (m *spyMatcher) Type() string { return utils.MatcherKeyword }

func (m *spyMatcher) Match(_ dto.MatchInput, _ dto.CompiledRules) []dto.RuleMatch {
	m.calls++

	return m.matches
}

type spyExecutor struct {
	action string
	calls  int
	err    error
}

func (e *spyExecutor) Action() string { return e.action }

func (e *spyExecutor) Execute(_ context.Context, _ dto.ActionRequest) (dto.ActionResult, error) {
	e.calls++

	if e.err != nil {
		return dto.ActionResult{}, e.err
	}

	return dto.ActionResult{Action: e.action, Status: utils.ActionInvoked}, nil
}

type spyWriter struct {
	records  []incidentdto.Record
	outcomes []incidentdto.ActionOutcome
	err      error
}

func (w *spyWriter) Record(_ context.Context, record incidentdto.Record) error {
	if w.err != nil {
		return w.err
	}

	w.records = append(w.records, record)

	return nil
}

func (w *spyWriter) UpdateAction(_ context.Context, _ string, outcome incidentdto.ActionOutcome) error {
	w.outcomes = append(w.outcomes, outcome)

	return nil
}

func message() delivery.EmailMessage {
	return delivery.EmailMessage{
		CorrelationID: "corr-1",
		MessageID:     "<m@sender.test>",
		CustomerID:    1,
		ConfigID:      2,
		From:          "alice@example.com",
		SenderDomain:  "example.com",
		Recipients:    []string{"a@ok.test", "b@blocked.test", "c@ok.test"},
		Raw:           []byte("From: alice@example.com\r\nSubject: s\r\n\r\nbody\r\n"),
	}
}

func components(set *dto.CompiledSet) (screening.Options, *spyMatcher, *spyExecutor, *spyWriter) {
	matcher := &spyMatcher{}
	executor := &spyExecutor{action: utils.ActionBlock}
	writer := &spyWriter{}

	return screening.Options{
		Cache:     stubCache{set: set},
		Content:   inspection.NewContentEngine(inspection.NewMatcherFactory(matcher)),
		Actions:   adjudication.NewActionFactory(executor),
		Incidents: recording.NewIncidentGenerator(writer),
	}, matcher, executor, writer
}

func compiledSet(policies int) *dto.CompiledSet {
	return &dto.CompiledSet{
		CustomerID:  1,
		EmailUserID: 42,
		PolicyCount: policies,
		RuleTypes:   []string{utils.MatcherKeyword},
	}
}

func blockDomains(set *dto.CompiledSet, domains ...string) *dto.CompiledSet {
	blocked := make(map[string]dto.PolicyRef, len(domains))

	for _, domain := range domains {
		blocked[domain] = dto.PolicyRef{PolicyID: 7, PolicyName: "No Competitors"}
	}

	set.Restrictions.Domain.Blocked = blocked

	return set
}

func blockExtensions(set *dto.CompiledSet, extensions ...string) *dto.CompiledSet {
	blocked := make(map[string]dto.PolicyRef, len(extensions))

	for _, extension := range extensions {
		blocked[extension] = dto.PolicyRef{PolicyID: 3, PolicyName: "No Executables"}
	}

	set.Restrictions.Attachment.Blocked = blocked

	return set
}

func TestSenderWithNoPoliciesPasses(t *testing.T) {
	parts, _, executor, writer := components(compiledSet(0))
	service := screening.New(parts)

	outcome, err := service.Enforce(context.Background(), message())
	if err != nil {
		t.Fatalf("Enforce returned %v", err)
	}

	if outcome.Result.Decision != utils.DecisionPass || outcome.Result.EffectiveAction != utils.ActionNone {
		t.Fatalf("result = %+v", outcome.Result)
	}
	if len(writer.records) != 0 {
		t.Fatal("a passing message must not produce an incident")
	}
	if executor.calls != 0 {
		t.Fatal("a passing message must not invoke an executor")
	}
	if len(outcome.Delivered) != 3 {
		t.Fatalf("delivered = %v, want every recipient", outcome.Delivered)
	}
}

func TestNoMatchesPassesWithoutIncident(t *testing.T) {
	parts, matcher, executor, writer := components(compiledSet(2))
	service := screening.New(parts)

	outcome, err := service.Enforce(context.Background(), message())
	if err != nil {
		t.Fatalf("Enforce returned %v", err)
	}

	if matcher.calls != 1 {
		t.Fatalf("content engine called %d times, want 1", matcher.calls)
	}
	if outcome.Result.Decision != utils.DecisionPass {
		t.Fatalf("decision = %q", outcome.Result.Decision)
	}
	if len(writer.records) != 0 || executor.calls != 0 {
		t.Fatal("PASS must produce neither an incident nor an action")
	}
}

func TestRestrictionViolationSkipsContentRules(t *testing.T) {
	parts, matcher, executor, writer := components(blockDomains(compiledSet(1), "blocked.test"))
	service := screening.New(parts)

	outcome, err := service.Enforce(context.Background(), message())
	if err != nil {
		t.Fatalf("Enforce returned %v", err)
	}

	if matcher.calls != 0 {
		t.Fatal("content rules must not run after a definitive restriction violation")
	}
	if outcome.Result.Trigger != utils.TriggerRestriction {
		t.Fatalf("trigger = %q", outcome.Result.Trigger)
	}
	if outcome.Result.EffectiveAction != utils.ActionBlock {
		t.Fatalf("action = %q, a restriction violation is always BLOCK", outcome.Result.EffectiveAction)
	}
	if len(writer.records) != 1 || executor.calls != 1 {
		t.Fatalf("incidents=%d executor=%d, want one of each", len(writer.records), executor.calls)
	}
}

func TestOnlyOffendingRecipientIsWithheld(t *testing.T) {
	parts, _, _, writer := components(blockDomains(compiledSet(1), "blocked.test"))
	service := screening.New(parts)

	outcome, err := service.Enforce(context.Background(), message())
	if err != nil {
		t.Fatalf("Enforce returned %v", err)
	}

	if len(outcome.Delivered) != 2 {
		t.Fatalf("delivered = %v, want the two unaffected recipients", outcome.Delivered)
	}
	for _, recipient := range outcome.Delivered {
		if recipient == "b@blocked.test" {
			t.Fatal("a withheld recipient must not remain in the envelope")
		}
	}
	if len(outcome.Withheld) != 1 || outcome.Withheld[0].Email != "b@blocked.test" {
		t.Fatalf("withheld = %+v", outcome.Withheld)
	}
	if len(writer.records[0].WithheldRecipients) != 1 {
		t.Fatalf("incident withheld = %+v", writer.records[0].WithheldRecipients)
	}
}

func TestEveryRecipientWithheldStopsDelivery(t *testing.T) {
	parts, _, _, writer := components(blockDomains(compiledSet(1), "ok.test", "blocked.test"))
	service := screening.New(parts)

	outcome, err := service.Enforce(context.Background(), message())

	if !errors.Is(err, utils.ErrAllRecipientsRestricted) {
		t.Fatalf("error = %v, want ErrAllRecipientsRestricted", err)
	}
	if len(outcome.Withheld) != 3 {
		t.Fatalf("withheld = %+v, want all three", outcome.Withheld)
	}
	if len(writer.records) != 1 {
		t.Fatal("an incident must still be recorded when nothing is delivered")
	}
}

func TestAttachmentViolationIsMessageLevel(t *testing.T) {
	parts, matcher, _, _ := components(blockExtensions(compiledSet(1), "exe"))
	parts.Parser = parserWithAttachments(dto.Attachment{Filename: "payload.exe", Extension: "exe"})
	service := screening.New(parts)

	outcome, err := service.Enforce(context.Background(), message())
	if err != nil {
		t.Fatalf("Enforce returned %v", err)
	}

	if matcher.calls != 0 {
		t.Fatal("content rules must not run after an attachment violation")
	}
	if len(outcome.Withheld) != 0 {
		t.Fatalf("withheld = %+v, an attachment violation withholds nobody this phase", outcome.Withheld)
	}
	if len(outcome.Delivered) != 3 {
		t.Fatalf("delivered = %v, want every recipient", outcome.Delivered)
	}
}

func TestContentMatchBelowBlockStillDelivers(t *testing.T) {
	parts, matcher, _, writer := components(compiledSet(1))
	executor := &spyExecutor{action: utils.ActionQuarantine}
	parts.Actions = adjudication.NewActionFactory(executor)
	matcher.matches = []dto.RuleMatch{{
		PolicyID:   1,
		PolicyName: "Data Loss",
		Action:     utils.ActionQuarantine,
		RuleID:     9,
		RuleType:   utils.MatcherKeyword,
	}}
	service := screening.New(parts)

	outcome, err := service.Enforce(context.Background(), message())
	if err != nil {
		t.Fatalf("Enforce returned %v, only BLOCK stops delivery", err)
	}

	if outcome.Result.EffectiveAction != utils.ActionQuarantine {
		t.Fatalf("action = %q", outcome.Result.EffectiveAction)
	}
	if len(outcome.Delivered) != 3 {
		t.Fatalf("delivered = %v, want every recipient", outcome.Delivered)
	}
	if len(outcome.Withheld) != 0 {
		t.Fatalf("withheld = %+v, QUARANTINE withholds nobody", outcome.Withheld)
	}
	if len(writer.records) != 1 || executor.calls != 1 {
		t.Fatalf("incidents=%d executor=%d, want one of each", len(writer.records), executor.calls)
	}
}

func TestContentMatchRecordsIncidentAndInvokesAction(t *testing.T) {
	parts, matcher, executor, writer := components(compiledSet(2))
	matcher.matches = []dto.RuleMatch{{
		PolicyID:        1,
		PolicyName:      "Data Loss",
		Action:          utils.ActionBlock,
		RuleID:          9,
		RuleName:        "Cards",
		RuleType:        utils.MatcherKeyword,
		ConfiguredValue: "card",
		Occurrences:     2,
		Locations:       []string{utils.LocationBody},
	}}
	service := screening.New(parts)

	outcome, err := service.Enforce(context.Background(), message())

	if !errors.Is(err, utils.ErrAllRecipientsRestricted) {
		t.Fatalf("error = %v, a content match resolving to BLOCK must stop delivery", err)
	}

	if outcome.Result.Trigger != utils.TriggerContent {
		t.Fatalf("trigger = %q", outcome.Result.Trigger)
	}
	if outcome.Result.EffectiveAction != utils.ActionBlock {
		t.Fatalf("action = %q", outcome.Result.EffectiveAction)
	}
	if len(outcome.Delivered) != 0 {
		t.Fatalf("delivered = %v, BLOCK must deliver to nobody", outcome.Delivered)
	}
	if len(outcome.Withheld) != 3 {
		t.Fatalf("withheld = %+v, a content rule applies to the whole message", outcome.Withheld)
	}
	if outcome.Withheld[0].PolicyName != "Data Loss" {
		t.Fatalf("attribution = %+v, want the blocking policy", outcome.Withheld[0])
	}
	if len(writer.records) != 1 || len(writer.records[0].Matches) != 1 {
		t.Fatalf("incident = %+v", writer.records)
	}
	if writer.records[0].EvaluatedPolicyCount != 2 {
		t.Fatalf("evaluated policies = %d, want 2", writer.records[0].EvaluatedPolicyCount)
	}
	if len(writer.records[0].TriggeredPolicyIDs) != 1 || writer.records[0].TriggeredPolicyIDs[0] != 1 {
		t.Fatalf("triggered policies = %v", writer.records[0].TriggeredPolicyIDs)
	}
	if executor.calls != 1 {
		t.Fatalf("executor calls = %d, want 1", executor.calls)
	}
	if len(writer.outcomes) != 1 || writer.outcomes[0].Status != utils.ActionInvoked {
		t.Fatalf("action outcomes = %+v", writer.outcomes)
	}
}

func TestExecutorFailureDoesNotDowngradeTheDecision(t *testing.T) {
	parts, matcher, executor, writer := components(compiledSet(1))
	matcher.matches = []dto.RuleMatch{{PolicyID: 1, RuleID: 1, Action: utils.ActionBlock, Occurrences: 1}}
	executor.err = errors.New("quarantine store unreachable")
	service := screening.New(parts)

	outcome, err := service.Enforce(context.Background(), message())
	if !errors.Is(err, utils.ErrAllRecipientsRestricted) {
		t.Fatalf("error = %v, a BLOCK still stops delivery when its action fails", err)
	}

	if outcome.Result.EffectiveAction != utils.ActionBlock {
		t.Fatalf("action = %q, an execution failure must not change the decision", outcome.Result.EffectiveAction)
	}
	if len(writer.outcomes) != 1 {
		t.Fatalf("action outcomes = %+v", writer.outcomes)
	}
	if writer.outcomes[0].Status != utils.ActionFailed {
		t.Fatalf("status = %q, want FAILED", writer.outcomes[0].Status)
	}
	if !strings.Contains(writer.outcomes[0].Error, "unreachable") {
		t.Fatalf("error = %q, want the executor failure recorded", writer.outcomes[0].Error)
	}
}

func TestIncidentWriteFailureSkipsTheExecutor(t *testing.T) {
	parts, matcher, executor, writer := components(compiledSet(1))
	matcher.matches = []dto.RuleMatch{{PolicyID: 1, RuleID: 1, Action: utils.ActionBlock, Occurrences: 1}}
	writer.err = errors.New("mongo down")
	service := screening.New(parts)

	if _, err := service.Enforce(context.Background(), message()); !errors.Is(err, utils.ErrAllRecipientsRestricted) {
		t.Fatalf("error = %v, the incident write failure must not mask the BLOCK", err)
	}

	if executor.calls != 0 {
		t.Fatal("the executor must not run when the incident could not be recorded")
	}
}

func TestPolicyResolutionFailureFailsOpen(t *testing.T) {
	parts, _, _, _ := components(compiledSet(1))
	parts.Cache = stubCache{err: errors.New("postgres down")}
	service := screening.New(parts)

	outcome, err := service.Enforce(context.Background(), message())
	if err != nil {
		t.Fatalf("Enforce returned %v, want fail-open", err)
	}

	if outcome.Result.Decision != utils.DecisionPass {
		t.Fatalf("decision = %q", outcome.Result.Decision)
	}
	if len(outcome.Delivered) != 3 {
		t.Fatalf("delivered = %v, want every recipient", outcome.Delivered)
	}
}

func TestPolicyResolutionFailureCanFailClosed(t *testing.T) {
	parts, _, _, _ := components(compiledSet(1))
	parts.Cache = stubCache{err: errors.New("postgres down")}
	parts.FailsClosed = true
	service := screening.New(parts)

	if _, err := service.Enforce(context.Background(), message()); err == nil {
		t.Fatal("fail-closed must surface the resolution failure")
	}
}

func TestCancellationIsNeverSwallowed(t *testing.T) {
	parts, _, _, _ := components(compiledSet(1))
	parts.Cache = stubCache{err: context.Canceled}
	service := screening.New(parts)

	if _, err := service.Enforce(context.Background(), message()); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, shutdown must not be reported as a pass", err)
	}
}

func TestUnparseableMessageFailsOpenOnHeadersOnly(t *testing.T) {
	parts, matcher, _, _ := components(compiledSet(1))
	parts.Parser = failingParser(utils.ErrMessageUnreadable)
	service := screening.New(parts)

	if _, err := service.Enforce(context.Background(), message()); err != nil {
		t.Fatalf("Enforce returned %v, want fail-open", err)
	}

	if matcher.calls != 1 {
		t.Fatalf("content engine called %d times, evaluation should continue", matcher.calls)
	}
}

func TestUnparseableMessageCanFailClosed(t *testing.T) {
	parts, _, _, _ := components(compiledSet(1))
	parts.Parser = failingParser(utils.ErrMessageUnreadable)
	parts.FailsClosed = true
	service := screening.New(parts)

	if _, err := service.Enforce(context.Background(), message()); !errors.Is(err, utils.ErrMessageUnreadable) {
		t.Fatalf("error = %v, want the parse failure", err)
	}
}

func TestWithholdMapsToBlockedRecipientResults(t *testing.T) {
	results := screening.Withhold([]dto.WithheldRecipient{{
		Email:      "b@blocked.test",
		Domain:     "blocked.test",
		PolicyID:   7,
		PolicyName: "No Competitors",
	}})

	if len(results) != 1 {
		t.Fatalf("results = %+v", results)
	}
	if results[0].Status != utils.StatusBlocked {
		t.Fatalf("status = %q, want BLOCKED not FAILED", results[0].Status)
	}
	if results[0].SMTPCode != 0 {
		t.Fatalf("smtp code = %d, no conversation happened", results[0].SMTPCode)
	}
	if !strings.Contains(results[0].Error, "No Competitors") {
		t.Fatalf("error = %q, want the policy named", results[0].Error)
	}
}
