package repositories_test

import (
	"testing"

	"dpdp-backend/internal/admin/repositories/emailprovider"
)

func TestAuthorizedDomainKey(t *testing.T) {
	if got := emailprovider.AuthorizedDomainKey("example.com"); got != "authorized_domain:example.com" {
		t.Fatalf("emailprovider.AuthorizedDomainKey = %q", got)
	}
}

func TestStaleKey(t *testing.T) {
	cases := []struct {
		name     string
		previous string
		domain   string
		want     string
	}{
		{"creating", "", "example.com", ""},
		{"unchanged", "example.com", "example.com", ""},
		{"renamed", "old.com", "example.com", "authorized_domain:old.com"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := emailprovider.StaleKey(tc.previous, tc.domain); got != tc.want {
				t.Fatalf("StaleKey(%q,%q) = %q, want %q", tc.previous, tc.domain, got, tc.want)
			}
		})
	}
}
