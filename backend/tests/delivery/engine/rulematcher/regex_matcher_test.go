package rulematcher_test

import (
	"strings"
	"testing"

	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	"dpdp-backend/internal/delivery/engine/rulematcher"
	"dpdp-backend/internal/delivery/utils"
)

func regexRule(id int, name, pattern string) dto.RuleRecord {
	return dto.RuleRecord{
		PolicyID:   1,
		PolicyName: "Data Loss",
		Action:     utils.ActionBlock,
		RuleID:     id,
		RuleName:   name,
		RuleType:   utils.MatcherRegex,
		RuleValue:  pattern,
	}
}

func regexMatch(t *testing.T, pattern string, parts ...dto.ContentPart) []dto.RuleMatch {
	t.Helper()

	set := compiled(t, regexRule(11, "Cards", pattern))

	return rulematcher.NewRegexMatcher().Match(dto.MatchInput{Parts: parts}, set.Rules)
}

func body(text string) dto.ContentPart {
	return dto.ContentPart{Location: utils.LocationBody, Text: text}
}

func subject(text string) dto.ContentPart {
	return dto.ContentPart{Location: utils.LocationSubject, Text: text}
}

func TestRegexMatcherMatchesAPattern(t *testing.T) {
	matches := regexMatch(t, `\d{16}`, body("card 4111111111111111 on file"))

	if len(matches) != 1 {
		t.Fatalf("matches = %v", matches)
	}

	match := matches[0]

	if match.RuleID != 11 || match.RuleType != utils.MatcherRegex {
		t.Fatalf("match = %+v", match)
	}
	if match.Occurrences != 1 {
		t.Fatalf("occurrences = %d, want 1", match.Occurrences)
	}
	if match.PolicyID != 1 || match.Action != utils.ActionBlock {
		t.Fatalf("match carries the wrong policy: %+v", match)
	}
}

func TestRegexMatcherReturnsNothingWithoutAMatch(t *testing.T) {
	if matches := regexMatch(t, `\d{16}`, body("no numbers here")); len(matches) != 0 {
		t.Fatalf("matches = %v, want none", matches)
	}
}

func TestRegexMatcherCountsEveryOccurrence(t *testing.T) {
	matches := regexMatch(t, `\d{4}`, body("1111 2222 3333"))

	if matches[0].Occurrences != 3 {
		t.Fatalf("occurrences = %d, want 3", matches[0].Occurrences)
	}
}

func TestRegexMatcherRecordsEveryLocation(t *testing.T) {
	matches := regexMatch(t, `\d{4}`, subject("1111"), body("2222"))

	locations := matches[0].Locations

	if len(locations) != 2 || locations[0] != utils.LocationSubject || locations[1] != utils.LocationBody {
		t.Fatalf("locations = %v, want subject then body", locations)
	}
	if matches[0].Occurrences != 2 {
		t.Fatalf("occurrences = %d, want 2 across both parts", matches[0].Occurrences)
	}
}

func TestRegexMatcherIsCaseSensitiveByDefault(t *testing.T) {
	if matches := regexMatch(t, `SECRET`, body("secret")); len(matches) != 0 {
		t.Fatalf("matches = %v, a bare pattern must not fold case", matches)
	}
}

func TestRegexMatcherHonoursTheInlineCaseFlag(t *testing.T) {
	if matches := regexMatch(t, `(?i)secret`, body("SECRET")); len(matches) != 1 {
		t.Fatalf("matches = %v, want (?i) to fold case", matches)
	}
}

func TestRegexMatcherIgnoresZeroWidthMatches(t *testing.T) {
	if matches := regexMatch(t, `a*`, body("xxx")); len(matches) != 0 {
		t.Fatalf("matches = %v, a pattern matching nothing is not evidence", matches)
	}
}

func TestRegexMatcherCountsZeroWidthOnlyWhenItConsumes(t *testing.T) {
	matches := regexMatch(t, `a*`, body("xaax"))

	if len(matches) != 1 || matches[0].Occurrences != 1 {
		t.Fatalf("matches = %v, want the one consuming match", matches)
	}
}

func TestRegexMatcherCapsOccurrenceCount(t *testing.T) {
	matcher := rulematcher.NewRegexMatcherWithLimit(10)
	set := compiled(t, regexRule(11, "Digits", `\d`))

	matches := matcher.Match(dto.MatchInput{
		Parts: []dto.ContentPart{body(strings.Repeat("1", 500))},
	}, set.Rules)

	if matches[0].Occurrences != 10 {
		t.Fatalf("occurrences = %d, want the count capped at 10", matches[0].Occurrences)
	}
}

func TestRegexMatcherSkipsEmptyParts(t *testing.T) {
	if matches := regexMatch(t, `.+`, body(""), subject("")); len(matches) != 0 {
		t.Fatalf("matches = %v, want none for empty content", matches)
	}
}

func TestRegexMatcherReportsEachRuleSeparately(t *testing.T) {
	set := compiled(t,
		regexRule(11, "Cards", `\d{16}`),
		regexRule(12, "Codes", `[A-Z]{3}-\d{3}`),
	)

	matches := rulematcher.NewRegexMatcher().Match(dto.MatchInput{
		Parts: []dto.ContentPart{body("4111111111111111 and ABC-123")},
	}, set.Rules)

	if len(matches) != 2 {
		t.Fatalf("matches = %v, want one per rule", matches)
	}

	seen := map[int]bool{}
	for _, match := range matches {
		seen[match.RuleID] = true
	}

	if !seen[11] || !seen[12] {
		t.Fatalf("matched rules = %v", seen)
	}
}

func TestRegexMatcherReturnsNothingWhenNoRegexRulesCompiled(t *testing.T) {
	set := compiled(t, keywordRule(1, "Cards", "card"))

	if matches := rulematcher.NewRegexMatcher().Match(dto.MatchInput{
		Parts: []dto.ContentPart{body("card")},
	}, set.Rules); len(matches) != 0 {
		t.Fatalf("matches = %v, keyword rules are not the regex matcher's business", matches)
	}
}

func TestRegexMatcherNeverReportsMatchedText(t *testing.T) {
	const secret = "4111111111111111"

	matches := regexMatch(t, `\d{16}`, body("card "+secret))

	for _, field := range []string{
		matches[0].RuleName,
		matches[0].PolicyName,
		matches[0].ConfiguredValue,
		strings.Join(matches[0].Locations, ","),
	} {
		if strings.Contains(field, secret) {
			t.Fatalf("matched text leaked into the evidence: %q", field)
		}
	}
}
