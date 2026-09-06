package email

import "testing"

func TestAuthorizedDomainKey(t *testing.T) {
	if got := authorizedDomainKey("example.com"); got != "authorized_domain:example.com" {
		t.Fatalf("authorizedDomainKey = %q", got)
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
			if got := staleKey(tc.previous, tc.domain); got != tc.want {
				t.Fatalf("staleKey(%q,%q) = %q, want %q", tc.previous, tc.domain, got, tc.want)
			}
		})
	}
}
