package auth_test

import (
	"strings"
	"testing"

	"dpdp-backend/internal/auth"
)

const testPepper = "test-pepper-value"

func TestNewSaltProducesUniqueValues(t *testing.T) {
	seen := make(map[string]bool)

	for range 20 {
		salt, err := auth.NewSalt()
		if err != nil {
			t.Fatalf("NewSalt returned %v", err)
		}
		if salt == "" {
			t.Fatal("salt is empty")
		}
		if seen[salt] {
			t.Fatalf("duplicate salt generated: %s", salt)
		}

		seen[salt] = true
	}
}

func TestHashPasswordRoundTrips(t *testing.T) {
	salt, err := auth.NewSalt()
	if err != nil {
		t.Fatalf("NewSalt returned %v", err)
	}

	hash, err := auth.HashPassword(testPepper, "correct horse battery staple", salt, 4)
	if err != nil {
		t.Fatalf("HashPassword returned %v", err)
	}

	if !strings.HasPrefix(hash, "v1$") {
		t.Fatalf("hash = %q, want a v1$ prefix", hash)
	}

	if err := auth.VerifyPassword(testPepper, hash, salt, "correct horse battery staple"); err != nil {
		t.Fatalf("VerifyPassword rejected the correct password: %v", err)
	}
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	salt, _ := auth.NewSalt()
	hash, _ := auth.HashPassword(testPepper, "the-real-password", salt, 4)

	err := auth.VerifyPassword(testPepper, hash, salt, "a-guessed-password")
	if err != auth.ErrInvalidCredentials {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
}

func TestVerifyPasswordRejectsWrongPepper(t *testing.T) {
	salt, _ := auth.NewSalt()
	hash, _ := auth.HashPassword(testPepper, "the-real-password", salt, 4)

	err := auth.VerifyPassword("a-different-pepper", hash, salt, "the-real-password")
	if err != auth.ErrInvalidCredentials {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
}

func TestVerifyPasswordRejectsWrongSalt(t *testing.T) {
	saltA, _ := auth.NewSalt()
	saltB, _ := auth.NewSalt()
	hash, _ := auth.HashPassword(testPepper, "the-real-password", saltA, 4)

	err := auth.VerifyPassword(testPepper, hash, saltB, "the-real-password")
	if err != auth.ErrInvalidCredentials {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
}

func TestVerifyPasswordRejectsMalformedStoredHash(t *testing.T) {
	salt, _ := auth.NewSalt()

	for _, stored := range []string{"", "no-dollar-separator", "$", "v1$"} {
		if err := auth.VerifyPassword(testPepper, stored, salt, "anything"); err != auth.ErrInvalidCredentials {
			t.Errorf("stored %q: error = %v, want ErrInvalidCredentials", stored, err)
		}
	}
}

func TestHashPasswordProducesDifferentHashesForTheSamePassword(t *testing.T) {
	salt, _ := auth.NewSalt()

	first, err := auth.HashPassword(testPepper, "same-password", salt, 4)
	if err != nil {
		t.Fatalf("HashPassword returned %v", err)
	}

	second, err := auth.HashPassword(testPepper, "same-password", salt, 4)
	if err != nil {
		t.Fatalf("HashPassword returned %v", err)
	}

	if first == second {
		t.Fatal("hashing the same password twice produced identical output; bcrypt's own salt is not being applied")
	}

	if err := auth.VerifyPassword(testPepper, second, salt, "same-password"); err != nil {
		t.Fatalf("second hash did not verify: %v", err)
	}
}

func TestVerifyPasswordIsSensitiveToASingleCharacter(t *testing.T) {
	salt, _ := auth.NewSalt()
	hash, _ := auth.HashPassword(testPepper, "Password123!", salt, 4)

	if err := auth.VerifyPassword(testPepper, hash, salt, "password123!"); err != auth.ErrInvalidCredentials {
		t.Fatalf("case-differing password was accepted: %v", err)
	}
	if err := auth.VerifyPassword(testPepper, hash, salt, "Password123! "); err != auth.ErrInvalidCredentials {
		t.Fatalf("trailing-space password was accepted: %v", err)
	}
}

func TestBurnTimeDoesNotPanic(t *testing.T) {
	auth.BurnTime(testPepper, "whatever-was-typed")
}
