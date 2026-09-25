package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"dpdp-backend/internal/admin/services/datadiscovery"
	"dpdp-backend/internal/datadiscovery/provider"
	"dpdp-backend/internal/datadiscovery/strategy"
	"dpdp-backend/internal/db"
)

const (
	persistAttempts  = 3
	persistBaseDelay = 200 * time.Millisecond

	maxFileKeyBytes   = 2048
	maxFileNameRunes  = 1024
	maxMIMETypeRunes  = 255
	maxErrorCodeBytes = 50
)

type counters struct {
	totalTargets     atomic.Int64
	completedTargets atomic.Int64
	failedTargets    atomic.Int64
	discovered       atomic.Int64
	supported        atomic.Int64
	skipped          atomic.Int64
	processed        atomic.Int64
	succeeded        atomic.Int64
	failed           atomic.Int64
	findings         atomic.Int64
	bytes            atomic.Int64

	openNanos    atomic.Int64
	extractNanos atomic.Int64
	persistNanos atomic.Int64
}

func (c *counters) snapshot() db.ScanCounters {
	return db.ScanCounters{
		TotalTargets:     int(c.totalTargets.Load()),
		CompletedTargets: int(c.completedTargets.Load()),
		FailedTargets:    int(c.failedTargets.Load()),
		FilesDiscovered:  c.discovered.Load(),
		FilesSupported:   c.supported.Load(),
		FilesSkipped:     c.skipped.Load(),
		FilesProcessed:   c.processed.Load(),
		FilesSucceeded:   c.succeeded.Load(),
		FilesFailed:      c.failed.Load(),
		FindingsTotal:    c.findings.Load(),
		BytesProcessed:   c.bytes.Load(),
	}
}

type targetTally struct {
	discovered int64
	skipped    int64
	succeeded  int64
	failed     int64
	findings   int64
}

type scanRun struct {
	scan       db.DataDiscoveryScan
	targets    []string
	allowed    map[string]bool
	source     strategy.Source
	evaluator  *Evaluator
	processors *Processors
	store      Store
	opts       Options
	cancel     context.CancelCauseFunc
	started    time.Time

	counters counters
	queue    atomic.Pointer[FileQueue]
	fatal    string
}

func (r *scanRun) runTarget(ctx context.Context, position int, target string) {
	started := time.Now()

	row := db.DataDiscoveryScanTarget{
		ScanID:     r.scan.ID,
		Position:   position,
		CustomerID: r.scan.CustomerID,
		Target:     target,
		Status:     db.TargetStatusRunning,
		StartedAt:  &started,
	}

	r.saveTarget(ctx, row)

	slog.InfoContext(ctx, "scan target started", "scan_id", r.scan.ID, "target_position", position)

	targetCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	queue := NewFileQueue(r.opts.QueueCapacity)
	r.queue.Store(queue)
	defer r.queue.Store(nil)

	tally := &targetTally{}
	consumed := make(chan struct{})

	go func() {
		defer close(consumed)

		r.consume(targetCtx, cancel, queue, position, tally)
	}()

	listErr := r.produce(targetCtx, queue, target, tally)

	queue.Close()
	<-consumed

	finished := time.Now()
	row.FinishedAt = &finished
	row.FilesDiscovered = tally.discovered
	row.FilesSkipped = tally.skipped
	row.FilesSucceeded = tally.succeeded
	row.FilesFailed = tally.failed
	row.FindingsTotal = tally.findings

	switch {
	case r.fatal != "":
		row.Status = db.TargetStatusFailed
		row.ErrorCode = stringPointer(r.fatal)
		r.counters.failedTargets.Add(1)
	case ctx.Err() != nil:
		row.Status = db.TargetStatusFailed
		row.ErrorCode = stringPointer(db.ScanErrorInterrupted)
		r.counters.failedTargets.Add(1)
	case listErr != nil:
		row.Status = db.TargetStatusFailed
		row.ErrorCode = stringPointer(providerCode("LISTING", listErr))
		r.counters.failedTargets.Add(1)
	case tally.failed > 0:
		row.Status = db.TargetStatusPartial
		r.counters.completedTargets.Add(1)
	default:
		row.Status = db.TargetStatusCompleted
		r.counters.completedTargets.Add(1)
	}

	r.saveTarget(ctx, row)
	r.flush(ctx)

	slog.InfoContext(ctx, "scan target finished",
		"scan_id", r.scan.ID,
		"target_position", position,
		"status", row.Status,
		"error_code", derefString(row.ErrorCode),
		"files_discovered", tally.discovered,
		"files_skipped", tally.skipped,
		"files_succeeded", tally.succeeded,
		"files_failed", tally.failed,
		"findings_total", tally.findings,
		"duration_ms", finished.Sub(started).Milliseconds(),
	)
}

func (r *scanRun) produce(ctx context.Context, queue *FileQueue, target string, tally *targetTally) error {
	return r.source.List(ctx, target, func(file provider.File) error {
		r.counters.discovered.Add(1)
		tally.discovered++

		extension := ResolveExtension(file.Name, file.MIMEType)
		if !r.allowed[extension] {
			r.counters.skipped.Add(1)
			tally.skipped++

			return nil
		}

		r.counters.supported.Add(1)
		r.counters.processed.Add(1)

		if err := queue.Put(ctx, FileJob{File: file, Extension: extension}); err != nil {
			r.counters.processed.Add(-1)

			return err
		}

		return nil
	})
}

func (r *scanRun) consume(
	ctx context.Context,
	cancel context.CancelCauseFunc,
	queue *FileQueue,
	position int,
	tally *targetTally,
) {
	for job := range queue.Jobs() {
		if ctx.Err() != nil {
			r.counters.failed.Add(1)
			tally.failed++
			queue.Done()

			continue
		}

		if err := r.processFile(ctx, job, position, tally); err != nil {
			r.fatal = db.ScanErrorDatabaseUnavailable
			cancel(err)
		}

		queue.Done()
	}
}

func (r *scanRun) processFile(ctx context.Context, job FileJob, position int, tally *targetTally) error {
	started := time.Now()

	fileCtx, cancel := context.WithTimeout(ctx, r.opts.FileTimeout)
	findings, read, code := r.extract(fileCtx, job)
	cancel()

	r.counters.bytes.Add(read)

	if ctx.Err() != nil {
		r.counters.failed.Add(1)
		tally.failed++

		return nil
	}

	result := fileResult(r.scan, position, job, started)
	result.BytesProcessed = read

	if code == "" {
		result.Status = db.FileStatusSucceeded
		result.FindingsTotal = findings.Total
		result.Findings = findings.Items
	} else {
		result.Status = db.FileStatusFailed
		result.ErrorCode = stringPointer(code)
	}

	result.DurationMS = time.Since(started).Milliseconds()

	persistStarted := time.Now()
	err := r.persist(ctx, result)
	r.counters.persistNanos.Add(int64(time.Since(persistStarted)))

	if err != nil {
		r.counters.failed.Add(1)
		tally.failed++

		if ctx.Err() != nil {
			return nil
		}

		slog.ErrorContext(ctx, "file result not persisted",
			"scan_id", r.scan.ID,
			"target_position", position,
			"file_ref", fileRef(job.Key),
			"error", err,
		)

		return err
	}

	if code != "" {
		r.counters.failed.Add(1)
		tally.failed++

		slog.WarnContext(ctx, "file processing failed",
			"scan_id", r.scan.ID,
			"target_position", position,
			"file_ref", fileRef(job.Key),
			"extension", job.Extension,
			"mime_type", job.MIMEType,
			"size_bytes", job.Size,
			"error_code", code,
			"duration_ms", result.DurationMS,
		)

		return nil
	}

	r.counters.succeeded.Add(1)
	r.counters.findings.Add(findings.Total)
	tally.succeeded++
	tally.findings += findings.Total

	return nil
}

func (r *scanRun) extract(ctx context.Context, job FileJob) (Findings, int64, string) {
	processor, ok := r.processors.For(job.Extension)
	if !ok {
		return Findings{}, 0, db.FileErrorParseFailed
	}

	openStarted := time.Now()
	body, err := r.source.Open(ctx, job.File)
	r.counters.openNanos.Add(int64(time.Since(openStarted)))

	if err != nil {
		return Findings{}, 0, fileCode(ctx, err, nil)
	}
	defer body.Close()

	counted := &countingReader{r: body}

	scan := r.evaluator.Begin(ctx)
	defer scan.Release()

	extractStarted := time.Now()
	defer func() { r.counters.extractNanos.Add(int64(time.Since(extractStarted))) }()

	if err := runProcessor(ctx, processor, counted, scan); err != nil {
		return Findings{}, counted.n, fileCode(ctx, err, counted.err)
	}

	findings, err := scan.Finish()
	if err != nil {
		return Findings{}, counted.n, fileCode(ctx, err, counted.err)
	}

	return findings, counted.n, ""
}

func (r *scanRun) persist(ctx context.Context, result db.DataDiscoveryFileResult) error {
	var err error

	for attempt := range persistAttempts {
		if err = r.store.SaveFile(ctx, result); err == nil {
			return nil
		}

		if ctx.Err() != nil || attempt == persistAttempts-1 {
			break
		}

		if waitErr := sleep(ctx, persistBaseDelay<<attempt); waitErr != nil {
			return waitErr
		}
	}

	return err
}

func (r *scanRun) saveTarget(ctx context.Context, row db.DataDiscoveryScanTarget) {
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), finishTimeout)
	defer cancel()

	if err := r.store.UpdateTarget(saveCtx, row); err != nil {
		slog.ErrorContext(ctx, "scan target update failed",
			"scan_id", r.scan.ID,
			"target_position", row.Position,
			"error", err,
		)
	}
}

func (r *scanRun) progress(ctx context.Context, stop <-chan struct{}) {
	ticker := time.NewTicker(r.opts.ProgressInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		r.flush(ctx)
		r.report(ctx)
	}
}

func (r *scanRun) flush(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	err := r.store.Progress(ctx, r.scan.ID, r.counters.snapshot())
	if err == nil || ctx.Err() != nil {
		return
	}

	if errors.Is(err, datadiscovery.ErrScanStateChanged) {
		r.cancel(err)

		return
	}

	slog.ErrorContext(ctx, "scan progress update failed", "scan_id", r.scan.ID, "error", err)
}

func (r *scanRun) report(ctx context.Context) {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)

	counters := r.counters.snapshot()
	elapsed := max(time.Since(r.started).Seconds(), 1)

	inFlight := 0
	if queue := r.queue.Load(); queue != nil {
		inFlight = queue.InFlight()
	}

	slog.InfoContext(ctx, "scan progress",
		"scan_id", r.scan.ID,
		"completed_targets", counters.CompletedTargets,
		"failed_targets", counters.FailedTargets,
		"total_targets", counters.TotalTargets,
		"files_discovered", counters.FilesDiscovered,
		"files_skipped", counters.FilesSkipped,
		"files_processed", counters.FilesProcessed,
		"files_succeeded", counters.FilesSucceeded,
		"files_failed", counters.FilesFailed,
		"findings_total", counters.FindingsTotal,
		"bytes_processed", counters.BytesProcessed,
		"queue_in_flight", inFlight,
		"queue_capacity", r.opts.QueueCapacity,
		"files_per_second", float64(counters.FilesSucceeded+counters.FilesFailed)/elapsed,
		"mb_per_second", float64(counters.BytesProcessed)/elapsed/(1<<20),
		"open_ms", time.Duration(r.counters.openNanos.Load()).Milliseconds(),
		"extract_ms", time.Duration(r.counters.extractNanos.Load()).Milliseconds(),
		"persist_ms", time.Duration(r.counters.persistNanos.Load()).Milliseconds(),
		"heap_inuse_mb", memory.HeapInuse>>20,
	)
}

func runProcessor(ctx context.Context, processor Processor, r io.Reader, w io.Writer) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%w: %v", ErrParse, recovered)
		}
	}()

	return processor.Extract(ctx, r, w)
}

func fileCode(ctx context.Context, err, readErr error) string {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return db.FileErrorTimeout
	case ctx.Err() != nil:
		return db.FileErrorCancelled
	case errors.Is(err, ErrFileTooLarge):
		return db.FileErrorTooLarge
	case errors.Is(err, ErrLimitExceeded):
		return db.FileErrorLimitExceeded
	case errors.Is(err, ErrEvaluation):
		return db.FileErrorEvaluationFailed
	case errors.Is(err, ErrParse):
		return db.FileErrorParseFailed
	case readErr != nil && errors.Is(err, readErr):
		return providerCode("FETCH", err)
	}

	var failure *provider.Error
	if errors.As(err, &failure) {
		return providerCode("FETCH", err)
	}

	return db.FileErrorParseFailed
}

func providerCode(prefix string, err error) string {
	var failure *provider.Error
	if errors.As(err, &failure) && failure.Reason != "" {
		return truncateBytes(prefix+"_"+failure.Reason, maxErrorCodeBytes)
	}

	return prefix + "_FAILED"
}

func fileResult(scan db.DataDiscoveryScan, position int, job FileJob, started time.Time) db.DataDiscoveryFileResult {
	result := db.DataDiscoveryFileResult{
		ScanID:         scan.ID,
		CustomerID:     scan.CustomerID,
		TargetPosition: position,
		FileKey:        fileKey(job.Key),
		FileName:       truncateRunes(cleanText(job.Name), maxFileNameRunes),
		Extension:      job.Extension,
		MIMEType:       truncateRunes(cleanText(job.MIMEType), maxMIMETypeRunes),
		SizeBytes:      job.Size,
		ProcessedAt:    started,
	}

	if !job.ModifiedAt.IsZero() {
		modified := job.ModifiedAt
		result.ModifiedAt = &modified
	}

	return result
}

func fileKey(key string) string {
	key = cleanText(key)
	if len(key) <= maxFileKeyBytes {
		return key
	}

	digest := sha256.Sum256([]byte(key))

	return truncateBytes(key, maxFileKeyBytes-33) + "#" + hex.EncodeToString(digest[:16])
}

func fileRef(key string) string {
	digest := sha256.Sum256([]byte(key))

	return hex.EncodeToString(digest[:6])
}

func cleanText(value string) string {
	return strings.ReplaceAll(strings.ToValidUTF8(value, "�"), "\x00", "")
}

func truncateRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}

	runes := []rune(value)

	return string(runes[:limit])
}

func truncateBytes(value string, limit int) string {
	if len(value) <= limit {
		return value
	}

	cut := limit
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}

	return value[:cut]
}

func stringPointer(value string) *string {
	return &value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func sleep(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type countingReader struct {
	r   io.Reader
	n   int64
	err error
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)

	if err != nil && !errors.Is(err, io.EOF) {
		c.err = err
	}

	return n, err
}
