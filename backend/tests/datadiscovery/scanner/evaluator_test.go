package scanner_test

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"dpdp-backend/internal/datadiscovery/scanner"
	"dpdp-backend/internal/db"
	dto "dpdp-backend/internal/delivery/dto/evaluation"
)

func keyword(id int, value string) dto.RuleRecord {
	return dto.RuleRecord{RuleID: id, RuleName: "keyword-" + value, RuleType: db.RuleTypeKeyword, RuleValue: value}
}

func regex(id int, value string) dto.RuleRecord {
	return dto.RuleRecord{RuleID: id, RuleName: "regex", RuleType: db.RuleTypeRegex, RuleValue: value}
}

func evaluate(t *testing.T, rules []dto.RuleRecord, text string, chunk, workers, piece int) scanner.Findings {
	t.Helper()

	evaluator, err := scanner.NewEvaluator(rules, chunk, workers)
	if err != nil {
		t.Fatalf("evaluator: %v", err)
	}
	defer evaluator.Close()

	scan := evaluator.Begin(context.Background())
	defer scan.Release()

	data := []byte(text)
	for len(data) > 0 {
		n := min(piece, len(data))
		if _, err := scan.Write(data[:n]); err != nil {
			t.Fatalf("write: %v", err)
		}

		data = data[n:]
	}

	findings, err := scan.Finish()
	if err != nil {
		t.Fatalf("finish: %v", err)
	}

	return findings
}

func countFor(findings scanner.Findings, ruleID int) int64 {
	for _, item := range findings.Items {
		if item.RuleID == ruleID {
			return item.Count
		}
	}

	return 0
}

func TestEvaluatorKeywordAndRegexMatches(t *testing.T) {
	rules := []dto.RuleRecord{
		keyword(1, "Confidential"),
		regex(2, `[A-Z]{5}[0-9]{4}[A-Z]`),
	}

	findings := evaluate(t, rules, "This CONFIDENTIAL note has PAN ABCDE1234F and ABCDE1234G. confidential", 1<<20, 2, 1<<20)

	if countFor(findings, 1) != 2 {
		t.Fatalf("keyword count = %d", countFor(findings, 1))
	}
	if countFor(findings, 2) != 2 {
		t.Fatalf("regex count = %d", countFor(findings, 2))
	}
	if findings.Total != 4 {
		t.Fatalf("total = %d", findings.Total)
	}
}

func TestEvaluatorMatchesAcrossWindowBoundary(t *testing.T) {
	rules := []dto.RuleRecord{
		keyword(1, "secret-project"),
		regex(2, `[a-z]+@example\.com`),
	}

	window := 4096 + scanner.RegexOverlap
	text := strings.Repeat("x ", (window-10)/2) + " customer@example.com secret-project " + strings.Repeat("y ", 5000)

	for _, piece := range []int{1, 7, 4096, len(text)} {
		findings := evaluate(t, rules, text, 4096, 3, piece)

		if countFor(findings, 1) != 1 {
			t.Fatalf("piece %d keyword count = %d", piece, countFor(findings, 1))
		}
		if countFor(findings, 2) != 1 {
			t.Fatalf("piece %d regex count = %d", piece, countFor(findings, 2))
		}
	}
}

func TestEvaluatorRegexStraddlingCommitIsCountedOnce(t *testing.T) {
	rules := []dto.RuleRecord{regex(1, `\d{12}`)}

	window := 4096 + scanner.RegexOverlap
	for shift := -20; shift <= 20; shift++ {
		prefix := strings.Repeat("a", 4096+shift)
		text := prefix + " 123456789012 " + strings.Repeat("b", window)

		findings := evaluate(t, rules, text, 4096, 1, 1000)
		if countFor(findings, 1) != 1 {
			t.Fatalf("shift %d count = %d", shift, countFor(findings, 1))
		}
	}
}

func TestEvaluatorKeywordAcrossNormalizationPiece(t *testing.T) {
	rules := []dto.RuleRecord{keyword(1, "aadhaar")}

	text := strings.Repeat("z", 65536-3) + "AADHAAR" + strings.Repeat("z", 100)
	findings := evaluate(t, rules, text, 1<<20, 1, len(text))

	if countFor(findings, 1) != 1 {
		t.Fatalf("count = %d", countFor(findings, 1))
	}
}

func TestEvaluatorCombiningSequenceSplitAcrossWindows(t *testing.T) {
	rules := []dto.RuleRecord{keyword(1, "caf\u00e9")}

	window := 4096 + scanner.RegexOverlap
	text := strings.Repeat("q", window-4) + "cafe\u0301 " + strings.Repeat("q", 100)

	findings := evaluate(t, rules, text, 4096, 1, 3)
	if countFor(findings, 1) != 1 {
		t.Fatalf("count = %d", countFor(findings, 1))
	}
}

func TestEvaluatorCountsStayExactWhenSamplesCapped(t *testing.T) {
	rules := []dto.RuleRecord{regex(1, `ID-\d{3}`)}

	text := strings.Repeat("ID-123 ", 25000)
	findings := evaluate(t, rules, text, 64<<10, 2, 4096)

	if countFor(findings, 1) != 25000 {
		t.Fatalf("count = %d", countFor(findings, 1))
	}

	item := findings.Items[0]
	if len(item.Offsets) != scanner.MaxSampleOffsets {
		t.Fatalf("offsets = %v", item.Offsets)
	}

	if !reflect.DeepEqual(item.Offsets, []int64{0, 7, 14, 21, 28}) {
		t.Fatalf("offsets = %v", item.Offsets)
	}
}

func TestEvaluatorResultsIndependentOfWorkers(t *testing.T) {
	rules := []dto.RuleRecord{
		keyword(1, "passport"),
		keyword(2, "salary"),
		regex(3, `\b\d{4} \d{4} \d{4}\b`),
		regex(4, `[A-Z]{5}\d{4}[A-Z]`),
		regex(5, `\+91[ -]?\d{10}`),
		regex(6, `(?i)ifsc[: ]+[a-z]{4}0[a-z0-9]{6}`),
	}

	var builder strings.Builder
	for index := range 4000 {
		switch index % 5 {
		case 0:
			builder.WriteString("Passport issued; salary slip attached. ")
		case 1:
			builder.WriteString("Aadhaar 1234 5678 9012 recorded. ")
		case 2:
			builder.WriteString("PAN ABCDE1234F with +91 9876543210. ")
		case 3:
			builder.WriteString("IFSC: HDFC0001234 branch. ")
		default:
			builder.WriteString("nothing to see here. ")
		}
	}

	text := builder.String()
	single := evaluate(t, rules, text, 64<<10, 1, 5000)
	parallel := evaluate(t, rules, text, 64<<10, 4, 5000)

	if !reflect.DeepEqual(single, parallel) {
		t.Fatalf("results differ\nsingle:   %+v\nparallel: %+v", single, parallel)
	}

	if single.Total != 4800 {
		t.Fatalf("total = %d", single.Total)
	}
}

func settledGoroutines() int {
	previous := runtime.NumGoroutine()

	for range 100 {
		time.Sleep(10 * time.Millisecond)

		current := runtime.NumGoroutine()
		if current == previous {
			return current
		}

		previous = current
	}

	return previous
}

func TestEvaluatorUsesFixedWorkerPool(t *testing.T) {
	before := settledGoroutines()

	evaluator, err := scanner.NewEvaluator([]dto.RuleRecord{keyword(1, "a"), regex(2, `b+`), regex(3, `c+`)}, 64<<10, 4)
	if err != nil {
		t.Fatalf("evaluator: %v", err)
	}

	if started := runtime.NumGoroutine() - before; started != 4 {
		t.Fatalf("pool goroutines = %d, want 4", started)
	}

	scan := evaluator.Begin(context.Background())
	for range 8 {
		_, _ = scan.Write([]byte(strings.Repeat("abc ", 16<<10)))
	}

	if peak := runtime.NumGoroutine() - before; peak != 4 {
		t.Fatalf("goroutines during evaluation = %d, want 4", peak)
	}

	_, _ = scan.Finish()
	scan.Release()
	evaluator.Close()

	deadline := time.Now().Add(time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if runtime.NumGoroutine() > before {
		t.Fatalf("goroutines leaked: before %d after %d", before, runtime.NumGoroutine())
	}
}

func TestEvaluatorRejectsOversizedRules(t *testing.T) {
	rules := make([]dto.RuleRecord, 0, 70)
	for index := range 70 {
		rules = append(rules, keyword(index+1, strings.Repeat(string(rune('a'+index%26)), 1000)))
	}

	if _, err := scanner.NewEvaluator(rules, 1<<20, 1); !errors.Is(err, scanner.ErrRulesTooLarge) {
		t.Fatalf("error = %v", err)
	}
}

func TestEvaluatorRequiresRules(t *testing.T) {
	if _, err := scanner.NewEvaluator(nil, 1<<20, 1); !errors.Is(err, scanner.ErrNoRules) {
		t.Fatalf("error = %v", err)
	}

	if _, err := scanner.NewEvaluator([]dto.RuleRecord{regex(1, `(`)}, 1<<20, 1); !errors.Is(err, scanner.ErrNoRules) {
		t.Fatalf("invalid-only error = %v", err)
	}
}

func TestEvaluatorStopsOnCancellation(t *testing.T) {
	evaluator, err := scanner.NewEvaluator([]dto.RuleRecord{keyword(1, "x")}, 64<<10, 1)
	if err != nil {
		t.Fatalf("evaluator: %v", err)
	}
	defer evaluator.Close()

	ctx, cancel := context.WithCancel(context.Background())
	scan := evaluator.Begin(ctx)
	defer scan.Release()

	cancel()

	if _, err := scan.Write(make([]byte, 256<<10)); !errors.Is(err, context.Canceled) {
		t.Fatalf("write error = %v", err)
	}
}

func TestEvaluatorEmptyInput(t *testing.T) {
	findings := evaluate(t, []dto.RuleRecord{keyword(1, "x"), regex(2, `y*`)}, "", 64<<10, 2, 1)

	if findings.Total != 0 || len(findings.Items) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
}
