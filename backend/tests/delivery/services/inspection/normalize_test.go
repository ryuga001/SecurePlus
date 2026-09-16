package inspection_test

import (
	"testing"

	"dpdp-backend/internal/delivery/services/inspection"
)

func TestNormalizeFoldsCase(t *testing.T) {
	for _, value := range []string{"PASSWORD", "Password", "password", "PaSsWoRd"} {
		if got := inspection.Normalize(value); got != "password" {
			t.Fatalf("Normalize(%q) = %q", value, got)
		}
	}
}

func TestNormalizeAppliesCompatibilityFolding(t *testing.T) {
	if got := inspection.Normalize("ｃｏｎｆｉｄｅｎｔｉａｌ"); got != "confidential" {
		t.Fatalf("Normalize = %q, want confidential", got)
	}
}

func TestNormalizePreservesPunctuationAndSpacing(t *testing.T) {
	if got := inspection.Normalize("Credit  Card, Number."); got != "credit  card, number." {
		t.Fatalf("Normalize = %q", got)
	}
}

func TestNormalizeIsIdempotent(t *testing.T) {
	once := inspection.Normalize("Ｓｅｃｒｅｔ")

	if twice := inspection.Normalize(once); twice != once {
		t.Fatalf("Normalize is not idempotent: %q then %q", once, twice)
	}
}
