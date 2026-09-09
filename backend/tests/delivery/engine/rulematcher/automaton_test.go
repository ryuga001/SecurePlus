package rulematcher_test

import (
	"testing"

	"dpdp-backend/internal/delivery/engine/rulematcher"
)

func hits(t *testing.T, patterns []string, text string) map[int]int {
	t.Helper()

	counts := map[int]int{}

	for _, hit := range rulematcher.BuildAutomaton(patterns).Find(text) {
		counts[hit.Index]++
	}

	return counts
}

func TestAutomatonFindsSeveralPatternsInOnePass(t *testing.T) {
	counts := hits(t, []string{"card", "secret", "confidential"}, "a card and a secret and confidential data")

	if counts[0] != 1 || counts[1] != 1 || counts[2] != 1 {
		t.Fatalf("counts = %v, want one of each", counts)
	}
}

func TestAutomatonCountsRepeatedOccurrences(t *testing.T) {
	counts := hits(t, []string{"card"}, "card card card")

	if counts[0] != 3 {
		t.Fatalf("occurrences = %d, want 3", counts[0])
	}
}

func TestAutomatonReportsOverlappingPatterns(t *testing.T) {
	counts := hits(t, []string{"he", "she", "hers"}, "hers")

	if counts[0] != 1 {
		t.Fatalf("he = %d, want 1", counts[0])
	}
	if counts[2] != 1 {
		t.Fatalf("hers = %d, want 1", counts[2])
	}
	if counts[1] != 0 {
		t.Fatalf("she = %d, want 0", counts[1])
	}
}

func TestAutomatonMatchesPrefixOfAnotherPattern(t *testing.T) {
	counts := hits(t, []string{"pass", "password"}, "my password")

	if counts[0] != 1 || counts[1] != 1 {
		t.Fatalf("counts = %v, want both pass and password", counts)
	}
}

func TestAutomatonMatchesWholeText(t *testing.T) {
	counts := hits(t, []string{"secret"}, "secret")

	if counts[0] != 1 {
		t.Fatalf("occurrences = %d, want 1", counts[0])
	}
}

func TestAutomatonDoesNotMatchPartialText(t *testing.T) {
	counts := hits(t, []string{"confidential"}, "confide")

	if len(counts) != 0 {
		t.Fatalf("counts = %v, want no matches", counts)
	}
}

func TestAutomatonReportsOffsets(t *testing.T) {
	found := rulematcher.BuildAutomaton([]string{"card"}).Find("a card here")

	if len(found) != 1 {
		t.Fatalf("hits = %v", found)
	}
	if found[0].Start != 2 || found[0].End != 6 {
		t.Fatalf("offsets = %d..%d, want 2..6", found[0].Start, found[0].End)
	}
}

func TestAutomatonIgnoresEmptyPatterns(t *testing.T) {
	found := rulematcher.BuildAutomaton([]string{""}).Find("anything")

	if len(found) != 0 {
		t.Fatalf("hits = %v, want none", found)
	}
}
