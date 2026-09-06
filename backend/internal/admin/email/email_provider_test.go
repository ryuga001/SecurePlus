package email

import (
	"strings"
	"testing"
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
			got := NormalizeDomain(tc.input)
			if got != tc.want {
				t.Fatalf("NormalizeDomain(%q) = %q, want %q", tc.input, got, tc.want)
			}

			if again := NormalizeDomain(got); again != got {
				t.Fatalf("NormalizeDomain is not idempotent: %q -> %q", got, again)
			}
		})
	}
}

func TestValidDomainAccepts(t *testing.T) {
	valid := []string{
		"example.com",
		"mail.example.co.uk",
		"a-b.example.com",
		"xn--80ak6aa92e.com",
		"example.travel",
	}

	for _, domain := range valid {
		if !ValidDomain(domain) {
			t.Errorf("ValidDomain(%q) = false, want true", domain)
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
			if ValidDomain(domain) {
				t.Fatalf("ValidDomain(%q) = true, want false", domain)
			}
		})
	}
}

func TestValidProvider(t *testing.T) {
	if !ValidProvider(ProviderOutlook365) || !ValidProvider(ProviderGmail) {
		t.Fatal("known providers must be valid")
	}

	for _, provider := range []string{"", "outlook", "Gmail", "smtp"} {
		if ValidProvider(provider) {
			t.Errorf("ValidProvider(%q) = true, want false", provider)
		}
	}
}

func TestNormalizeProvider(t *testing.T) {
	if got := NormalizeProvider("  Gmail "); got != ProviderGmail {
		t.Fatalf("NormalizeProvider = %q, want %q", got, ProviderGmail)
	}
}

func TestNormalizeName(t *testing.T) {
	if got := NormalizeName("  Corporate   outbound  "); got != "Corporate outbound" {
		t.Fatalf("NormalizeName = %q", got)
	}
}

func TestNormalizePaging(t *testing.T) {
	cases := []struct {
		page, pageSize   int
		wantPage, wantSz int
	}{
		{0, 0, defaultPage, defaultPageSize},
		{-3, -1, defaultPage, defaultPageSize},
		{2, 10, 2, 10},
		{1, 500, 1, maxPageSize},
	}

	for _, tc := range cases {
		page, size := normalizePaging(tc.page, tc.pageSize)
		if page != tc.wantPage || size != tc.wantSz {
			t.Errorf("normalizePaging(%d,%d) = (%d,%d), want (%d,%d)",
				tc.page, tc.pageSize, page, size, tc.wantPage, tc.wantSz)
		}
	}
}

func TestEscapeLike(t *testing.T) {
	if got := escapeLike(`100%_off\now`); got != `100\%\_off\\now` {
		t.Fatalf("escapeLike = %q", got)
	}
}
