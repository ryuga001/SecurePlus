package services_test

import (
	"errors"
	"strings"
	"testing"

	providersvc "dpdp-backend/internal/admin/services/emailprovider"
	"dpdp-backend/internal/admin/utils"
)

func TestNormalizeDomain(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"trims whitespace", "  example.com  ", "example.com"},
		{"lowercases", "Example.COM", "example.com"},
		{"strips trailing dot", "example.com.", "example.com"},
		{"strips scheme", "https://example.com", "example.com"},
		{"strips www", "www.example.com", "example.com"},
		{"strips everything", "  HTTPS://WWW.Example.COM.  ", "example.com"},
		{"keeps subdomain", "Mail.Example.co.uk", "mail.example.co.uk"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := providersvc.NormalizeDomain(tc.input)
			if got != tc.want {
				t.Fatalf("providersvc.NormalizeDomain(%q) = %q, want %q", tc.input, got, tc.want)
			}

			if again := providersvc.NormalizeDomain(got); again != got {
				t.Fatalf("providersvc.NormalizeDomain is not idempotent: %q -> %q", got, again)
			}
		})
	}
}

func TestValidDomainAccepts(t *testing.T) {
	valid := []string{"example.com", "mail.example.co.uk", "a-b.example.com", "xn--80ak6aa92e.com", "example.travel"}

	for _, domain := range valid {
		if !providersvc.ValidDomain(domain) {
			t.Errorf("providersvc.ValidDomain(%q) = false, want true", domain)
		}
	}
}

func TestValidDomainRejects(t *testing.T) {
	invalid := map[string]string{
		"empty":               "",
		"no tld":              "example",
		"leading hyphen":      "-example.com",
		"trailing hyphen":     "example-.com",
		"empty label":         "example..com",
		"scheme":              "https://example.com",
		"path":                "example.com/inbox",
		"space":               "exa mple.com",
		"numeric tld":         "1.2.3.4",
		"single char tld":     "example.c",
		"uppercase":           "Example.com",
		"label over 63 chars": strings.Repeat("a", 64) + ".com",
		"over 253 chars":      strings.Repeat("a.", 130) + "com",
		"underscore":          "exa_mple.com",
	}

	for name, domain := range invalid {
		t.Run(name, func(t *testing.T) {
			if providersvc.ValidDomain(domain) {
				t.Fatalf("providersvc.ValidDomain(%q) = true, want false", domain)
			}
		})
	}
}

func TestValidProvider(t *testing.T) {
	if !providersvc.ValidProvider(utils.ProviderOutlook365) || !providersvc.ValidProvider(utils.ProviderGmail) {
		t.Fatal("known providers must be valid")
	}

	for _, provider := range []string{"", "outlook", "Gmail", "smtp"} {
		if providersvc.ValidProvider(provider) {
			t.Errorf("providersvc.ValidProvider(%q) = true, want false", provider)
		}
	}
}

func TestNormalizeConfigurationInput(t *testing.T) {
	input, err := providersvc.NormalizeConfigurationInput(providersvc.ConfigurationInput{
		Name:     "  Corporate   outbound ",
		Domain:   "  WWW.Example.COM. ",
		Provider: " Gmail ",
	})
	if err != nil {
		t.Fatalf("providersvc.NormalizeConfigurationInput returned %v", err)
	}

	if input.Name != "Corporate outbound" || input.Domain != "example.com" || input.Provider != utils.ProviderGmail {
		t.Fatalf("normalized = %+v", input)
	}
}

func TestNormalizeConfigurationInputRejects(t *testing.T) {
	_, err := providersvc.NormalizeConfigurationInput(providersvc.ConfigurationInput{Name: "x", Domain: "not a domain", Provider: utils.ProviderGmail})
	if !errors.Is(err, utils.ErrInvalidDomain) {
		t.Fatalf("expected ErrInvalidDomain, got %v", err)
	}

	_, err = providersvc.NormalizeConfigurationInput(providersvc.ConfigurationInput{Name: "x", Domain: "example.com", Provider: "smtp"})
	if !errors.Is(err, utils.ErrInvalidProvider) {
		t.Fatalf("expected ErrInvalidProvider, got %v", err)
	}
}
