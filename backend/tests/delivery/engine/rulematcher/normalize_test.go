package rulematcher_test

import (
	"testing"

	"dpdp-backend/internal/delivery/engine/rulematcher"
)

func TestNormalizeFoldsCase(t *testing.T) {
	for _, value := range []string{"PASSWORD", "Password", "password", "PaSsWoRd"} {
		if got := rulematcher.Normalize(value); got != "password" {
			t.Fatalf("Normalize(%q) = %q", value, got)
		}
	}
}

func TestNormalizeAppliesCompatibilityFolding(t *testing.T) {
	if got := rulematcher.Normalize("ｃｏｎｆｉｄｅｎｔｉａｌ"); got != "confidential" {
		t.Fatalf("Normalize = %q, want confidential", got)
	}
}

func TestNormalizePreservesPunctuationAndSpacing(t *testing.T) {
	if got := rulematcher.Normalize("Credit  Card, Number."); got != "credit  card, number." {
		t.Fatalf("Normalize = %q", got)
	}
}

func TestNormalizeIsIdempotent(t *testing.T) {
	once := rulematcher.Normalize("Ｓｅｃｒｅｔ")

	if twice := rulematcher.Normalize(once); twice != once {
		t.Fatalf("Normalize is not idempotent: %q then %q", once, twice)
	}
}
