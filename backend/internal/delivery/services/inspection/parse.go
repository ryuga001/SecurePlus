package inspection

import (
	"bytes"
	"net/mail"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jhillyerd/enmime/v2"

	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

var (
	scriptOrStyle = regexp.MustCompile(`(?is)<(script|style)\b[^>]*>.*?</\s*(script|style)\s*>`)
	htmlTag       = regexp.MustCompile(`(?s)<[^>]*>`)
	whitespaceRun = regexp.MustCompile(`\s+`)
)

func Parse(raw []byte) (dto.ParsedMessage, error) {
	envelope, err := enmime.ReadEnvelope(bytes.NewReader(raw))
	if err != nil {
		return degraded(raw), utils.ErrMessageUnreadable
	}

	subject := strings.TrimSpace(envelope.GetHeader("Subject"))

	body := strings.TrimSpace(envelope.Text)
	if body == "" {
		body = VisibleText(envelope.HTML)
	}

	attachments := make([]dto.Attachment, 0, len(envelope.Attachments))
	for _, part := range envelope.Attachments {
		attachments = append(attachments, dto.Attachment{
			Filename:    part.FileName,
			Extension:   ExtensionOf(part.FileName),
			ContentType: part.ContentType,
			Size:        int64(len(part.Content)),
		})
	}

	return dto.ParsedMessage{
		Subject:     subject,
		Parts:       parts(subject, body),
		Attachments: attachments,
	}, nil
}

func degraded(raw []byte) dto.ParsedMessage {
	subject := ""

	if message, err := mail.ReadMessage(bytes.NewReader(raw)); err == nil {
		subject = strings.TrimSpace(message.Header.Get("Subject"))
	}

	return dto.ParsedMessage{
		Subject:     subject,
		Parts:       parts(subject, ""),
		Attachments: []dto.Attachment{},
	}
}

func parts(subject, body string) []dto.ContentPart {
	collected := make([]dto.ContentPart, 0, 2)

	if subject != "" {
		collected = append(collected, dto.ContentPart{Location: utils.LocationSubject, Text: subject})
	}

	if body != "" {
		collected = append(collected, dto.ContentPart{Location: utils.LocationBody, Text: body})
	}

	return collected
}

func VisibleText(html string) string {
	if html == "" {
		return ""
	}

	stripped := scriptOrStyle.ReplaceAllString(html, " ")
	stripped = htmlTag.ReplaceAllString(stripped, " ")
	stripped = whitespaceRun.ReplaceAllString(stripped, " ")

	return strings.TrimSpace(stripped)
}

func ExtensionOf(filename string) string {
	extension := strings.ToLower(strings.TrimPrefix(filepath.Ext(strings.TrimSpace(filename)), "."))

	if extension == "" || strings.ContainsAny(extension, " /\\") {
		return ""
	}

	return extension
}
