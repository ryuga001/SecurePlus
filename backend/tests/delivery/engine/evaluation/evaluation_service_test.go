package evaluation_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	"dpdp-backend/internal/delivery/engine/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

type stubParser struct {
	parsed dto.ParsedMessage
	err    error
}

func (p stubParser) Parse(_ []byte) (dto.ParsedMessage, error) {
	return p.parsed, p.err
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

type stubDomain struct {
	violations []dto.RestrictionViolation
}

func (d stubDomain) Evaluate(_ []string, _ dto.EffectiveRestrictions) []dto.RestrictionViolation {
	return d.violations
}

type stubAttachment struct {
	violations []dto.RestrictionViolation
}

func (a stubAttachment) Evaluate(_ []dto.Attachment, _ dto.EffectiveRestrictions) []dto.RestrictionViolation {
	return a.violations
}

type spyContent struct {
	calls   int
	matches []dto.RuleMatch
}

func (c *spyContent) Evaluate(_ dto.MatchInput, _ dto.CompiledSet) []dto.RuleMatch {
	c.calls++

	return c.matches
}

type stubResolver struct {
	action string
}

func (r stubResolver) Resolve(_ []dto.RuleMatch) string {
	return r.action
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

type stubActions struct {
	executor *spyExecutor
}

func (a stubActions) For(action string) (dto.ActionExecutor, bool) {
	if a.executor == nil || action == utils.ActionNone {
		return nil, false
	}

	return a.executor, true
}

type spyIncidents struct {
	records  []dto.EvaluationResult
	outcomes []dto.ActionResult
	err      error
}

func (i *spyIncidents) Record(_ context.Context, result dto.EvaluationResult, _ dto.MessageContext) error {
	if i.err != nil {
		return i.err
	}

	i.records = append(i.records, result)

	return nil
}

func (i *spyIncidents) UpdateAction(_ context.Context, _ string, result dto.ActionResult) error {
	i.outcomes = append(i.outcomes, result)

	return nil
}

func message() dto.MessageContext {
	return dto.MessageContext{
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

func components(set *dto.CompiledSet) (evaluation.Components, *spyContent, *spyExecutor, *spyIncidents) {
	content := &spyContent{}
	executor := &spyExecutor{action: utils.ActionBlock}
	incidents := &spyIncidents{}

	return evaluation.Components{
		Parser:     stubParser{},
		Cache:      stubCache{set: set},
		Domain:     stubDomain{},
		Attachment: stubAttachment{},
		Content:    content,
		Resolver:   stubResolver{action: utils.ActionBlock},
		Actions:    stubActions{executor: executor},
		Incidents:  incidents,
	}, content, executor, incidents
}

func compiledSet(policies int) *dto.CompiledSet {
	return &dto.CompiledSet{CustomerID: 1, EmailUserID: 42, PolicyCount: policies}
}

func domainViolation(recipient, domain string) dto.RestrictionViolation {
	return dto.RestrictionViolation{
		Kind:       utils.KindDomain,
		Mode:       utils.RestrictionBlock,
		Value:      domain,
		Recipient:  recipient,
		PolicyID:   7,
		PolicyName: "No Competitors",
	}
}

func TestSenderWithNoPoliciesPasses(t *testing.T) {
	parts, _, executor, incidents := components(compiledSet(0))
	service := evaluation.NewEvaluationService(parts)

	outcome, err := service.Enforce(context.Background(), message())
	if err != nil {
		t.Fatalf("Enforce returned %v", err)
	}

	if outcome.Result.Decision != utils.DecisionPass || outcome.Result.EffectiveAction != utils.ActionNone {
		t.Fatalf("result = %+v", outcome.Result)
	}
	if len(incidents.records) != 0 {
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
	parts, content, executor, incidents := components(compiledSet(2))
	service := evaluation.NewEvaluationService(parts)

	outcome, err := service.Enforce(context.Background(), message())
	if err != nil {
		t.Fatalf("Enforce returned %v", err)
	}

	if content.calls != 1 {
		t.Fatalf("content engine called %d times, want 1", content.calls)
	}
	if outcome.Result.Decision != utils.DecisionPass {
		t.Fatalf("decision = %q", outcome.Result.Decision)
	}
	if len(incidents.records) != 0 || executor.calls != 0 {
		t.Fatal("PASS must produce neither an incident nor an action")
	}
}

func TestRestrictionViolationSkipsContentRules(t *testing.T) {
	parts, content, executor, incidents := components(compiledSet(1))
	parts.Domain = stubDomain{violations: []dto.RestrictionViolation{domainViolation("b@blocked.test", "blocked.test")}}
	service := evaluation.NewEvaluationService(parts)

	outcome, err := service.Enforce(context.Background(), message())
	if err != nil {
		t.Fatalf("Enforce returned %v", err)
	}

	if content.calls != 0 {
		t.Fatal("content rules must not run after a definitive restriction violation")
	}
	if outcome.Result.Trigger != utils.TriggerRestriction {
		t.Fatalf("trigger = %q", outcome.Result.Trigger)
	}
	if outcome.Result.EffectiveAction != utils.ActionBlock {
		t.Fatalf("action = %q, a restriction violation is always BLOCK", outcome.Result.EffectiveAction)
	}
	if len(incidents.records) != 1 || executor.calls != 1 {
		t.Fatalf("incidents=%d executor=%d, want one of each", len(incidents.records), executor.calls)
	}
}

func TestOnlyOffendingRecipientIsWithheld(t *testing.T) {
	parts, _, _, incidents := components(compiledSet(1))
	parts.Domain = stubDomain{violations: []dto.RestrictionViolation{domainViolation("b@blocked.test", "blocked.test")}}
	service := evaluation.NewEvaluationService(parts)

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
	if len(incidents.records[0].Withheld) != 1 {
		t.Fatalf("incident withheld = %+v", incidents.records[0].Withheld)
	}
}

func TestEveryRecipientWithheldStopsDelivery(t *testing.T) {
	parts, _, _, incidents := components(compiledSet(1))
	parts.Domain = stubDomain{violations: []dto.RestrictionViolation{
		domainViolation("a@ok.test", "ok.test"),
		domainViolation("b@blocked.test", "blocked.test"),
		domainViolation("c@ok.test", "ok.test"),
	}}
	service := evaluation.NewEvaluationService(parts)

	outcome, err := service.Enforce(context.Background(), message())

	if !errors.Is(err, utils.ErrAllRecipientsRestricted) {
		t.Fatalf("error = %v, want ErrAllRecipientsRestricted", err)
	}
	if len(outcome.Withheld) != 3 {
		t.Fatalf("withheld = %+v, want all three", outcome.Withheld)
	}
	if len(incidents.records) != 1 {
		t.Fatal("an incident must still be recorded when nothing is delivered")
	}
}

func TestAttachmentViolationIsMessageLevel(t *testing.T) {
	parts, content, _, _ := components(compiledSet(1))
	parts.Attachment = stubAttachment{violations: []dto.RestrictionViolation{{
		Kind:       utils.KindAttachment,
		Mode:       utils.RestrictionBlock,
		Value:      "exe",
		Filename:   "payload.exe",
		PolicyID:   3,
		PolicyName: "No Executables",
	}}}
	service := evaluation.NewEvaluationService(parts)

	outcome, err := service.Enforce(context.Background(), message())
	if err != nil {
		t.Fatalf("Enforce returned %v", err)
	}

	if content.calls != 0 {
		t.Fatal("content rules must not run after an attachment violation")
	}
	if len(outcome.Withheld) != 0 {
		t.Fatalf("withheld = %+v, an attachment violation withholds nobody this phase", outcome.Withheld)
	}
	if len(outcome.Delivered) != 3 {
		t.Fatalf("delivered = %v, want every recipient", outcome.Delivered)
	}
}

func TestContentMatchRecordsIncidentAndInvokesAction(t *testing.T) {
	parts, content, executor, incidents := components(compiledSet(2))
	content.matches = []dto.RuleMatch{{
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
	service := evaluation.NewEvaluationService(parts)

	outcome, err := service.Enforce(context.Background(), message())
	if err != nil {
		t.Fatalf("Enforce returned %v", err)
	}

	if outcome.Result.Trigger != utils.TriggerContent {
		t.Fatalf("trigger = %q", outcome.Result.Trigger)
	}
	if outcome.Result.EffectiveAction != utils.ActionBlock {
		t.Fatalf("action = %q", outcome.Result.EffectiveAction)
	}
	if len(outcome.Delivered) != 3 {
		t.Fatalf("delivered = %v, content matches withhold nobody this phase", outcome.Delivered)
	}
	if len(incidents.records) != 1 || len(incidents.records[0].Matches) != 1 {
		t.Fatalf("incident = %+v", incidents.records)
	}
	if incidents.records[0].EvaluatedPolicyCount != 2 {
		t.Fatalf("evaluated policies = %d, want 2", incidents.records[0].EvaluatedPolicyCount)
	}
	if len(incidents.records[0].TriggeredPolicyIDs) != 1 || incidents.records[0].TriggeredPolicyIDs[0] != 1 {
		t.Fatalf("triggered policies = %v", incidents.records[0].TriggeredPolicyIDs)
	}
	if executor.calls != 1 {
		t.Fatalf("executor calls = %d, want 1", executor.calls)
	}
	if len(incidents.outcomes) != 1 || incidents.outcomes[0].Status != utils.ActionInvoked {
		t.Fatalf("action outcomes = %+v", incidents.outcomes)
	}
}

func TestExecutorFailureDoesNotDowngradeTheDecision(t *testing.T) {
	parts, content, executor, incidents := components(compiledSet(1))
	content.matches = []dto.RuleMatch{{PolicyID: 1, RuleID: 1, Action: utils.ActionBlock, Occurrences: 1}}
	executor.err = errors.New("quarantine store unreachable")
	service := evaluation.NewEvaluationService(parts)

	outcome, err := service.Enforce(context.Background(), message())
	if err != nil {
		t.Fatalf("Enforce returned %v", err)
	}

	if outcome.Result.EffectiveAction != utils.ActionBlock {
		t.Fatalf("action = %q, an execution failure must not change the decision", outcome.Result.EffectiveAction)
	}
	if len(incidents.outcomes) != 1 {
		t.Fatalf("action outcomes = %+v", incidents.outcomes)
	}
	if incidents.outcomes[0].Status != utils.ActionFailed {
		t.Fatalf("status = %q, want FAILED", incidents.outcomes[0].Status)
	}
	if !strings.Contains(incidents.outcomes[0].Error, "unreachable") {
		t.Fatalf("error = %q, want the executor failure recorded", incidents.outcomes[0].Error)
	}
}

func TestIncidentWriteFailureSkipsTheExecutor(t *testing.T) {
	parts, content, executor, incidents := components(compiledSet(1))
	content.matches = []dto.RuleMatch{{PolicyID: 1, RuleID: 1, Action: utils.ActionBlock, Occurrences: 1}}
	incidents.err = errors.New("mongo down")
	service := evaluation.NewEvaluationService(parts)

	if _, err := service.Enforce(context.Background(), message()); err != nil {
		t.Fatalf("Enforce returned %v, fail-open must not surface the error", err)
	}

	if executor.calls != 0 {
		t.Fatal("the executor must not run when the incident could not be recorded")
	}
}

func TestPolicyResolutionFailureFailsOpen(t *testing.T) {
	parts, _, _, _ := components(compiledSet(1))
	parts.Cache = stubCache{err: errors.New("postgres down")}
	service := evaluation.NewEvaluationService(parts)

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
	service := evaluation.NewEvaluationService(parts)

	if _, err := service.Enforce(context.Background(), message()); err == nil {
		t.Fatal("fail-closed must surface the resolution failure")
	}
}

func TestCancellationIsNeverSwallowed(t *testing.T) {
	parts, _, _, _ := components(compiledSet(1))
	parts.Cache = stubCache{err: context.Canceled}
	service := evaluation.NewEvaluationService(parts)

	if _, err := service.Enforce(context.Background(), message()); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, shutdown must not be reported as a pass", err)
	}
}

func TestUnparseableMessageFailsOpenOnHeadersOnly(t *testing.T) {
	parts, content, _, _ := components(compiledSet(1))
	parts.Parser = stubParser{err: utils.ErrMessageUnreadable}
	service := evaluation.NewEvaluationService(parts)

	if _, err := service.Enforce(context.Background(), message()); err != nil {
		t.Fatalf("Enforce returned %v, want fail-open", err)
	}

	if content.calls != 1 {
		t.Fatalf("content engine called %d times, evaluation should continue", content.calls)
	}
}

func TestUnparseableMessageCanFailClosed(t *testing.T) {
	parts, _, _, _ := components(compiledSet(1))
	parts.Parser = stubParser{err: utils.ErrMessageUnreadable}
	parts.FailsClosed = true
	service := evaluation.NewEvaluationService(parts)

	if _, err := service.Enforce(context.Background(), message()); !errors.Is(err, utils.ErrMessageUnreadable) {
		t.Fatalf("error = %v, want the parse failure", err)
	}
}

func TestWithholdMapsToBlockedRecipientResults(t *testing.T) {
	results := evaluation.Withhold([]dto.WithheldRecipient{{
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
