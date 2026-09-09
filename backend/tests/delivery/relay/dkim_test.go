package relay_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"
	"testing"

	"dpdp-backend/internal/delivery"
	"dpdp-backend/internal/delivery/relay"
	deliveryutils "dpdp-backend/internal/delivery/utils"
)

const rawMessage = "From: alice@example.com\r\n" +
	"To: bob@other.test\r\n" +
	"Subject: quarterly report\r\n" +
	"Message-ID: <abc@example.com>\r\n" +
	"Date: Mon, 08 Sep 2026 09:00:00 +0000\r\n" +
	"\r\n" +
	"body line one\r\nbody line two\r\n"

func testConfig(t *testing.T) delivery.TenantConfig {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	encoded, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("key marshal failed: %v", err)
	}

	return delivery.TenantConfig{
		CustomerID:     1,
		ConfigID:       2,
		Domain:         "example.com",
		DKIMSelector:   deliveryutils.DefaultDKIMSelector,
		DKIMPrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded})),
	}
}

func TestSignAddsSignature(t *testing.T) {
	signed, err := relay.Sign([]byte(rawMessage), testConfig(t))
	if err != nil {
		t.Fatalf("Sign returned %v", err)
	}

	header := string(signed[:strings.Index(string(signed), "\r\n\r\n")])

	if !strings.Contains(header, "DKIM-Signature:") {
		t.Fatalf("no signature header in %q", header)
	}
	if !strings.Contains(header, "s=dpdp") || !strings.Contains(header, "d=example.com") {
		t.Fatalf("selector or domain missing: %q", header)
	}
	if !strings.Contains(header, "c=relaxed/relaxed") {
		t.Fatalf("canonicalization missing: %q", header)
	}

	for _, covered := range []string{"from", "to", "subject", "message-id"} {
		if !strings.Contains(strings.ToLower(header), covered) {
			t.Errorf("header %q not covered", covered)
		}
	}

	if !strings.Contains(string(signed), "body line one") {
		t.Fatal("body was not preserved")
	}
}

func TestSignPreservesExistingSignature(t *testing.T) {
	incoming := "DKIM-Signature: v=1; a=rsa-sha256; d=sender.test; s=other; b=AAAA\r\n" + rawMessage

	signed, err := relay.Sign([]byte(incoming), testConfig(t))
	if err != nil {
		t.Fatalf("Sign returned %v", err)
	}

	body := string(signed)

	if strings.Count(body, "DKIM-Signature:") != 2 {
		t.Fatalf("expected two signatures, got %d", strings.Count(body, "DKIM-Signature:"))
	}
	if !strings.Contains(body, "d=sender.test") {
		t.Fatal("incoming signature was stripped")
	}
	if !strings.Contains(body, "d=example.com") {
		t.Fatal("our signature is missing")
	}
}

func TestSignRejectsMissingKey(t *testing.T) {
	cfg := testConfig(t)
	cfg.DKIMPrivateKey = ""

	if _, err := relay.Sign([]byte(rawMessage), cfg); !errors.Is(err, deliveryutils.ErrPrivateKeyMissing) {
		t.Fatalf("error = %v, want ErrPrivateKeyMissing", err)
	}
}

func TestSignRejectsInvalidKey(t *testing.T) {
	cfg := testConfig(t)
	cfg.DKIMPrivateKey = "-----BEGIN PRIVATE KEY-----\nbm90IGEga2V5\n-----END PRIVATE KEY-----\n"

	if _, err := relay.Sign([]byte(rawMessage), cfg); !errors.Is(err, deliveryutils.ErrPrivateKeyInvalid) {
		t.Fatalf("error = %v, want ErrPrivateKeyInvalid", err)
	}
}

func TestSignFallsBackToDefaultSelector(t *testing.T) {
	cfg := testConfig(t)
	cfg.DKIMSelector = ""

	signed, err := relay.Sign([]byte(rawMessage), cfg)
	if err != nil {
		t.Fatalf("Sign returned %v", err)
	}

	if !strings.Contains(string(signed), "s=dpdp") {
		t.Fatal("default selector was not applied")
	}
}
