//go:build integration

package scans_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	discoveryhandler "dpdp-backend/internal/admin/handler/datadiscovery"
	discoveryrepo "dpdp-backend/internal/admin/repositories/datadiscovery"
	discoverysvc "dpdp-backend/internal/admin/services/datadiscovery"
	"dpdp-backend/internal/admin/utils"
	appcrypto "dpdp-backend/internal/crypto"
	"dpdp-backend/internal/datadiscovery/provider"
	"dpdp-backend/internal/datadiscovery/scanner"
	"dpdp-backend/internal/datadiscovery/strategy"
	"dpdp-backend/internal/db"
	"dpdp-backend/tests/testsupport"
)

type fixture struct {
	database *gorm.DB
	service  *discoverysvc.ScanService
	customer db.Customer
	box      *appcrypto.SecretBox
}

func setup(t *testing.T) fixture {
	t.Helper()

	database := testsupport.Postgres(t)
	customer := testsupport.Customer(t, database)

	t.Cleanup(func() {
		database.Exec("DELETE FROM customers WHERE id = ?", customer.ID)
	})

	box, err := appcrypto.NewSecretBox(base64.StdEncoding.EncodeToString(make([]byte, 32)), 1)
	if err != nil {
		t.Fatalf("secret box: %v", err)
	}

	registry := strategy.DefaultPolicyRegistry()
	configurations := discoverysvc.NewConfigurationService(
		database,
		discoveryrepo.NewConfigurationRepository(database),
		strategy.DefaultConfigurationRegistry(provider.NewClient(time.Second)),
		box,
		time.Second,
	)

	service := discoverysvc.NewScanService(
		database,
		discoveryrepo.NewScanRepository(database),
		discoveryrepo.NewPolicyRepository(database),
		configurations,
		registry,
	)

	return fixture{database: database, service: service, customer: customer, box: box}
}

func (f fixture) configuration(t *testing.T, kind string, config db.StringMap) db.DataDiscoveryConfiguration {
	t.Helper()

	row := db.DataDiscoveryConfiguration{
		CustomerID:        f.customer.ID,
		Name:              "cfg-" + strconv.FormatInt(time.Now().UnixNano(), 36),
		ConfigurationType: kind,
		Config:            config,
		Status:            db.DiscoveryStatusActive,
		LastTestedAt:      time.Now(),
	}

	if err := discoveryrepo.NewConfigurationRepository(f.database).Insert(context.Background(), &row); err != nil {
		t.Fatalf("configuration insert: %v", err)
	}

	sealed, err := f.box.Seal([]byte(`{"clientSecret":"s3cret"}`), appcrypto.CredentialAAD(1, f.customer.ID, row.ID))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	if err := discoveryrepo.NewConfigurationRepository(f.database).UpsertCredential(context.Background(), f.customer.ID, row.ID, sealed, 1); err != nil {
		t.Fatalf("credential insert: %v", err)
	}

	return row
}

func (f fixture) blobPolicy(t *testing.T, name, status string, targets ...string) db.DataDiscoveryPolicy {
	t.Helper()

	configuration := f.configuration(t, db.ConfigurationTypeAzureStorage, db.StringMap{
		"storageAccount": "acct",
		"tenantId":       "tenant",
		"clientId":       "client",
		"authMode":       db.AzureAuthModeServicePrincipal,
	})

	return f.policy(t, name, status, configuration, db.SourceTypeAzureBlob, targets...)
}

func (f fixture) policy(
	t *testing.T,
	name, status string,
	configuration db.DataDiscoveryConfiguration,
	source string,
	targets ...string,
) db.DataDiscoveryPolicy {
	t.Helper()

	row := db.DataDiscoveryPolicy{
		CustomerID:        f.customer.ID,
		Name:              name,
		ConfigurationID:   configuration.ID,
		ConfigurationType: configuration.ConfigurationType,
		SourceType:        source,
		TargetList:        db.StringList(targets),
		FileTypes:         db.StringList{"txt"},
		Status:            status,
	}

	repository := discoveryrepo.NewPolicyRepository(f.database)
	if err := repository.Insert(context.Background(), &row); err != nil {
		t.Fatalf("policy insert: %v", err)
	}

	rule := testsupport.Rule(t, f.database, f.customer.ID, "PAN-"+name, db.RuleTypeRegex, `[A-Z]{5}[0-9]{4}[A-Z]`)
	if err := repository.ReplaceRules(context.Background(), f.customer.ID, row.ID, []int{rule.ID}); err != nil {
		t.Fatalf("rule mapping: %v", err)
	}

	return row
}

func claimed(t *testing.T, f fixture, policy db.DataDiscoveryPolicy) db.DataDiscoveryScan {
	t.Helper()

	detail, err := f.service.Create(context.Background(), f.customer.ID, 0, policy.ID)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	for {
		scan, ok, err := f.service.Claim(context.Background())
		if err != nil || !ok {
			t.Fatalf("claim: ok=%v err=%v", ok, err)
		}

		if scan.ID == detail.Scan.ID {
			return scan
		}
	}
}

func TestIntegrationScanCreateRules(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	active := f.blobPolicy(t, "Blob active", db.DiscoveryStatusActive, "finance")

	detail, err := f.service.Create(ctx, f.customer.ID, 5, active.ID)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if detail.Scan.Status != db.ScanStatusPending || detail.PolicyName != "Blob active" || detail.Scan.RequestedBy == nil {
		t.Fatalf("detail = %+v", detail)
	}

	select {
	case <-f.service.Created():
	default:
		t.Fatal("scanner was not signalled")
	}

	if _, err := f.service.Create(ctx, f.customer.ID, 5, active.ID); !errors.Is(err, utils.ErrDiscoveryScanActive) {
		t.Fatalf("second active scan error = %v", err)
	}

	inactive := f.blobPolicy(t, "Blob inactive", db.DiscoveryStatusInactive, "finance")
	if _, err := f.service.Create(ctx, f.customer.ID, 5, inactive.ID); !errors.Is(err, utils.ErrDiscoveryPolicyInactive) {
		t.Fatalf("inactive policy error = %v", err)
	}

	aws := f.configuration(t, db.ConfigurationTypeAWSIAM, db.StringMap{"accessKeyId": "AKIAABCDEFGHIJKLMNOP", "region": "ap-south-1"})
	s3 := f.policy(t, "S3", db.DiscoveryStatusActive, aws, db.SourceTypeAWSS3, "bucket-one")
	if created, err := f.service.Create(ctx, f.customer.ID, 5, s3.ID); err != nil || created.Scan.Status != db.ScanStatusPending {
		t.Fatalf("s3 scan = %+v err = %v", created.Scan, err)
	}

	if _, err := f.service.Create(ctx, f.customer.ID, 5, 999999); !errors.Is(err, utils.ErrDiscoveryPolicyNotFound) {
		t.Fatalf("missing policy error = %v", err)
	}
}

func TestIntegrationScanClaimSkipsLockedRows(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	ids := map[int64]bool{}
	for index := range 2 {
		policy := f.blobPolicy(t, "Claim "+strconv.Itoa(index), db.DiscoveryStatusActive, "finance")

		detail, err := f.service.Create(ctx, f.customer.ID, 0, policy.ID)
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		ids[detail.Scan.ID] = true
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	got := map[int64]int{}

	for range 2 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			scan, ok, err := f.service.Claim(ctx)
			if err != nil || !ok {
				t.Errorf("claim ok=%v err=%v", ok, err)
				return
			}

			mu.Lock()
			got[scan.ID]++
			mu.Unlock()
		}()
	}

	wg.Wait()

	for id, count := range got {
		if !ids[id] || count != 1 {
			t.Fatalf("claims = %v, created = %v", got, ids)
		}
	}

	if len(got) != 2 {
		t.Fatalf("claims = %v", got)
	}
}

func TestIntegrationScanLifecycle(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	policy := f.blobPolicy(t, "Lifecycle", db.DiscoveryStatusActive, "finance", "hr/payroll")
	scan := claimed(t, f, policy)

	definition, err := f.service.Definition(ctx, scan)
	if err != nil {
		t.Fatalf("definition: %v", err)
	}

	if string(definition.Credential) != `{"clientSecret":"s3cret"}` || len(definition.Rules) != 1 || definition.Rules[0].Value == "" {
		t.Fatalf("definition = %+v", definition)
	}

	if err := f.service.Start(ctx, scan, definition.Policy.TargetList); err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := f.service.Start(ctx, scan, definition.Policy.TargetList); !errors.Is(err, discoverysvc.ErrScanStateChanged) {
		t.Fatalf("second start error = %v", err)
	}

	counters := db.ScanCounters{TotalTargets: 2, FilesDiscovered: 3, FilesSupported: 2, FilesSkipped: 1, FilesProcessed: 2}
	if err := f.service.Progress(ctx, scan.ID, counters); err != nil {
		t.Fatalf("progress: %v", err)
	}

	result := db.DataDiscoveryFileResult{
		ScanID: scan.ID, CustomerID: f.customer.ID, TargetPosition: 0,
		FileKey: "finance/a.txt", FileName: "a.txt", Extension: "txt",
		Status: db.FileStatusFailed, ErrorCode: pointer(db.FileErrorTimeout), ProcessedAt: time.Now(),
	}

	if err := f.service.SaveFile(ctx, result); err != nil {
		t.Fatalf("save: %v", err)
	}

	result.Status = db.FileStatusSucceeded
	result.ErrorCode = nil
	result.FindingsTotal = 7
	result.Findings = db.FileFindings{{RuleID: 1, RuleName: "PAN", RuleType: db.RuleTypeRegex, Count: 7, Offsets: []int64{0, 12}}}

	if err := f.service.SaveFile(ctx, result); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	files, err := f.service.Files(ctx, f.customer.ID, scan.ID, discoveryrepo.FileResultListParams{WithFindings: true})
	if err != nil {
		t.Fatalf("files: %v", err)
	}

	if files.Total != 1 || files.Items[0].Status != db.FileStatusSucceeded || files.Items[0].ErrorCode != nil ||
		files.Items[0].Findings[0].Count != 7 || len(files.Items[0].Findings[0].Offsets) != 2 {
		t.Fatalf("files = %+v", files)
	}

	if err := f.service.UpdateTarget(ctx, db.DataDiscoveryScanTarget{
		ScanID: scan.ID, Position: 0, Status: db.TargetStatusCompleted, FilesSucceeded: 1, FindingsTotal: 7,
	}); err != nil {
		t.Fatalf("update target: %v", err)
	}

	counters.FilesSucceeded, counters.FindingsTotal, counters.CompletedTargets = 1, 7, 1
	if err := f.service.Finish(ctx, scan.ID, db.ScanStatusFailed, db.ScanErrorInterrupted, counters); err != nil {
		t.Fatalf("finish: %v", err)
	}

	detail, err := f.service.Get(ctx, f.customer.ID, scan.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if detail.Scan.Status != db.ScanStatusFailed || detail.Scan.FindingsTotal != 7 || detail.Scan.FinishedAt == nil {
		t.Fatalf("scan = %+v", detail.Scan)
	}

	if detail.Targets[0].Status != db.TargetStatusCompleted || detail.Targets[1].Status != db.TargetStatusFailed ||
		detail.Targets[1].ErrorCode == nil || *detail.Targets[1].ErrorCode != db.ScanErrorInterrupted {
		t.Fatalf("targets = %+v", detail.Targets)
	}

	if err := f.service.Progress(ctx, scan.ID, counters); !errors.Is(err, discoverysvc.ErrScanStateChanged) {
		t.Fatalf("progress after finish error = %v", err)
	}

	if err := f.service.Finish(ctx, scan.ID, db.ScanStatusCompleted, "", counters); err != nil {
		t.Fatalf("second finish: %v", err)
	}

	again, _ := f.service.Get(ctx, f.customer.ID, scan.ID)
	if again.Scan.Status != db.ScanStatusFailed {
		t.Fatalf("terminal status overwritten: %s", again.Scan.Status)
	}

	if _, err := f.service.Create(ctx, f.customer.ID, 0, policy.ID); err != nil {
		t.Fatalf("new scan after terminal state: %v", err)
	}
}

func TestIntegrationFailStaleInterruptsAbandonedScans(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	policy := f.blobPolicy(t, "Stale", db.DiscoveryStatusActive, "finance")
	scan := claimed(t, f, policy)

	if err := f.service.Start(ctx, scan, []string{"finance"}); err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := f.database.Exec("UPDATE data_discovery_scans SET updated_at = now() - interval '1 hour' WHERE id = ?", scan.ID).Error; err != nil {
		t.Fatalf("age scan: %v", err)
	}

	failed, err := f.service.FailStale(ctx, time.Now().Add(-2*time.Minute))
	if err != nil || failed < 1 {
		t.Fatalf("fail stale = %d err = %v", failed, err)
	}

	detail, _ := f.service.Get(ctx, f.customer.ID, scan.ID)
	if detail.Scan.Status != db.ScanStatusFailed || *detail.Scan.ErrorCode != db.ScanErrorInterrupted {
		t.Fatalf("scan = %+v", detail.Scan)
	}

	if detail.Targets[0].Status != db.TargetStatusFailed {
		t.Fatalf("targets = %+v", detail.Targets)
	}
}

type memorySource struct {
	files map[string]string
}

func (s memorySource) List(_ context.Context, target string, emit func(provider.File) error) error {
	for key, content := range s.files {
		if !strings.HasPrefix(key, target+"/") {
			continue
		}

		if err := emit(provider.File{Key: key, Name: strings.TrimPrefix(key, target+"/"), Size: int64(len(content))}); err != nil {
			return err
		}
	}

	return nil
}

func (s memorySource) Open(_ context.Context, file provider.File) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(s.files[file.Key])), nil
}

func TestIntegrationScannerEndToEnd(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	policy := f.blobPolicy(t, "End to end", db.DiscoveryStatusActive, "finance", "hr")

	if _, err := f.service.Create(ctx, f.customer.ID, 0, policy.ID); err != nil {
		t.Fatalf("create: %v", err)
	}

	source := memorySource{files: map[string]string{
		"finance/a.txt":     "PAN ABCDE1234F",
		"finance/b.txt":     "nothing here",
		"finance/photo.png": "binary",
		"hr/c.txt":          "FGHIJ5678K and KLMNO9012P",
	}}

	instance := scanner.New(scanner.Options{
		Store: f.service,
		Connect: func(context.Context, discoverysvc.ScanDefinition) (strategy.Source, error) {
			return source, nil
		},
		Processors:    scanner.DefaultProcessors(scanner.DefaultLimits(t.TempDir(), 16<<20)),
		QueueCapacity: 2,
	})

	var scan db.DataDiscoveryScan
	for {
		next, ok, err := f.service.Claim(ctx)
		if err != nil || !ok {
			t.Fatalf("claim ok=%v err=%v", ok, err)
		}

		if next.PolicyID == policy.ID {
			scan = next
			break
		}
	}

	if status := instance.Receive(ctx, scan); status != db.ScanStatusCompleted {
		t.Fatalf("status = %s", status)
	}

	detail, err := f.service.Get(ctx, f.customer.ID, scan.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	counters := detail.Scan.ScanCounters
	if counters.FilesDiscovered != 4 || counters.FilesSkipped != 1 || counters.FilesSucceeded != 3 ||
		counters.FindingsTotal != 3 || counters.CompletedTargets != 2 {
		t.Fatalf("counters = %+v", counters)
	}

	files, _ := f.service.Files(ctx, f.customer.ID, scan.ID, discoveryrepo.FileResultListParams{})
	if files.Total != 3 || files.Items[0].FileKey != "hr/c.txt" || files.Items[0].FindingsTotal != 2 {
		t.Fatalf("files = %+v", files.Items)
	}
}

func TestIntegrationScanRoutes(t *testing.T) {
	f := setup(t)

	policy := f.blobPolicy(t, "Routes", db.DiscoveryStatusActive, "finance")

	router, group := testsupport.Router(t, f.customer.ID, nil)
	discoveryhandler.NewScanHandler(f.service).RegisterRoutes(group, func(string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})

	created := call(t, router, http.MethodPost, "/api/v1/admin/data-discovery/scans", `{"policy_id":`+strconv.Itoa(policy.ID)+`}`)
	if created.Code != http.StatusAccepted {
		t.Fatalf("create status = %d body = %s", created.Code, created.Body)
	}

	var response struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}

	if err := json.Unmarshal(created.Body.Bytes(), &response); err != nil || response.Status != db.ScanStatusPending {
		t.Fatalf("create body = %s", created.Body)
	}

	if conflict := call(t, router, http.MethodPost, "/api/v1/admin/data-discovery/scans", `{"policy_id":`+strconv.Itoa(policy.ID)+`}`); conflict.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d", conflict.Code)
	}

	path := "/api/v1/admin/data-discovery/scans/" + strconv.FormatInt(response.ID, 10)

	for _, url := range []string{"/api/v1/admin/data-discovery/scans?status=PENDING", path, path + "/files"} {
		if got := call(t, router, http.MethodGet, url, ""); got.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d body = %s", url, got.Code, got.Body)
		}
	}

	if missing := call(t, router, http.MethodGet, "/api/v1/admin/data-discovery/scans/999999", ""); missing.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d", missing.Code)
	}

	if invalid := call(t, router, http.MethodPost, "/api/v1/admin/data-discovery/scans", `{}`); invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d", invalid.Code)
	}
}

func call(t *testing.T, router *gin.Engine, method, url, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, url, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

func pointer(value string) *string {
	return &value
}
