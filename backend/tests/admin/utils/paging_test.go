package utils_test

import (
	"slices"
	"strings"
	"testing"

	"dpdp-backend/internal/admin/utils"
)

func TestNormalizePaging(t *testing.T) {
	cases := []struct {
		page, pageSize   int
		wantPage, wantSz int
	}{
		{0, 0, utils.DefaultPage, utils.DefaultPageSize},
		{-3, -1, utils.DefaultPage, utils.DefaultPageSize},
		{2, 10, 2, 10},
		{1, 5000, 1, utils.MaxPageSize},
	}

	for _, tc := range cases {
		page, size := utils.NormalizePaging(tc.page, tc.pageSize)
		if page != tc.wantPage || size != tc.wantSz {
			t.Errorf("utils.NormalizePaging(%d,%d) = (%d,%d), want (%d,%d)",
				tc.page, tc.pageSize, page, size, tc.wantPage, tc.wantSz)
		}
	}
}

func TestOffset(t *testing.T) {
	if got := utils.Offset(3, 25); got != 50 {
		t.Fatalf("utils.Offset = %d", got)
	}
}

func TestEscapeLike(t *testing.T) {
	if got := utils.EscapeLike(`100%_off\now`); got != `100\%\_off\\now` {
		t.Fatalf("utils.EscapeLike = %q", got)
	}
}

func TestLikePatternWraps(t *testing.T) {
	if got := utils.LikePattern("  term  "); got != "%term%" {
		t.Fatalf("utils.LikePattern = %q", got)
	}
}

func TestNormalizeName(t *testing.T) {
	if got := utils.NormalizeName("  Corporate   outbound  "); got != "Corporate outbound" {
		t.Fatalf("utils.NormalizeName = %q", got)
	}
}

func TestNormalizeIDs(t *testing.T) {
	got := utils.NormalizeIDs([]int{5, 1, 5, 0, -2, 1})
	if !slices.Equal(got, []int{1, 5}) {
		t.Fatalf("utils.NormalizeIDs = %v", got)
	}

	if empty := utils.NormalizeIDs(nil); empty == nil || len(empty) != 0 {
		t.Fatalf("utils.NormalizeIDs(nil) = %v, want empty non-nil", empty)
	}
}

func TestNormalizeEmail(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"  Alice@Example.com  ", "alice@example.com"},
		{"bob@sub.example.co.uk", "bob@sub.example.co.uk"},
		{"", ""},
		{"   ", ""},
		{"no-at-sign.example.com", ""},
		{"@example.com", ""},
		{"user@", ""},
		{"user@localhost", ""},
		{"user name@example.com", ""},
		{"user@ex\tample.com", ""},
		{"user<script>@example.com", ""},
		{"user@" + strings.Repeat("a", 250) + ".com", ""},
		{strings.Repeat("a", 250) + "@example.com", ""},
	}

	for _, testCase := range cases {
		if got := utils.NormalizeEmail(testCase.raw); got != testCase.want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", testCase.raw, got, testCase.want)
		}
	}
}

func TestValidDomain(t *testing.T) {
	valid := []string{"example.com", "sub.example.com", "a-b.example.co.uk", "xn--80ak6aa92e.com"}
	invalid := []string{"", "a.b", "-example.com", "example-.com", "example..com", "example", strings.Repeat("a", 260) + ".com"}

	for _, domain := range valid {
		if !utils.ValidDomain(domain) {
			t.Errorf("ValidDomain(%q) = false, want true", domain)
		}
	}

	for _, domain := range invalid {
		if utils.ValidDomain(domain) {
			t.Errorf("ValidDomain(%q) = true, want false", domain)
		}
	}
}

func TestNormalizeDomain(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"  Example.COM  ", "example.com"},
		{"https://example.com", "example.com"},
		{"http://www.example.com", "example.com"},
		{"example.com.", "example.com"},
		{"www.example.com/", "example.com/"},
	}

	for _, testCase := range cases {
		if got := utils.NormalizeDomain(testCase.raw); got != testCase.want {
			t.Errorf("NormalizeDomain(%q) = %q, want %q", testCase.raw, got, testCase.want)
		}
	}
}
