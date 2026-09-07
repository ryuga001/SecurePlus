package utils_test

import (
	"slices"
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
