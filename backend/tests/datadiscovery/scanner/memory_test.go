package scanner_test

import (
	"context"
	"io"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"dpdp-backend/internal/datadiscovery/scanner"
	dto "dpdp-backend/internal/delivery/dto/evaluation"
)

type generatedReader struct {
	line      []byte
	remaining int64
	offset    int
}

func (g *generatedReader) Read(p []byte) (int, error) {
	if g.remaining <= 0 {
		return 0, io.EOF
	}

	n := 0
	for n < len(p) && g.remaining > 0 {
		copied := copy(p[n:], g.line[g.offset:])
		if int64(copied) > g.remaining {
			copied = int(g.remaining)
		}

		n += copied
		g.remaining -= int64(copied)
		g.offset = (g.offset + copied) % len(g.line)
	}

	return n, nil
}

func TestStreamingMemoryIndependentOfFileSize(t *testing.T) {
	if testing.Short() {
		t.Skip("streams 128 MiB")
	}

	const size = 128 << 20

	evaluator, err := scanner.NewEvaluator([]dto.RuleRecord{
		keyword(1, "confidential"),
		regex(2, `[A-Z]{5}[0-9]{4}[A-Z]`),
	}, 1<<20, 2)
	if err != nil {
		t.Fatalf("evaluator: %v", err)
	}
	defer evaluator.Close()

	line := []byte(strings.Repeat("routine ledger entry with nothing sensitive in it, just filler text.\n", 14) + "ABCDE1234F confidential\n")
	source := &generatedReader{line: line, remaining: size}

	runtime.GC()

	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	var peak atomic.Uint64
	stop := make(chan struct{})
	sampled := make(chan struct{})

	go func() {
		defer close(sampled)

		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()

		for {
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)

			if stats.HeapInuse > peak.Load() {
				peak.Store(stats.HeapInuse)
			}

			select {
			case <-stop:
				return
			case <-ticker.C:
			}
		}
	}()

	scan := evaluator.Begin(context.Background())
	defer scan.Release()

	if err := (scanner.TextProcessor{}).Extract(context.Background(), source, scan); err != nil {
		t.Fatalf("extract: %v", err)
	}

	findings, err := scan.Finish()
	if err != nil {
		t.Fatalf("finish: %v", err)
	}

	close(stop)
	<-sampled

	lines := int64(size) / int64(len(line))
	if findings.Extracted != size || countFor(findings, 1) < lines || countFor(findings, 2) < lines {
		t.Fatalf("findings = %+v lines = %d", findings, lines)
	}

	growth := int64(peak.Load()) - int64(baseline.HeapInuse)
	if growth > 64<<20 {
		t.Fatalf("heap grew by %d MiB while streaming %d MiB", growth>>20, size>>20)
	}

	t.Logf("streamed %d MiB, heap growth %d MiB", size>>20, growth>>20)
}
