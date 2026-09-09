package restrictionevalutor_test

import (
	"testing"

	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	"dpdp-backend/internal/delivery/engine/parser"
	"dpdp-backend/internal/delivery/engine/restrictionevalutor"
	"dpdp-backend/internal/delivery/utils"
)

func attachmentRestrictions(blocked, allowed map[string]dto.PolicyRef) dto.EffectiveRestrictions {
	if blocked == nil {
		blocked = map[string]dto.PolicyRef{}
	}

	return dto.EffectiveRestrictions{
		Domain:     dto.RestrictionSet{Blocked: map[string]dto.PolicyRef{}},
		Attachment: dto.RestrictionSet{Blocked: blocked, Allowed: allowed},
	}
}

func attachment(filename string) dto.Attachment {
	return dto.Attachment{
		Filename:    filename,
		Extension:   parser.ExtensionOf(filename),
		ContentType: "application/octet-stream",
	}
}

func TestBlockedExtensionIsAViolation(t *testing.T) {
	found := restrictionevalutor.NewAttachmentEvaluator().Evaluate(
		[]dto.Attachment{attachment("payload.exe")},
		attachmentRestrictions(map[string]dto.PolicyRef{"exe": ref(3, "No Executables")}, nil),
	)

	if len(found) != 1 {
		t.Fatalf("violations = %+v", found)
	}
	if found[0].Kind != utils.KindAttachment || found[0].Filename != "payload.exe" || found[0].Value != "exe" {
		t.Fatalf("violation = %+v", found[0])
	}
}

func TestUppercaseExtensionStillMatches(t *testing.T) {
	found := restrictionevalutor.NewAttachmentEvaluator().Evaluate(
		[]dto.Attachment{attachment("REPORT.PDF")},
		attachmentRestrictions(map[string]dto.PolicyRef{"pdf": ref(3, "No PDFs")}, nil),
	)

	if len(found) != 1 {
		t.Fatalf("violations = %+v, extension matching must be case insensitive", found)
	}
}

func TestExtensionOutsideAllowlistIsAViolation(t *testing.T) {
	found := restrictionevalutor.NewAttachmentEvaluator().Evaluate(
		[]dto.Attachment{attachment("notes.docx")},
		attachmentRestrictions(nil, map[string]dto.PolicyRef{"pdf": ref(2, "PDF Only")}),
	)

	if len(found) != 1 || found[0].Mode != utils.RestrictionAllow {
		t.Fatalf("violations = %+v", found)
	}
}

func TestAllowedExtensionPasses(t *testing.T) {
	found := restrictionevalutor.NewAttachmentEvaluator().Evaluate(
		[]dto.Attachment{attachment("report.pdf")},
		attachmentRestrictions(nil, map[string]dto.PolicyRef{"pdf": ref(2, "PDF Only")}),
	)

	if len(found) != 0 {
		t.Fatalf("violations = %+v, want none", found)
	}
}

func TestAttachmentBlocklistTakesPrecedence(t *testing.T) {
	found := restrictionevalutor.NewAttachmentEvaluator().Evaluate(
		[]dto.Attachment{attachment("thing.zip")},
		attachmentRestrictions(
			map[string]dto.PolicyRef{"zip": ref(1, "Blocker")},
			map[string]dto.PolicyRef{"zip": ref(2, "Allower")},
		),
	)

	if len(found) != 1 || found[0].Mode != utils.RestrictionBlock {
		t.Fatalf("violations = %+v, blocklist must win", found)
	}
}

func TestExtensionlessFileNeverMatchesABlocklist(t *testing.T) {
	found := restrictionevalutor.NewAttachmentEvaluator().Evaluate(
		[]dto.Attachment{attachment("README")},
		attachmentRestrictions(map[string]dto.PolicyRef{"exe": ref(1, "Blocker")}, nil),
	)

	if len(found) != 0 {
		t.Fatalf("violations = %+v, want none", found)
	}
}

func TestExtensionlessFileViolatesAnAllowlist(t *testing.T) {
	found := restrictionevalutor.NewAttachmentEvaluator().Evaluate(
		[]dto.Attachment{attachment("README")},
		attachmentRestrictions(nil, map[string]dto.PolicyRef{"pdf": ref(2, "PDF Only")}),
	)

	if len(found) != 1 {
		t.Fatalf("violations = %+v, an unidentifiable file cannot satisfy an allowlist", found)
	}
}

func TestContentTypeIsEvidenceNotAMatchingAttribute(t *testing.T) {
	found := restrictionevalutor.NewAttachmentEvaluator().Evaluate(
		[]dto.Attachment{{Filename: "payload.txt", Extension: "txt", ContentType: "application/x-msdownload"}},
		attachmentRestrictions(map[string]dto.PolicyRef{"exe": ref(1, "Blocker")}, nil),
	)

	if len(found) != 0 {
		t.Fatalf("violations = %+v, only the extension is matched this phase", found)
	}
}

func TestNoAttachmentsMeansNoViolations(t *testing.T) {
	found := restrictionevalutor.NewAttachmentEvaluator().Evaluate(
		nil,
		attachmentRestrictions(map[string]dto.PolicyRef{"exe": ref(1, "Blocker")}, nil),
	)

	if len(found) != 0 {
		t.Fatalf("violations = %+v, want none", found)
	}
}
