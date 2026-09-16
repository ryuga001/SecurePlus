package adjudication

import (
	"context"
	"encoding/base64"
	"html"
	"log/slog"
	"mime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"dpdp-backend/internal/db"
	"dpdp-backend/internal/delivery/dto/delivery"
	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

type TemplateStore interface {
	Find(ctx context.Context, customerID int, name string) (db.EmailTemplate, error)
	OrgName(ctx context.Context, customerID int) (string, error)
}

type ConfigLoader interface {
	Resolve(ctx context.Context, customerID, configID int) (delivery.TenantConfig, error)
}

type BlockNoticeService struct {
	templates TemplateStore
	configs   ConfigLoader
	relay     delivery.Sender
}

func NewBlockNoticeService(templates TemplateStore, configs ConfigLoader, relay delivery.Sender) *BlockNoticeService {
	return &BlockNoticeService{templates: templates, configs: configs, relay: relay}
}

func (s *BlockNoticeService) Notify(ctx context.Context, request dto.ActionRequest) error {
	sender := strings.TrimSpace(request.Sender)
	if sender == "" {
		return utils.ErrNoticeSenderMissing
	}

	from := utils.NoticeMailbox + "@" + request.SenderDomain

	if strings.EqualFold(sender, from) {
		slog.WarnContext(ctx, "block notice skipped, sender is the notice mailbox",
			"correlation_id", request.CorrelationID,
			"sender", sender,
		)

		return nil
	}

	template, err := s.templates.Find(ctx, request.CustomerID, utils.TemplatePolicyBlockNotice)
	if err != nil {
		return err
	}

	orgName, err := s.templates.OrgName(ctx, request.CustomerID)
	if err != nil {
		return err
	}

	cfg, err := s.configs.Resolve(ctx, request.CustomerID, request.ConfigID)
	if err != nil {
		return err
	}

	vars := variables(request, orgName)

	notice := delivery.EmailMessage{
		CorrelationID: request.CorrelationID,
		CustomerID:    request.CustomerID,
		ConfigID:      request.ConfigID,
		From:          from,
		SenderDomain:  request.SenderDomain,
		Recipients:    []string{sender},
		ReceivedAt:    time.Now().UTC(),
	}

	notice.MessageID = "<" + uuid.NewString() + "@" + request.SenderDomain + ">"
	notice.Raw = compose(notice, orgName, render(template.Subject, vars, false), render(template.Body, vars, true))
	notice.Size = int64(len(notice.Raw))

	results, err := s.relay.Deliver(ctx, notice, cfg, func(attempt delivery.Attempt) {
		slog.InfoContext(ctx, "block notice relay attempt",
			"correlation_id", request.CorrelationID,
			"attempt", attempt.Number,
			"mx_host", attempt.MXHost,
			"smtp_code", attempt.SMTPCode,
			"error", attempt.Error,
		)
	})
	if err != nil {
		return err
	}

	for _, result := range results {
		if result.Status != utils.StatusSuccess {
			return utils.NoticeNotDelivered(result.Email, result.SMTPCode, result.Error)
		}
	}

	slog.InfoContext(ctx, "block notice delivered",
		"correlation_id", request.CorrelationID,
		"customer_id", request.CustomerID,
		"from", from,
		"to", sender,
	)

	return nil
}

func variables(request dto.ActionRequest, orgName string) map[string]string {
	blocked := request.Blocked
	if len(blocked) == 0 {
		blocked = request.Recipients
	}

	reason := utils.NoticeReasonContent
	if request.Trigger == utils.TriggerRestriction {
		reason = utils.NoticeReasonRestriction
	}

	subject := request.Subject
	if strings.TrimSpace(subject) == "" {
		subject = utils.NoticeNoSubject
	}

	return map[string]string{
		"org_name":       orgName,
		"policy_name":    strings.Join(request.PolicyNames, ", "),
		"policy_count":   strconv.Itoa(len(request.PolicyNames)),
		"reason":         reason,
		"subject":        subject,
		"recipients":     strings.Join(blocked, ", "),
		"message_id":     request.MessageID,
		"correlation_id": request.CorrelationID,
		"blocked_at":     time.Now().UTC().Format(time.RFC1123Z),
	}
}

func render(layout string, vars map[string]string, escape bool) string {
	pairs := make([]string, 0, len(vars)*2)

	for key, value := range vars {
		if escape {
			value = html.EscapeString(value)
		} else {
			value = strings.NewReplacer("\r", "", "\n", "").Replace(value)
		}

		pairs = append(pairs, "{{"+key+"}}", value)
	}

	return strings.NewReplacer(pairs...).Replace(layout)
}

func compose(notice delivery.EmailMessage, orgName, subject, body string) []byte {
	encoded := base64.StdEncoding.EncodeToString([]byte(body))

	var message strings.Builder

	message.WriteString("From: " + mime.QEncoding.Encode("utf-8", orgName) + " <" + notice.From + ">\r\n")
	message.WriteString("To: " + notice.Recipients[0] + "\r\n")
	message.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	message.WriteString("Message-ID: " + notice.MessageID + "\r\n")
	message.WriteString("Date: " + notice.ReceivedAt.Format(time.RFC1123Z) + "\r\n")
	message.WriteString("Auto-Submitted: auto-replied\r\n")
	message.WriteString("X-Auto-Response-Suppress: All\r\n")
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	message.WriteString("Content-Transfer-Encoding: base64\r\n")
	message.WriteString("\r\n")

	for start := 0; start < len(encoded); start += utils.NoticeLineLength {
		end := min(start+utils.NoticeLineLength, len(encoded))

		message.WriteString(encoded[start:end] + "\r\n")
	}

	return []byte(message.String())
}
