package recording_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	auditdto "dpdp-backend/internal/audit/dto/emailincident"
	auditutils "dpdp-backend/internal/audit/utils"
	"dpdp-backend/internal/delivery/dto/delivery"
	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/services/recording"
	generatordto "dpdp-backend/internal/delivery/services/recording"
	"dpdp-backend/internal/delivery/utils"
)

const secret = "4111111111111111"

type spyWriter struct {
	records  []auditdto.Record
	outcomes []auditdto.ActionOutcome
	err      error
}

func (w *spyWriter) Record(_ context.Context, record auditdto.Record) error {
	if w.err != nil {
		return w.err
	}

	w.records = append(w.records, record)

	return nil
}

func (w *spyWriter) UpdateAction(_ context.Context, _ string, outcome auditdto.ActionOutcome) error {
	w.outcomes = append(w.outcomes, outcome)

	return nil
}

func message() delivery.EmailMessage {
	return delivery.EmailMessage{
		CorrelationID: "corr-1",
		MessageID:     "<m@sender.test>",
		CustomerID:    3,
		ConfigID:      4,
		From:          "alice@example.com",
		SenderDomain:  "example.com",
		Recipients:    []string{"a@ok.test", "b@blocked.test"},
		Raw:           []byte("the card number is " + secret),
	}
}

func result() dto.EvaluationResult {
	return dto.EvaluationResult{
		CorrelationID:        "corr-1",
		CustomerID:           3,
		EmailUserID:          42,
		Decision:             utils.DecisionFlagged,
		Trigger:              utils.TriggerContent,
		EffectiveAction:      utils.ActionQuarantine,
		EvaluatedPolicyCount: 2,
		TriggeredPolicyIDs:   []int{1},
		Recipients:           []string{"a@ok.test", "b@blocked.test"},
		Matches: []dto.RuleMatch{{
			PolicyID:        1,
			PolicyName:      "Data Loss",
			Action:          utils.ActionQuarantine,
			RuleID:          9,
			RuleName:        "Card Number",
			RuleType:        utils.MatcherKeyword,
			ConfiguredValue: "card",
			Occurrences:     3,
			Locations:       []string{utils.LocationBody},
		}},
	}
}

func TestGeneratedRecordCarriesTheEnvelopeAndDecision(t *testing.T) {
	record := recording.NewIncidentGenerator(&spyWriter{}).
		Generate(generatordto.GenerationInput{Message: message(), Result: result()})

	if record.CorrelationID != "corr-1" || record.MessageID != "<m@sender.test>" {
		t.Fatalf("record = %+v", record)
	}
	if record.CustomerID != 3 || record.ConfigID != 4 || record.EmailUserID != 42 {
		t.Fatalf("record = %+v", record)
	}
	if record.Decision != utils.DecisionFlagged || record.Trigger != utils.TriggerContent {
		t.Fatalf("record = %+v", record)
	}
	if record.EffectiveAction != utils.ActionQuarantine {
		t.Fatalf("action = %q", record.EffectiveAction)
	}
	if record.EvaluatedPolicyCount != 2 || len(record.TriggeredPolicyIDs) != 1 {
		t.Fatalf("record = %+v", record)
	}
}

func TestRecipientDomainsAreDerived(t *testing.T) {
	record := recording.NewIncidentGenerator(&spyWriter{}).
		Generate(generatordto.GenerationInput{Message: message(), Result: result()})

	if len(record.Recipients) != 2 {
		t.Fatalf("recipients = %+v", record.Recipients)
	}
	if record.Recipients[0].Domain != "ok.test" || record.Recipients[1].Domain != "blocked.test" {
		t.Fatalf("recipients = %+v", record.Recipients)
	}
}

func TestActionStartsPending(t *testing.T) {
	record := recording.NewIncidentGenerator(&spyWriter{}).
		Generate(generatordto.GenerationInput{Message: message(), Result: result()})

	if record.ActionStatus != auditutils.ActionPending {
		t.Fatalf("action status = %q, want PENDING before the executor runs", record.ActionStatus)
	}
	if record.ActionInvoked != utils.ActionQuarantine {
		t.Fatalf("action invoked = %q", record.ActionInvoked)
	}
}

func TestMatchEvidenceCarriesCountsNotContent(t *testing.T) {
	record := recording.NewIncidentGenerator(&spyWriter{}).
		Generate(generatordto.GenerationInput{Message: message(), Result: result()})

	if len(record.Matches) != 1 {
		t.Fatalf("matches = %+v", record.Matches)
	}

	match := record.Matches[0]

	if match.RuleName != "Card Number" || match.RuleType != utils.MatcherKeyword {
		t.Fatalf("match = %+v", match)
	}
	if match.ConfiguredValue != "card" {
		t.Fatalf("configured value = %q, want the admin-authored keyword", match.ConfiguredValue)
	}
	if match.Occurrences != 3 {
		t.Fatalf("occurrences = %d, want 3", match.Occurrences)
	}
	if len(match.Locations) != 1 || match.Locations[0] != utils.LocationBody {
		t.Fatalf("locations = %v", match.Locations)
	}
}

func TestNoMessageContentIsEverPersisted(t *testing.T) {
	record := recording.NewIncidentGenerator(&spyWriter{}).
		Generate(generatordto.GenerationInput{Message: message(), Result: result()})

	rendered := fmt.Sprintf("%+v", record)

	if strings.Contains(rendered, secret) {
		t.Fatalf("the incident record contains matched message content: %s", rendered)
	}
	if strings.Contains(rendered, "the card number is") {
		t.Fatalf("the incident record contains the message body: %s", rendered)
	}
}

func TestWithheldRecipientsAreRecorded(t *testing.T) {
	evaluated := result()
	evaluated.Trigger = utils.TriggerRestriction
	evaluated.Matches = nil
	evaluated.Withheld = []dto.WithheldRecipient{{
		Email:      "b@blocked.test",
		Domain:     "blocked.test",
		PolicyID:   7,
		PolicyName: "No Competitors",
	}}
	evaluated.RestrictionViolations = []dto.RestrictionViolation{{
		Kind:       utils.KindDomain,
		Mode:       utils.RestrictionBlock,
		Value:      "blocked.test",
		PolicyID:   7,
		PolicyName: "No Competitors",
	}}

	record := recording.NewIncidentGenerator(&spyWriter{}).
		Generate(generatordto.GenerationInput{Message: message(), Result: evaluated})

	if len(record.WithheldRecipients) != 1 || record.WithheldRecipients[0].PolicyName != "No Competitors" {
		t.Fatalf("withheld = %+v", record.WithheldRecipients)
	}
	if len(record.RestrictionViolations) != 1 || record.RestrictionViolations[0].Kind != utils.KindDomain {
		t.Fatalf("violations = %+v", record.RestrictionViolations)
	}
	if len(record.Matches) != 0 {
		t.Fatalf("matches = %+v, a restriction violation skips content rules", record.Matches)
	}
}

func TestRecordDelegatesToTheAuditWriter(t *testing.T) {
	writer := &spyWriter{}

	if err := recording.NewIncidentGenerator(writer).Record(context.Background(), result(), message()); err != nil {
		t.Fatalf("Record returned %v", err)
	}

	if len(writer.records) != 1 || writer.records[0].CorrelationID != "corr-1" {
		t.Fatalf("writer received %+v", writer.records)
	}
}

func TestRecordSurfacesTheWriterFailure(t *testing.T) {
	failure := errors.New("mongo down")
	writer := &spyWriter{err: failure}

	err := recording.NewIncidentGenerator(writer).Record(context.Background(), result(), message())

	if !errors.Is(err, failure) {
		t.Fatalf("error = %v, want the writer failure", err)
	}
}

func TestUpdateActionDelegatesTheOutcome(t *testing.T) {
	writer := &spyWriter{}

	err := recording.NewIncidentGenerator(writer).UpdateAction(context.Background(), "corr-1", dto.ActionResult{
		Action: utils.ActionQuarantine,
		Status: utils.ActionFailed,
		Error:  "store unreachable",
	})
	if err != nil {
		t.Fatalf("UpdateAction returned %v", err)
	}

	if len(writer.outcomes) != 1 {
		t.Fatalf("outcomes = %+v", writer.outcomes)
	}
	if writer.outcomes[0].Status != utils.ActionFailed || writer.outcomes[0].Error != "store unreachable" {
		t.Fatalf("outcome = %+v", writer.outcomes[0])
	}
}
