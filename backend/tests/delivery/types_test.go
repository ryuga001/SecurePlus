package delivery_test

import (
	"slices"
	"testing"

	"dpdp-backend/internal/delivery"
)

func TestDomainOf(t *testing.T) {
	cases := map[string]string{
		"alice@example.com":   "example.com",
		"  bob@Example.COM  ": "example.com",
		"carol@":              "",
		"no-at-sign":          "",
		"":                    "",
	}

	for address, want := range cases {
		if got := delivery.DomainOf(address); got != want {
			t.Errorf("DomainOf(%q) = %q, want %q", address, got, want)
		}
	}
}

func TestGroupByDomain(t *testing.T) {
	grouped := delivery.GroupByDomain([]string{
		"alice@example.com",
		"bob@example.com",
		"carol@other.test",
	})

	if len(grouped) != 2 {
		t.Fatalf("expected two domains, got %d", len(grouped))
	}

	if !slices.Equal(grouped["example.com"], []string{"alice@example.com", "bob@example.com"}) {
		t.Fatalf("example.com group = %v", grouped["example.com"])
	}
	if !slices.Equal(grouped["other.test"], []string{"carol@other.test"}) {
		t.Fatalf("other.test group = %v", grouped["other.test"])
	}
}

func TestDefaultSelector(t *testing.T) {
	if delivery.DefaultDKIMSelector != "dpdp" {
		t.Fatalf("selector = %q, want dpdp", delivery.DefaultDKIMSelector)
	}
}
