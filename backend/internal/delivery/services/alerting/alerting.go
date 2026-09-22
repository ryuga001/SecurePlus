package alerting

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"dpdp-backend/internal/delivery"
	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
	"dpdp-backend/internal/notification"
)

const defaultTimeout = 10 * time.Second

type Alert struct {
	ID         string
	Name       string
	Recipients []string
}

type Lookup interface {
	RealTimeEmailAlerts(ctx context.Context, customerID int, policyIDs []int) ([]Alert, error)
}

type OrgLookup interface {
	OrgName(ctx context.Context, customerID int) (string, error)
}

type Options struct {
	Alerts      Lookup
	Queue       notification.Queue
	Orgs        OrgLookup
	FrontendURL string
	Timeout     time.Duration
	Now         func() time.Time
}

type Notifier struct {
	opts Options
}

func New(opts Options) *Notifier {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Timeout <= 0 {
		opts.Timeout = defaultTimeout
	}

	return &Notifier{opts: opts}
}

func (n *Notifier) Raise(
	ctx context.Context,
	message delivery.EmailMessage,
	result dto.EvaluationResult,
	subject string,
) {
	if n.opts.Alerts == nil || n.opts.Queue == nil {
		return
	}

	if len(result.TriggeredPolicyIDs) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), n.opts.Timeout)
	defer cancel()

	alerts, err := n.opts.Alerts.RealTimeEmailAlerts(ctx, message.CustomerID, result.TriggeredPolicyIDs)
	if err != nil {
		slog.ErrorContext(ctx, "breach alert lookup failed",
			"correlation_id", message.CorrelationID,
			"customer_id", message.CustomerID,
			"error", err,
		)

		return
	}

	targets := dedupe(alerts)
	if len(targets) == 0 {
		return
	}

	vars := n.variables(ctx, message, result, subject)
	published := 0

	for _, alert := range targets {
		msg := notification.NotificationMessage{
			CustomerID:    message.CustomerID,
			MessageType:   notification.MessageTypeEmail,
			TemplateTitle: notification.TemplatePolicyBreachAlert,
			To:            alert.Recipients,
			Body:          withAlertName(vars, alert.Name),
			CorrelationID: message.CorrelationID,
			AlertID:       alert.ID,
		}

		if err := n.opts.Queue.Publish(ctx, msg); err != nil {
			slog.ErrorContext(ctx, "breach alert publish failed",
				"correlation_id", message.CorrelationID,
				"alert_id", alert.ID,
				"error", err,
			)

			continue
		}

		published++
	}

	slog.InfoContext(ctx, "breach alerts published",
		"correlation_id", message.CorrelationID,
		"customer_id", message.CustomerID,
		"alerts", len(targets),
		"published", published,
	)
}

func (n *Notifier) variables(
	ctx context.Context,
	message delivery.EmailMessage,
	result dto.EvaluationResult,
	subject string,
) map[string]string {
	orgName := ""

	if n.opts.Orgs != nil {
		name, err := n.opts.Orgs.OrgName(ctx, message.CustomerID)
		if err != nil {
			slog.WarnContext(ctx, "breach alert org name lookup failed",
				"correlation_id", message.CorrelationID,
				"error", err,
			)
		} else {
			orgName = name
		}
	}

	if strings.TrimSpace(subject) == "" {
		subject = utils.NoticeNoSubject
	}

	names := policyNames(result)

	return map[string]string{
		"org_name":        orgName,
		"alert_name":      "",
		"sender":          message.From,
		"recipients":      strings.Join(result.Recipients, ", "),
		"subject":         subject,
		"decision":        result.Decision,
		"trigger":         result.Trigger,
		"action":          result.EffectiveAction,
		"policy_name":     strings.Join(names, ", "),
		"policy_count":    strconv.Itoa(len(names)),
		"match_count":     strconv.Itoa(len(result.Matches)),
		"violation_count": strconv.Itoa(len(result.RestrictionViolations)),
		"withheld_count":  strconv.Itoa(len(result.Withheld)),
		"message_id":      message.MessageID,
		"correlation_id":  message.CorrelationID,
		"detected_at":     n.opts.Now().UTC().Format(time.RFC1123Z),
		"incident_url":    n.incidentURL(message.CorrelationID),
	}
}

func (n *Notifier) incidentURL(correlationID string) string {
	base := strings.TrimSuffix(n.opts.FrontendURL, "/")
	if base == "" {
		return ""
	}

	return base + "/admin/email/audits/incidents/" + correlationID
}

func withAlertName(vars map[string]string, name string) map[string]string {
	copied := make(map[string]string, len(vars))
	for key, value := range vars {
		copied[key] = value
	}

	copied["alert_name"] = name

	return copied
}

func dedupe(alerts []Alert) []Alert {
	seen := make(map[string]bool, len(alerts))
	unique := make([]Alert, 0, len(alerts))

	for _, alert := range alerts {
		if alert.ID == "" || seen[alert.ID] {
			continue
		}

		seen[alert.ID] = true
		unique = append(unique, alert)
	}

	return unique
}

func policyNames(result dto.EvaluationResult) []string {
	seen := make(map[string]bool)
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
