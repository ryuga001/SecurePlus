package evaluation

import (
	"context"
	"errors"
	"log/slog"

	"dpdp-backend/internal/delivery"
	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

type Components struct {
	Parser      dto.MessageParser
	Cache       dto.PolicyCache
	Domain      dto.DomainRestrictionEvaluator
	Attachment  dto.AttachmentRestrictionEvaluator
	Content     dto.ContentEngine
	Resolver    dto.ActionResolver
	Actions     dto.ActionFactory
	Incidents   dto.IncidentRecorder
	FailsClosed bool
}

type EvaluationService struct {
	components Components
}

func NewEvaluationService(components Components) *EvaluationService {
	return &EvaluationService{components: components}
}

func (s *EvaluationService) Enforce(ctx context.Context, message dto.MessageContext) (dto.Outcome, error) {
	pass := dto.Outcome{
		Result:    dto.EvaluationResult{Decision: utils.DecisionPass, EffectiveAction: utils.ActionNone},
		Delivered: message.Recipients,
	}

	compiled, err := s.components.Cache.Load(ctx, message.CustomerID, message.From)
	if err != nil {
		return s.failure(ctx, message, "policy resolution failed", err, pass)
	}

	if compiled.PolicyCount == 0 {
		return pass, nil
	}

	parsed, err := s.components.Parser.Parse(message.Raw)
	if err != nil {
		if s.components.FailsClosed {
			return dto.Outcome{}, err
		}

		slog.WarnContext(ctx, "message could not be parsed, evaluating on headers only",
			"correlation_id", message.CorrelationID,
			"customer_id", message.CustomerID,
			"error", err,
		)
	}

	result := dto.EvaluationResult{
		CorrelationID:        message.CorrelationID,
		CustomerID:           message.CustomerID,
		EmailUserID:          compiled.EmailUserID,
		Decision:             utils.DecisionPass,
		EffectiveAction:      utils.ActionNone,
		EvaluatedPolicyCount: compiled.PolicyCount,
		Recipients:           message.Recipients,
	}

	domainViolations := s.components.Domain.Evaluate(message.Recipients, compiled.Restrictions)
	attachmentViolations := s.components.Attachment.Evaluate(parsed.Attachments, compiled.Restrictions)

	if len(domainViolations) > 0 || len(attachmentViolations) > 0 {
		result.Decision = utils.DecisionFlagged
		result.Trigger = utils.TriggerRestriction
		result.EffectiveAction = utils.ActionBlock
		result.RestrictionViolations = append(domainViolations, attachmentViolations...)
		result.Withheld = withheld(domainViolations)
		result.TriggeredPolicyIDs = policyIDs(result.RestrictionViolations, nil)

		delivered := survivors(message.Recipients, result.Withheld)

		if err := s.report(ctx, message, result, parsed.Subject); err != nil {
			return dto.Outcome{}, err
		}

		if len(delivered) == 0 && len(domainViolations) > 0 {
			return dto.Outcome{Result: result, Withheld: result.Withheld}, utils.ErrAllRecipientsRestricted
		}

		return dto.Outcome{Result: result, Withheld: result.Withheld, Delivered: delivered}, nil
	}

	matches := s.components.Content.Evaluate(dto.MatchInput{Parts: parsed.Parts}, *compiled)
	if len(matches) == 0 {
		return pass, nil
	}

	result.Decision = utils.DecisionFlagged
	result.Trigger = utils.TriggerContent
	result.EffectiveAction = s.components.Resolver.Resolve(matches)
	result.Matches = matches
	result.TriggeredPolicyIDs = policyIDs(nil, matches)

	if err := s.report(ctx, message, result, parsed.Subject); err != nil {
		return dto.Outcome{}, err
	}

	return dto.Outcome{Result: result, Delivered: message.Recipients}, nil
}

func (s *EvaluationService) report(ctx context.Context, message dto.MessageContext, result dto.EvaluationResult, subject string) error {
	if err := s.components.Incidents.Record(ctx, result, message); err != nil {
		slog.ErrorContext(ctx, "incident not recorded, action not invoked",
			"correlation_id", message.CorrelationID,
			"customer_id", message.CustomerID,
			"error", err,
		)

		if s.components.FailsClosed {
			return err
		}

		return nil
	}

	executor, ok := s.components.Actions.For(result.EffectiveAction)
	if !ok {
		slog.ErrorContext(ctx, "no executor registered for action",
			"correlation_id", message.CorrelationID,
			"action", result.EffectiveAction,
		)

		return nil
	}

	outcome, err := executor.Execute(ctx, dto.ActionRequest{
		CorrelationID:   message.CorrelationID,
		CustomerID:      message.CustomerID,
		ConfigID:        message.ConfigID,
		Action:          result.EffectiveAction,
		Trigger:         result.Trigger,
		Sender:          message.From,
		SenderDomain:    message.SenderDomain,
		MessageID:       message.MessageID,
		Subject:         subject,
		Recipients:      result.Recipients,
		Blocked:         blockedAddresses(result),
		PolicyNames:     policyNames(result),
		TriggeredPolicy: result.TriggeredPolicyIDs,
	})
	if err != nil {
		outcome = dto.ActionResult{
			Action: result.EffectiveAction,
			Status: utils.ActionFailed,
			Error:  err.Error(),
		}
	}

	if updateErr := s.components.Incidents.UpdateAction(ctx, message.CorrelationID, outcome); updateErr != nil {
		slog.WarnContext(ctx, "incident action outcome not updated",
			"correlation_id", message.CorrelationID,
			"error", updateErr,
		)
	}

	return nil
}

func (s *EvaluationService) failure(
	ctx context.Context,
	message dto.MessageContext,
	reason string,
	cause error,
	pass dto.Outcome,
) (dto.Outcome, error) {
	if errors.Is(cause, context.Canceled) || s.components.FailsClosed {
		return dto.Outcome{}, cause
	}

	slog.ErrorContext(ctx, reason,
		"correlation_id", message.CorrelationID,
		"customer_id", message.CustomerID,
		"error", cause,
	)

	return pass, nil
}

func withheld(violations []dto.RestrictionViolation) []dto.WithheldRecipient {
	recipients := make([]dto.WithheldRecipient, 0, len(violations))

	for _, violation := range violations {
		if violation.Recipient == "" {
			continue
		}

		recipients = append(recipients, dto.WithheldRecipient{
			Email:      violation.Recipient,
			Domain:     violation.Value,
			PolicyID:   violation.PolicyID,
			PolicyName: violation.PolicyName,
		})
	}

	return recipients
}

func survivors(recipients []string, blocked []dto.WithheldRecipient) []string {
	if len(blocked) == 0 {
		return recipients
	}

	removed := make(map[string]bool, len(blocked))
	for _, recipient := range blocked {
		removed[recipient.Email] = true
	}

	kept := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		if !removed[recipient] {
			kept = append(kept, recipient)
		}
	}

	return kept
}

func blockedAddresses(result dto.EvaluationResult) []string {
	addresses := make([]string, 0, len(result.Withheld))

	for _, recipient := range result.Withheld {
		addresses = append(addresses, recipient.Email)
	}

	return addresses
}

func policyNames(result dto.EvaluationResult) []string {
	seen := map[string]bool{}
	names := make([]string, 0)

	add := func(name string) {
		if name == "" || seen[name] {
			return
		}

		seen[name] = true
		names = append(names, name)
	}

	for _, violation := range result.RestrictionViolations {
		add(violation.PolicyName)
	}

	for _, match := range result.Matches {
		add(match.PolicyName)
	}

	return names
}

func policyIDs(violations []dto.RestrictionViolation, matches []dto.RuleMatch) []int {
	seen := map[int]bool{}
	ids := make([]int, 0)

	for _, violation := range violations {
		if violation.PolicyID > 0 && !seen[violation.PolicyID] {
			seen[violation.PolicyID] = true
			ids = append(ids, violation.PolicyID)
		}
	}

	for _, match := range matches {
		if match.PolicyID > 0 && !seen[match.PolicyID] {
			seen[match.PolicyID] = true
			ids = append(ids, match.PolicyID)
		}
	}

	return ids
}

func Withhold(recipients []dto.WithheldRecipient) []delivery.RecipientResult {
	results := make([]delivery.RecipientResult, 0, len(recipients))

	for _, recipient := range recipients {
		results = append(results, delivery.RecipientResult{
			Email:     recipient.Email,
			Domain:    recipient.Domain,
			Status:    utils.StatusBlocked,
			Error:     "withheld by policy: " + recipient.PolicyName,
			Permanent: true,
		})
	}

	return results
}
