package scanner_test

import (
	"context"
	"errors"
	"io"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"dpdp-backend/internal/admin/services/datadiscovery"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/datadiscovery/provider"
	"dpdp-backend/internal/datadiscovery/scanner"
	"dpdp-backend/internal/datadiscovery/strategy"
	"dpdp-backend/internal/db"
)

type fakeStore struct {
	mu sync.Mutex

	definition    datadiscovery.ScanDefinition
	definitionErr error
	startErr      error
	progressErr   error
	saveErr       error
	claims        []db.DataDiscoveryScan

	started    bool
	saves      int
	saved      []db.DataDiscoveryFileResult
	targets    map[int]db.DataDiscoveryScanTarget
	finished   bool
	status     string
	code       string
	counters   db.ScanCounters
	staleSweep int
	events     *eventLog
	created    chan struct{}
}

func newStore(events *eventLog, targets []string, fileTypes []string, rules []db.Rule) *fakeStore {
	return &fakeStore{
		definition: datadiscovery.ScanDefinition{
			Policy: db.DataDiscoveryPolicy{
				ID:         7,
				Name:       "Blob PII",
				SourceType: db.SourceTypeAzureBlob,
				TargetList: db.StringList(targets),
				FileTypes:  db.StringList(fileTypes),
				Status:     db.DiscoveryStatusActive,
			},
			Rules: rules,
		},
		targets: map[int]db.DataDiscoveryScanTarget{},
		events:  events,
		created: make(chan struct{}, 1),
	}
}

func (s *fakeStore) Claim(context.Context) (db.DataDiscoveryScan, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.claims) == 0 {
		return db.DataDiscoveryScan{}, false, nil
	}

	scan := s.claims[0]
	s.claims = s.claims[1:]

	return scan, true, nil
}

func (s *fakeStore) Definition(_ context.Context, scan db.DataDiscoveryScan) (datadiscovery.ScanDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	definition := s.definition
	definition.Scan = scan

	return definition, s.definitionErr
}

func (s *fakeStore) Start(context.Context, db.DataDiscoveryScan, []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.started = s.startErr == nil

	return s.startErr
}

func (s *fakeStore) Progress(context.Context, int64, db.ScanCounters) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.progressErr
}

func (s *fakeStore) UpdateTarget(_ context.Context, target db.DataDiscoveryScanTarget) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.targets[target.Position] = target
	s.events.add("target:" + target.Target + ":" + target.Status)

	return nil
}

func (s *fakeStore) SaveFile(_ context.Context, result db.DataDiscoveryFileResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.saves++

	if s.saveErr != nil {
		return s.saveErr
	}

	s.saved = append(s.saved, result)
	s.events.add("save:" + result.FileKey)

	return nil
}

func (s *fakeStore) Finish(_ context.Context, _ int64, status, code string, counters db.ScanCounters) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.finished = true
	s.status = status
	s.code = code
	s.counters = counters

	return nil
}

func (s *fakeStore) FailStale(context.Context, time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.staleSweep++

	return 0, nil
}

func (s *fakeStore) Created() <-chan struct{} {
	return s.created
}

func (s *fakeStore) result(key string) (db.DataDiscoveryFileResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, result := range s.saved {
		if result.FileKey == key {
			return result, true
		}
	}

	return db.DataDiscoveryFileResult{}, false
}

type eventLog struct {
	mu     sync.Mutex
	events []string
}

func (l *eventLog) add(event string) {
	l.mu.Lock()
	l.events = append(l.events, event)
	l.mu.Unlock()
}

func (l *eventLog) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return slices.Clone(l.events)
}

func (l *eventLog) index(event string) int {
	return slices.Index(l.snapshot(), event)
}

func (l *eventLog) waitFor(t *testing.T, event string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if l.index(event) >= 0 {
			return
		}

		time.Sleep(time.Millisecond)
	}

	t.Fatalf("event %q never happened, saw %v", event, l.snapshot())
}

type fakeSource struct {
	files    map[string][]provider.File
	contents map[string]string
	listErr  map[string]error
	openErr  map[string]error
	gates    map[string]chan struct{}
	events   *eventLog
	opened   atomic.Int64
}

func (s *fakeSource) List(ctx context.Context, target string, emit func(provider.File) error) error {
	s.events.add("list:" + target)

	for _, file := range s.files[target] {
		s.events.add("emit-start:" + file.Key)

		if err := emit(file); err != nil {
			return err
		}

		s.events.add("emit-done:" + file.Key)
	}

	return s.listErr[target]
}

func (s *fakeSource) Open(ctx context.Context, file provider.File) (io.ReadCloser, error) {
	s.opened.Add(1)
	s.events.add("open:" + file.Key)

	if gate, ok := s.gates[file.Key]; ok {
		select {
		case <-gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if err := s.openErr[file.Key]; err != nil {
		return nil, err
	}

	return io.NopCloser(strings.NewReader(s.contents[file.Key])), nil
}

var panRule = db.Rule{ID: 11, RuleName: "PAN", Type: db.RuleTypeRegex, Value: `[A-Z]{5}[0-9]{4}[A-Z]`}

func file(key string) provider.File {
	return provider.File{Key: key, Name: key, Size: 64}
}

func newScanner(t *testing.T, store *fakeStore, source strategy.Source, configure func(*scanner.Options)) *scanner.Scanner {
	t.Helper()

	options := scanner.Options{
		Store: store,
		Connect: func(context.Context, datadiscovery.ScanDefinition) (strategy.Source, error) {
			return source, nil
		},
		Processors:       scanner.DefaultProcessors(scanner.DefaultLimits(t.TempDir(), 16<<20)),
		QueueCapacity:    2,
		ChunkBytes:       64 << 10,
		EvaluatorWorkers: 2,
		FileTimeout:      5 * time.Second,
		ProgressInterval: time.Hour,
	}

	if configure != nil {
		configure(&options)
	}

	return scanner.New(options)
}

func scanRow() db.DataDiscoveryScan {
	return db.DataDiscoveryScan{ID: 42, CustomerID: 3, PolicyID: 7, Status: db.ScanStatusPending}
}

func assertInvariants(t *testing.T, counters db.ScanCounters) {
	t.Helper()

	if counters.FilesDiscovered != counters.FilesSupported+counters.FilesSkipped {
		t.Fatalf("discovered != supported + skipped: %+v", counters)
	}
	if counters.FilesSupported != counters.FilesProcessed {
		t.Fatalf("supported != processed: %+v", counters)
	}
	if counters.FilesProcessed != counters.FilesSucceeded+counters.FilesFailed {
		t.Fatalf("processed != succeeded + failed: %+v", counters)
	}
	if counters.CompletedTargets+counters.FailedTargets != counters.TotalTargets {
		t.Fatalf("targets do not add up: %+v", counters)
	}
}

func TestScannerHoldsQueueSlotUntilFilePersisted(t *testing.T) {
	events := &eventLog{}
	store := newStore(events, []string{"T1"}, []string{"txt"}, []db.Rule{panRule})
	release := make(chan struct{})

	source := &fakeSource{
		files:    map[string][]provider.File{"T1": {file("F11.txt"), file("F12.txt"), file("F13.txt")}},
		contents: map[string]string{"F11.txt": "PAN ABCDE1234F", "F12.txt": "clean", "F13.txt": "clean"},
		gates:    map[string]chan struct{}{"F11.txt": release},
		events:   events,
	}

	done := make(chan string, 1)
	go func() { done <- newScanner(t, store, source, nil).Receive(context.Background(), scanRow()) }()

	events.waitFor(t, "emit-start:F13.txt")
	events.waitFor(t, "open:F11.txt")
	time.Sleep(50 * time.Millisecond)

	if events.index("emit-done:F13.txt") >= 0 {
		t.Fatalf("F13 admitted before F11 completed: %v", events.snapshot())
	}
	if events.index("emit-done:F12.txt") < 0 {
		t.Fatalf("F12 should be admitted while F11 is processing: %v", events.snapshot())
	}

	close(release)

	select {
	case status := <-done:
		if status != db.ScanStatusCompleted {
			t.Fatalf("status = %s", status)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("scan did not finish")
	}

	saved, admitted := events.index("save:F11.txt"), events.index("emit-done:F13.txt")
	if saved < 0 || admitted < 0 || saved > admitted {
		t.Fatalf("F13 must be admitted only after F11 is persisted: %v", events.snapshot())
	}

	result, ok := store.result("F11.txt")
	if !ok || result.FindingsTotal != 1 || result.Status != db.FileStatusSucceeded {
		t.Fatalf("F11 result = %+v", result)
	}
}

func TestScannerProcessesTargetsSequentially(t *testing.T) {
	events := &eventLog{}
	store := newStore(events, []string{"T1", "T2"}, []string{"txt"}, []db.Rule{panRule})

	source := &fakeSource{
		files: map[string][]provider.File{
			"T1": {file("F11.txt"), file("F12.txt"), file("F13.txt")},
			"T2": {file("F21.txt")},
		},
		contents: map[string]string{},
		events:   events,
	}

	status := newScanner(t, store, source, nil).Receive(context.Background(), scanRow())
	if status != db.ScanStatusCompleted {
		t.Fatalf("status = %s", status)
	}

	second := events.index("list:T2")
	for _, key := range []string{"F11.txt", "F12.txt", "F13.txt"} {
		if saved := events.index("save:" + key); saved < 0 || saved > second {
			t.Fatalf("T2 started before %s drained: %v", key, events.snapshot())
		}
	}

	if finished := events.index("target:T1:" + db.TargetStatusCompleted); finished < 0 || finished > second {
		t.Fatalf("T1 not finalized before T2: %v", events.snapshot())
	}
}

func TestScannerCompletedScanCounters(t *testing.T) {
	events := &eventLog{}
	store := newStore(events, []string{"T1", "T2"}, nil, []db.Rule{panRule, {ID: 12, RuleName: "Secret", Type: db.RuleTypeKeyword, Value: "secret"}})

	source := &fakeSource{
		files: map[string][]provider.File{
			"T1": {file("a.txt"), file("photo.png"), file("b.csv")},
			"T2": {file("c.json"), file("noextension"), {Key: "typed", Name: "typed", MIMEType: "text/plain"}},
		},
		contents: map[string]string{
			"a.txt":  "PAN ABCDE1234F and FGHIJ5678K",
			"b.csv":  "id,secret\n1,SECRET",
			"c.json": `{"note":"clean"}`,
			"typed":  "ABCDE1234F",
		},
		events: events,
	}

	status := newScanner(t, store, source, nil).Receive(context.Background(), scanRow())
	if status != db.ScanStatusCompleted || store.status != db.ScanStatusCompleted {
		t.Fatalf("status = %s stored = %s", status, store.status)
	}

	counters := store.counters
	want := db.ScanCounters{
		TotalTargets: 2, CompletedTargets: 2,
		FilesDiscovered: 6, FilesSupported: 4, FilesSkipped: 2,
		FilesProcessed: 4, FilesSucceeded: 4, FindingsTotal: 5,
	}

	counters.BytesProcessed = 0
	if counters != want {
		t.Fatalf("counters = %+v\nwant       %+v", counters, want)
	}

	assertInvariants(t, store.counters)

	if store.targets[0].FilesSkipped != 1 || store.targets[1].FilesSkipped != 1 {
		t.Fatalf("target skipped counts = %+v", store.targets)
	}
}

func TestScannerFileFailureMakesScanPartial(t *testing.T) {
	events := &eventLog{}
	store := newStore(events, []string{"T1"}, []string{"txt", "docx"}, []db.Rule{panRule})

	source := &fakeSource{
		files:    map[string][]provider.File{"T1": {file("F11.txt"), file("F12.txt"), file("F13.docx"), file("F14.txt")}},
		contents: map[string]string{"F11.txt": "ok", "F13.docx": "not a zip", "F14.txt": "ABCDE1234F"},
		openErr:  map[string]error{"F12.txt": &provider.Error{Provider: "azure-storage", Stage: "download", Reason: provider.ReasonNotFound}},
		events:   events,
	}

	status := newScanner(t, store, source, nil).Receive(context.Background(), scanRow())
	if status != db.ScanStatusPartial {
		t.Fatalf("status = %s", status)
	}

	fetch, _ := store.result("F12.txt")
	if fetch.Status != db.FileStatusFailed || fetch.ErrorCode == nil || *fetch.ErrorCode != "FETCH_NOT_FOUND" {
		t.Fatalf("fetch failure = %+v", fetch)
	}

	parse, _ := store.result("F13.docx")
	if parse.Status != db.FileStatusFailed || parse.ErrorCode == nil || *parse.ErrorCode != db.FileErrorParseFailed {
		t.Fatalf("parse failure = %+v", parse)
	}

	last, _ := store.result("F14.txt")
	if last.Status != db.FileStatusSucceeded || last.FindingsTotal != 1 {
		t.Fatalf("processing did not continue after failures: %+v", last)
	}

	if store.counters.FilesFailed != 2 || store.counters.FilesSucceeded != 2 {
		t.Fatalf("counters = %+v", store.counters)
	}

	if store.targets[0].Status != db.TargetStatusPartial {
		t.Fatalf("target = %+v", store.targets[0])
	}

	assertInvariants(t, store.counters)
}

func TestScannerTargetFailureContinuesWithNextTarget(t *testing.T) {
	events := &eventLog{}
	store := newStore(events, []string{"T1", "T2"}, []string{"txt"}, []db.Rule{panRule})

	source := &fakeSource{
		files: map[string][]provider.File{
			"T1": {file("F11.txt")},
			"T2": {file("F21.txt")},
		},
		contents: map[string]string{},
		listErr: map[string]error{
			"T1": &provider.Error{Provider: "azure-storage", Stage: "list", Reason: provider.ReasonPermissionDenied},
		},
		events: events,
	}

	status := newScanner(t, store, source, nil).Receive(context.Background(), scanRow())
	if status != db.ScanStatusPartial {
		t.Fatalf("status = %s", status)
	}

	if _, ok := store.result("F11.txt"); !ok {
		t.Fatal("file queued before the listing failure was not processed")
	}

	if _, ok := store.result("F21.txt"); !ok {
		t.Fatal("next target did not run")
	}

	first := store.targets[0]
	if first.Status != db.TargetStatusFailed || first.ErrorCode == nil || *first.ErrorCode != "LISTING_PERMISSION_DENIED" {
		t.Fatalf("first target = %+v", first)
	}

	if store.counters.FailedTargets != 1 || store.counters.CompletedTargets != 1 {
		t.Fatalf("counters = %+v", store.counters)
	}

	assertInvariants(t, store.counters)
}

func TestScannerAllTargetsFailed(t *testing.T) {
	events := &eventLog{}
	store := newStore(events, []string{"T1", "T2"}, []string{"txt"}, []db.Rule{panRule})
	failure := &provider.Error{Provider: "azure-storage", Stage: "list", Reason: provider.ReasonNotFound}

	source := &fakeSource{
		files:   map[string][]provider.File{},
		listErr: map[string]error{"T1": failure, "T2": failure},
		events:  events,
	}

	status := newScanner(t, store, source, nil).Receive(context.Background(), scanRow())
	if status != db.ScanStatusFailed || store.code != db.ScanErrorAllTargetsFailed {
		t.Fatalf("status = %s code = %s", status, store.code)
	}
}

func TestScannerInitializationFailures(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(*fakeStore, *scanner.Options)
		code    string
	}{
		{
			name:    "policy inactive",
			prepare: func(s *fakeStore, _ *scanner.Options) { s.definitionErr = utils.ErrDiscoveryPolicyInactive },
			code:    db.ScanErrorPolicyInactive,
		},
		{
			name:    "credential unavailable",
			prepare: func(s *fakeStore, _ *scanner.Options) { s.definitionErr = utils.ErrCredentialUnavailable },
			code:    db.ScanErrorCredentialUnavailable,
		},
		{
			name:    "no rules",
			prepare: func(s *fakeStore, _ *scanner.Options) { s.definition.Rules = nil },
			code:    db.ScanErrorNoRules,
		},
		{
			name: "unsupported source",
			prepare: func(_ *fakeStore, o *scanner.Options) {
				o.Connect = func(context.Context, datadiscovery.ScanDefinition) (strategy.Source, error) {
					return nil, strategy.ErrSourceUnsupported
				}
			},
			code: db.ScanErrorSourceUnsupported,
		},
		{
			name: "source authentication",
			prepare: func(_ *fakeStore, o *scanner.Options) {
				o.Connect = func(context.Context, datadiscovery.ScanDefinition) (strategy.Source, error) {
					return nil, &provider.Error{Provider: "azure-storage", Stage: "token", Reason: provider.ReasonAuthFailed}
				}
			},
			code: db.ScanErrorSourceAuthFailed,
		},
		{
			name:    "database",
			prepare: func(s *fakeStore, _ *scanner.Options) { s.definitionErr = errors.New("connection refused") },
			code:    db.ScanErrorDatabaseUnavailable,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events := &eventLog{}
			store := newStore(events, []string{"T1"}, []string{"txt"}, []db.Rule{panRule})
			source := &fakeSource{events: events}

			instance := newScanner(t, store, source, func(o *scanner.Options) { tc.prepare(store, o) })

			status := instance.Receive(context.Background(), scanRow())
			if status != db.ScanStatusFailed || store.code != tc.code {
				t.Fatalf("status = %s code = %s", status, store.code)
			}

			if store.started {
				t.Fatal("scan must not be marked running after an initialization failure")
			}
		})
	}
}

func TestScannerSkipsScanWhoseStateChangedBeforeStart(t *testing.T) {
	events := &eventLog{}
	store := newStore(events, []string{"T1"}, []string{"txt"}, []db.Rule{panRule})
	store.startErr = datadiscovery.ErrScanStateChanged

	status := newScanner(t, store, &fakeSource{events: events}, nil).Receive(context.Background(), scanRow())
	if status != "" || store.finished {
		t.Fatalf("status = %q finished = %v", status, store.finished)
	}
}

func TestScannerPersistenceFailureFailsScan(t *testing.T) {
	events := &eventLog{}
	store := newStore(events, []string{"T1", "T2"}, []string{"txt"}, []db.Rule{panRule})
	store.saveErr = errors.New("database is down")

	source := &fakeSource{
		files: map[string][]provider.File{
			"T1": {file("F11.txt"), file("F12.txt"), file("F13.txt")},
			"T2": {file("F21.txt")},
		},
		contents: map[string]string{},
		events:   events,
	}

	status := newScanner(t, store, source, nil).Receive(context.Background(), scanRow())
	if status != db.ScanStatusFailed || store.code != db.ScanErrorDatabaseUnavailable {
		t.Fatalf("status = %s code = %s", status, store.code)
	}

	if store.saves != 3 {
		t.Fatalf("save attempts = %d, want 3 retries for the first file only", store.saves)
	}

	if events.index("list:T2") >= 0 {
		t.Fatal("later targets must not run after a database failure")
	}

	if store.counters.FilesFailed != store.counters.FilesProcessed {
		t.Fatalf("counters = %+v", store.counters)
	}
}

func TestScannerFileTimeout(t *testing.T) {
	events := &eventLog{}
	store := newStore(events, []string{"T1"}, []string{"txt"}, []db.Rule{panRule})

	source := &fakeSource{
		files:    map[string][]provider.File{"T1": {file("slow.txt"), file("fast.txt")}},
		contents: map[string]string{"fast.txt": "ABCDE1234F"},
		gates:    map[string]chan struct{}{"slow.txt": make(chan struct{})},
		events:   events,
	}

	instance := newScanner(t, store, source, func(o *scanner.Options) { o.FileTimeout = 300 * time.Millisecond })

	status := instance.Receive(context.Background(), scanRow())
	if status != db.ScanStatusPartial {
		t.Fatalf("status = %s", status)
	}

	slow, _ := store.result("slow.txt")
	if slow.ErrorCode == nil || *slow.ErrorCode != db.FileErrorTimeout {
		t.Fatalf("slow result = %+v", slow)
	}

	if fast, _ := store.result("fast.txt"); fast.FindingsTotal != 1 {
		t.Fatalf("fast result = %+v", fast)
	}
}

func TestScannerCancellationInterruptsWithoutLeaks(t *testing.T) {
	before := settledGoroutines()

	events := &eventLog{}
	store := newStore(events, []string{"T1", "T2"}, []string{"txt"}, []db.Rule{panRule})

	source := &fakeSource{
		files: map[string][]provider.File{
			"T1": {file("F11.txt"), file("F12.txt"), file("F13.txt"), file("F14.txt")},
			"T2": {file("F21.txt")},
		},
		contents: map[string]string{},
		gates:    map[string]chan struct{}{"F11.txt": make(chan struct{})},
		events:   events,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan string, 1)

	go func() { done <- newScanner(t, store, source, nil).Receive(ctx, scanRow()) }()

	events.waitFor(t, "open:F11.txt")
	cancel()

	select {
	case status := <-done:
		if status != db.ScanStatusFailed || store.code != db.ScanErrorInterrupted {
			t.Fatalf("status = %s code = %s", status, store.code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("scan did not stop after cancellation")
	}

	if events.index("list:T2") >= 0 {
		t.Fatal("later targets must not start after cancellation")
	}

	if len(store.saved) != 0 {
		t.Fatalf("no results should be persisted for cancelled files: %+v", store.saved)
	}

	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if runtime.NumGoroutine() > before {
		t.Fatalf("goroutines leaked: before %d after %d", before, runtime.NumGoroutine())
	}
}

func TestScannerStopsWhenStateChangesExternally(t *testing.T) {
	events := &eventLog{}
	store := newStore(events, []string{"T1"}, []string{"txt"}, []db.Rule{panRule})
	store.progressErr = datadiscovery.ErrScanStateChanged

	source := &fakeSource{
		files:    map[string][]provider.File{"T1": {file("F11.txt")}},
		contents: map[string]string{},
		gates:    map[string]chan struct{}{"F11.txt": make(chan struct{})},
		events:   events,
	}

	instance := newScanner(t, store, source, func(o *scanner.Options) { o.ProgressInterval = 10 * time.Millisecond })

	done := make(chan string, 1)
	go func() { done <- instance.Receive(context.Background(), scanRow()) }()

	select {
	case status := <-done:
		if status != "" || store.finished {
			t.Fatalf("status = %q finished = %v", status, store.finished)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("scan did not stop after an external state change")
	}
}

func TestScannerRunClaimsPendingScans(t *testing.T) {
	events := &eventLog{}
	store := newStore(events, []string{"T1"}, []string{"txt"}, []db.Rule{panRule})
	store.claims = []db.DataDiscoveryScan{scanRow()}

	source := &fakeSource{
		files:    map[string][]provider.File{"T1": {file("F11.txt")}},
		contents: map[string]string{"F11.txt": "ABCDE1234F"},
		events:   events,
	}

	instance := newScanner(t, store, source, func(o *scanner.Options) { o.PollInterval = 10 * time.Millisecond })

	ctx, cancel := context.WithCancel(context.Background())
	instance.Start(ctx)

	events.waitFor(t, "save:F11.txt")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		store.mu.Lock()
		finished := store.finished
		store.mu.Unlock()

		if finished {
			break
		}

		time.Sleep(5 * time.Millisecond)
	}

	cancel()
	instance.Wait()

	if !store.finished || store.status != db.ScanStatusCompleted {
		t.Fatalf("finished = %v status = %s", store.finished, store.status)
	}

	if store.staleSweep == 0 {
		t.Fatal("stale sweep never ran")
	}
}
