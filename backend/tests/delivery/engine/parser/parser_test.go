package parser_test

import (
	"errors"
	"strings"
	"testing"

	"dpdp-backend/internal/delivery/engine/parser"
	"dpdp-backend/internal/delivery/utils"
)

func bodyOf(t *testing.T, raw string) string {
	t.Helper()

	parsed, err := parser.NewMessageParser().Parse([]byte(raw))
	if err != nil {
		t.Fatalf("Parse returned %v", err)
	}

	for _, part := range parsed.Parts {
		if part.Location == utils.LocationBody {
			return part.Text
		}
	}

	return ""
}

func TestParsesPlainTextMessage(t *testing.T) {
	parsed, err := parser.NewMessageParser().Parse([]byte(
		"From: a@b.test\r\nTo: c@d.test\r\nSubject: Quarterly numbers\r\n\r\nthe body text\r\n",
	))
	if err != nil {
		t.Fatalf("Parse returned %v", err)
	}

	if parsed.Subject != "Quarterly numbers" {
		t.Fatalf("subject = %q", parsed.Subject)
	}
	if len(parsed.Parts) != 2 {
		t.Fatalf("parts = %+v, want subject and body", parsed.Parts)
	}
	if !strings.Contains(bodyOf(t, "From: a@b.test\r\nSubject: s\r\n\r\nthe body text\r\n"), "the body text") {
		t.Fatal("body text was not extracted")
	}
}

func TestDecodesQuotedPrintableBody(t *testing.T) {
	raw := "From: a@b.test\r\nSubject: s\r\nContent-Type: text/plain\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n\r\nthis is confid=\r\nential\r\n"

	if body := bodyOf(t, raw); !strings.Contains(body, "confidential") {
		t.Fatalf("body = %q, quoted-printable was not decoded", body)
	}
}

func TestExtractsAttachmentNameAndExtension(t *testing.T) {
	raw := "From: a@b.test\r\nSubject: s\r\n" +
		"Content-Type: multipart/mixed; boundary=\"sep\"\r\n\r\n" +
		"--sep\r\nContent-Type: text/plain\r\n\r\nsee attached\r\n" +
		"--sep\r\nContent-Type: application/pdf\r\n" +
		"Content-Disposition: attachment; filename=\"Report.PDF\"\r\n" +
		"Content-Transfer-Encoding: base64\r\n\r\naGVsbG8=\r\n--sep--\r\n"

	parsed, err := parser.NewMessageParser().Parse([]byte(raw))
	if err != nil {
		t.Fatalf("Parse returned %v", err)
	}

	if len(parsed.Attachments) != 1 {
		t.Fatalf("attachments = %+v", parsed.Attachments)
	}
	if parsed.Attachments[0].Filename != "Report.PDF" || parsed.Attachments[0].Extension != "pdf" {
		t.Fatalf("attachment = %+v", parsed.Attachments[0])
	}
}

func TestFallsBackToHtmlWhenNoTextPart(t *testing.T) {
	raw := "From: a@b.test\r\nSubject: s\r\nContent-Type: text/html\r\n\r\n" +
		"<html><body><p>visible confidential text</p></body></html>\r\n"

	body := bodyOf(t, raw)

	if !strings.Contains(body, "visible confidential text") {
		t.Fatalf("body = %q", body)
	}
	if strings.Contains(body, "<p>") || strings.Contains(body, "body") {
		t.Fatalf("body = %q, markup must be stripped", body)
	}
}

func TestVisibleTextDropsScriptAndStyleContent(t *testing.T) {
	got := parser.VisibleText(
		`<style>.confidential{color:red}</style><script>var secret="hidden"</script><p>shown</p>`,
	)

	if got != "shown" {
		t.Fatalf("VisibleText = %q, want only the visible text", got)
	}
}

func TestVisibleTextDropsAttributeValues(t *testing.T) {
	got := parser.VisibleText(`<div class="confidential" title="secret">plain</div>`)

	if got != "plain" {
		t.Fatalf("VisibleText = %q, attribute values must not be matchable", got)
	}
}

func TestMalformedMessageDegradesInsteadOfFailingHard(t *testing.T) {
	parsed, err := parser.NewMessageParser().Parse([]byte("this is not a message at all"))

	if err != nil && !errors.Is(err, utils.ErrMessageUnreadable) {
		t.Fatalf("error = %v, want ErrMessageUnreadable or nil", err)
	}
	if parsed.Attachments == nil {
		t.Fatal("attachments must never be nil")
	}
}

func TestExtensionOf(t *testing.T) {
	cases := map[string]string{
		"report.pdf":  "pdf",
		"REPORT.PDF":  "pdf",
		"archive.TAR": "tar",
		"README":      "",
		"":            "",
		"a.b.c.docx":  "docx",
		"trailing.":   "",
	}

	for filename, want := range cases {
		if got := parser.ExtensionOf(filename); got != want {
			t.Fatalf("ExtensionOf(%q) = %q, want %q", filename, got, want)
		}
	}
}
