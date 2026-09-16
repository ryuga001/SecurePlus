package recording

import (
	"context"

	auditdto "dpdp-backend/internal/audit/dto/emailincident"
	auditutils "dpdp-backend/internal/audit/utils"
	"dpdp-backend/internal/delivery/dto/delivery"
	evaluation "dpdp-backend/internal/delivery/dto/evaluation"
)

type IncidentGenerator struct {
	writer auditdto.Recorder
}

func NewIncidentGenerator(writer auditdto.Recorder) *IncidentGenerator {
	return &IncidentGenerator{writer: writer}
}

func (g *IncidentGenerator) Record(ctx context.Context, result evaluation.EvaluationResult, message delivery.EmailMessage) error {
	return g.writer.Record(ctx, g.Generate(GenerationInput{Message: message, Result: result}))
}

func (g *IncidentGenerator) UpdateAction(ctx context.Context, correlationID string, result evaluation.ActionResult) error {
	return g.writer.UpdateAction(ctx, correlationID, auditdto.ActionOutcome{
		Action: result.Action,
		Status: result.Status,
		Error:  result.Error,
	})
}

func (g *IncidentGenerator) Generate(input GenerationInput) auditdto.Record {
	message := input.Message
	result := input.Result

	return auditdto.Record{
		CorrelationID:         message.CorrelationID,
		MessageID:             message.MessageID,
		CustomerID:            message.CustomerID,
		ConfigID:              message.ConfigID,
		From:                  message.From,
		SenderDomain:          message.SenderDomain,
		Recipients:            recipients(message.Recipients),
		EmailUserID:           result.EmailUserID,
		EvaluatedPolicyCount:  result.EvaluatedPolicyCount,
		TriggeredPolicyIDs:    result.TriggeredPolicyIDs,
		Decision:              result.Decision,
		Trigger:               result.Trigger,
		EffectiveAction:       result.EffectiveAction,
		ActionInvoked:         result.EffectiveAction,
		ActionStatus:          auditutils.ActionPending,
		WithheldRecipients:    withheld(result.Withheld),
		RestrictionViolations: violations(result.RestrictionViolations),
		Matches:               matches(result.Matches),
	}
}

func recipients(addresses []string) []auditdto.Recipient {
	entries := make([]auditdto.Recipient, 0, len(addresses))

	for _, address := range addresses {
		entries = append(entries, auditdto.Recipient{
			Email:  address,
			Domain: delivery.DomainOf(address),
		})
	}

	return entries
}

func withheld(recipients []evaluation.WithheldRecipient) []auditdto.WithheldRecipient {
	entries := make([]auditdto.WithheldRecipient, 0, len(recipients))

	for _, recipient := range recipients {
		entries = append(entries, auditdto.WithheldRecipient{
			Email:      recipient.Email,
			Domain:     recipient.Domain,
			PolicyID:   recipient.PolicyID,
			PolicyName: recipient.PolicyName,
		})
	}

	return entries
}

func violations(found []evaluation.RestrictionViolation) []auditdto.RestrictionViolation {
	entries := make([]auditdto.RestrictionViolation, 0, len(found))

	for _, violation := range found {
		entries = append(entries, auditdto.RestrictionViolation{
			Kind:        violation.Kind,
			Mode:        violation.Mode,
			Value:       violation.Value,
			Filename:    violation.Filename,
			ContentType: violation.ContentType,
			PolicyID:    violation.PolicyID,
			PolicyName:  violation.PolicyName,
		})
	}

	return entries
}

func matches(found []evaluation.RuleMatch) []auditdto.Match {
	entries := make([]auditdto.Match, 0, len(found))

	for _, match := range found {
		entries = append(entries, auditdto.Match{
			PolicyID:        match.PolicyID,
			PolicyName:      match.PolicyName,
			RuleID:          match.RuleID,
			RuleName:        match.RuleName,
			RuleType:        match.RuleType,
			ConfiguredValue: match.ConfiguredValue,
			Occurrences:     match.Occurrences,
			Locations:       match.Locations,
		})
	}

	return entries
}
