package scanner

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"dpdp-backend/internal/admin/services/datadiscovery"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/datadiscovery/provider"
	"dpdp-backend/internal/datadiscovery/strategy"
	"dpdp-backend/internal/db"
	dto "dpdp-backend/internal/delivery/dto/evaluation"
)

const (
	defaultQueueCapacity    = 100
	defaultChunkBytes       = 1 << 20
	defaultEvaluatorWorkers = 4
	defaultFileTimeout      = 30 * time.Minute
	defaultProgressInterval = 5 * time.Second
	defaultPollInterval     = 30 * time.Second
	defaultStaleAfter       = 2 * time.Minute
	finishTimeout           = 5 * time.Second
)

type Store interface {
	Claim(ctx context.Context) (db.DataDiscoveryScan, bool, error)
	Definition(ctx context.Context, scan db.DataDiscoveryScan) (datadiscovery.ScanDefinition, error)
	Start(ctx context.Context, scan db.DataDiscoveryScan, targets []string) error
	Progress(ctx context.Context, scanID int64, counters db.ScanCounters) error
	UpdateTarget(ctx context.Context, target db.DataDiscoveryScanTarget) error
	SaveFile(ctx context.Context, result db.DataDiscoveryFileResult) error
	Finish(ctx context.Context, scanID int64, status, errorCode string, counters db.ScanCounters) error
	FailStale(ctx context.Context, before time.Time) (int64, error)
	Created() <-chan struct{}
}

type ConnectFunc func(ctx context.Context, definition datadiscovery.ScanDefinition) (strategy.Source, error)

type Options struct {
	Store            Store
	Connect          ConnectFunc
	Processors       *Processors
	QueueCapacity    int
	ChunkBytes       int
	EvaluatorWorkers int
	FileTimeout      time.Duration
	ProgressInterval time.Duration
	PollInterval     time.Duration
	StaleAfter       time.Duration
	SpoolDir         string
}

type Scanner struct {
	opts Options
	wg   sync.WaitGroup
}

func New(opts Options) *Scanner {
	if opts.QueueCapacity <= 0 {
		opts.QueueCapacity = defaultQueueCapacity
	}
	if opts.ChunkBytes <= 0 {
		opts.ChunkBytes = defaultChunkBytes
	}
	if opts.EvaluatorWorkers <= 0 {
		opts.EvaluatorWorkers = defaultEvaluatorWorkers
	}
	if opts.FileTimeout <= 0 {
		opts.FileTimeout = defaultFileTimeout
	}
	if opts.ProgressInterval <= 0 {
		opts.ProgressInterval = defaultProgressInterval
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = defaultPollInterval
	}
	if opts.StaleAfter <= 0 {
		opts.StaleAfter = defaultStaleAfter
	}

	return &Scanner{opts: opts}
}

func RegistryConnector(registry *strategy.PolicyRegistry, client *provider.Client) ConnectFunc {
	return func(ctx context.Context, definition datadiscovery.ScanDefinition) (strategy.Source, error) {
		selected, ok := registry.For(definition.Policy.SourceType)
		if !ok {
			return nil, strategy.ErrSourceUnsupported
		}

		return selected.Connect(ctx, client, definition.Configuration.Config, definition.Credential)
	}
}

func (s *Scanner) Start(ctx context.Context) {
	if s.opts.SpoolDir != "" {
		if removed, err := RemoveStaleSpools(s.opts.SpoolDir); err != nil {
			slog.WarnContext(ctx, "scanner spool cleanup failed", "error", err)
		} else if removed > 0 {
			slog.InfoContext(ctx, "scanner removed stale spool files", "count", removed)
		}
	}

	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		s.Run(ctx)
	}()

	slog.InfoContext(ctx, "data discovery scanner started",
		"queue_capacity", s.opts.QueueCapacity,
		"chunk_bytes", s.opts.ChunkBytes,
		"evaluator_workers", s.opts.EvaluatorWorkers,
	)
}

func (s *Scanner) Wait() {
	s.wg.Wait()
}

func (s *Scanner) Run(ctx context.Context) {
	ticker := time.NewTicker(s.opts.PollInterval)
	defer ticker.Stop()

	for {
		s.failStale(ctx)
		s.drain(ctx)

		select {
		case <-ctx.Done():
			return
		case <-s.opts.Store.Created():
		case <-ticker.C:
		}
	}
}

func (s *Scanner) drain(ctx context.Context) {
	for ctx.Err() == nil {
		scan, ok, err := s.opts.Store.Claim(ctx)
		if err != nil {
			if ctx.Err() == nil {
				slog.ErrorContext(ctx, "scan claim failed", "error", err)
			}

			return
		}

		if !ok {
			return
		}

		s.Receive(ctx, scan)
	}
}

func (s *Scanner) failStale(ctx context.Context) {
	failed, err := s.opts.Store.FailStale(ctx, time.Now().Add(-s.opts.StaleAfter))
	if err != nil {
		if ctx.Err() == nil {
			slog.ErrorContext(ctx, "stale scan sweep failed", "error", err)
		}

		return
	}

	if failed > 0 {
		slog.WarnContext(ctx, "stale scans marked interrupted", "count", failed)
	}
}

func (s *Scanner) Receive(ctx context.Context, scan db.DataDiscoveryScan) string {
	started := time.Now()

	runCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	run, code := s.preProcess(runCtx, cancel, scan)
	if run == nil {
		if code == "" {
			return ""
		}

		s.finish(ctx, scan.ID, db.ScanStatusFailed, code, db.ScanCounters{})

		slog.WarnContext(ctx, "scan failed during initialization",
			"scan_id", scan.ID,
			"customer_id", scan.CustomerID,
			"policy_id", scan.PolicyID,
			"error_code", code,
			"duration_ms", time.Since(started).Milliseconds(),
		)

		return db.ScanStatusFailed
	}
	defer run.evaluator.Close()

	slog.InfoContext(ctx, "scan started",
		"scan_id", scan.ID,
		"customer_id", scan.CustomerID,
		"policy_id", scan.PolicyID,
		"targets", len(run.targets),
		"file_types", len(run.allowed),
	)

	s.process(runCtx, run)

	return s.postProcess(ctx, runCtx, run, started)
}

func (s *Scanner) preProcess(
	ctx context.Context,
	cancel context.CancelCauseFunc,
	scan db.DataDiscoveryScan,
) (*scanRun, string) {
	definition, err := s.opts.Store.Definition(ctx, scan)
	if err != nil {
		return nil, definitionCode(ctx, err)
	}

	targets := []string(definition.Policy.TargetList)
	if len(targets) == 0 {
		return nil, db.ScanErrorInitializationFailed
	}

	allowed, ignored := allowedExtensions(definition.Policy.FileTypes, s.opts.Processors)
	if len(ignored) > 0 {
		slog.InfoContext(ctx, "configured file types have no processor and will be skipped",
			"scan_id", scan.ID,
			"file_types", ignored,
		)
	}

	evaluator, err := NewEvaluator(ruleRecords(definition), s.opts.ChunkBytes, s.opts.EvaluatorWorkers)
	if err != nil {
		switch {
		case errors.Is(err, ErrNoRules):
			return nil, db.ScanErrorNoRules
		case errors.Is(err, ErrRulesTooLarge):
			return nil, db.ScanErrorRulesTooLarge
		}

		return nil, db.ScanErrorInitializationFailed
	}

	source, err := s.opts.Connect(ctx, definition)
	if err != nil {
		evaluator.Close()

		return nil, connectCode(ctx, err)
	}

	if err := s.opts.Store.Start(ctx, scan, targets); err != nil {
		evaluator.Close()

		if errors.Is(err, datadiscovery.ErrScanStateChanged) {
			slog.WarnContext(ctx, "scan changed state before start, skipping", "scan_id", scan.ID)

			return nil, ""
		}

		if ctx.Err() != nil {
			return nil, db.ScanErrorInterrupted
		}

		return nil, db.ScanErrorDatabaseUnavailable
	}

	run := &scanRun{
		scan:       scan,
		targets:    targets,
		allowed:    allowed,
		source:     source,
		evaluator:  evaluator,
		processors: s.opts.Processors,
		store:      s.opts.Store,
		opts:       s.opts,
		cancel:     cancel,
		started:    time.Now(),
	}

	run.counters.totalTargets.Store(int64(len(targets)))

	return run, ""
}

func (s *Scanner) process(ctx context.Context, run *scanRun) {
	stop := make(chan struct{})
	flushed := make(chan struct{})

	go func() {
		defer close(flushed)

		run.progress(ctx, stop)
	}()

	for position, target := range run.targets {
		if ctx.Err() != nil || run.fatal != "" {
			break
		}

		run.runTarget(ctx, position, target)
	}

	close(stop)
	<-flushed
}

func (s *Scanner) postProcess(ctx, runCtx context.Context, run *scanRun, started time.Time) string {
	if errors.Is(context.Cause(runCtx), datadiscovery.ErrScanStateChanged) {
		slog.WarnContext(ctx, "scan changed state outside the scanner, stopping without finalizing",
			"scan_id", run.scan.ID,
		)

		return ""
	}

	counters := run.counters.snapshot()
	status, code := terminalStatus(runCtx, run.fatal, counters)

	s.finish(ctx, run.scan.ID, status, code, counters)

	slog.InfoContext(ctx, "scan finished",
		"scan_id", run.scan.ID,
		"customer_id", run.scan.CustomerID,
		"policy_id", run.scan.PolicyID,
		"status", status,
		"error_code", code,
		"total_targets", counters.TotalTargets,
		"completed_targets", counters.CompletedTargets,
		"failed_targets", counters.FailedTargets,
		"files_discovered", counters.FilesDiscovered,
		"files_skipped", counters.FilesSkipped,
		"files_processed", counters.FilesProcessed,
		"files_succeeded", counters.FilesSucceeded,
		"files_failed", counters.FilesFailed,
		"findings_total", counters.FindingsTotal,
		"bytes_processed", counters.BytesProcessed,
		"duration_ms", time.Since(started).Milliseconds(),
	)

	return status
}

func (s *Scanner) finish(ctx context.Context, scanID int64, status, code string, counters db.ScanCounters) {
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), finishTimeout)
	defer cancel()

	if err := s.opts.Store.Finish(finishCtx, scanID, status, code, counters); err != nil {
		slog.ErrorContext(ctx, "scan finalization failed", "scan_id", scanID, "status", status, "error", err)
	}
}

func terminalStatus(ctx context.Context, fatal string, counters db.ScanCounters) (string, string) {
	switch {
	case fatal != "":
		return db.ScanStatusFailed, fatal
	case ctx.Err() != nil:
		return db.ScanStatusFailed, db.ScanErrorInterrupted
	case counters.TotalTargets > 0 && counters.FailedTargets == counters.TotalTargets:
		return db.ScanStatusFailed, db.ScanErrorAllTargetsFailed
	case counters.FailedTargets > 0 || counters.FilesFailed > 0:
		return db.ScanStatusPartial, ""
	}

	return db.ScanStatusCompleted, ""
}

func definitionCode(ctx context.Context, err error) string {
	switch {
	case ctx.Err() != nil:
		return db.ScanErrorInterrupted
	case errors.Is(err, utils.ErrDiscoveryPolicyNotFound):
		return db.ScanErrorPolicyNotFound
	case errors.Is(err, utils.ErrDiscoveryPolicyInactive):
		return db.ScanErrorPolicyInactive
	case errors.Is(err, utils.ErrUnknownConfiguration):
		return db.ScanErrorConfigurationInactive
	case errors.Is(err, utils.ErrSecretNeeded), errors.Is(err, utils.ErrCredentialUnavailable):
		return db.ScanErrorCredentialUnavailable
	}

	return db.ScanErrorDatabaseUnavailable
}

func connectCode(ctx context.Context, err error) string {
	if ctx.Err() != nil {
		return db.ScanErrorInterrupted
	}

	var failure *provider.Error

	switch {
	case errors.Is(err, strategy.ErrSourceUnsupported):
		return db.ScanErrorSourceUnsupported
	case errors.Is(err, utils.ErrCredentialUnavailable):
		return db.ScanErrorCredentialUnavailable
	case errors.As(err, &failure) && failure.Transient():
		return db.ScanErrorSourceUnavailable
	case errors.As(err, &failure):
		return db.ScanErrorSourceAuthFailed
	}

	return db.ScanErrorInitializationFailed
}

func ruleRecords(definition datadiscovery.ScanDefinition) []dto.RuleRecord {
	records := make([]dto.RuleRecord, 0, len(definition.Rules))

	for _, rule := range definition.Rules {
		records = append(records, dto.RuleRecord{
			PolicyID:   definition.Policy.ID,
			PolicyName: definition.Policy.Name,
			RuleID:     rule.ID,
			RuleName:   rule.RuleName,
			RuleType:   rule.Type,
			RuleValue:  rule.Value,
		})
	}

	return records
}

func allowedExtensions(configured []string, processors *Processors) (map[string]bool, []string) {
	allowed := make(map[string]bool)
	ignored := make([]string, 0)

	if len(configured) == 0 {
		for _, extension := range processors.Extensions() {
			allowed[extension] = true
		}

		return allowed, ignored
	}

	for _, extension := range configured {
		if _, ok := processors.For(extension); ok {
			allowed[extension] = true
		} else {
			ignored = append(ignored, extension)
		}
	}

	return allowed, ignored
}
