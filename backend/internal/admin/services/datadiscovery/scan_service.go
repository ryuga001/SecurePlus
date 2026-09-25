package datadiscovery

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	repo "dpdp-backend/internal/admin/repositories/datadiscovery"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/datadiscovery/strategy"
	"dpdp-backend/internal/db"
)

var ErrScanStateChanged = errors.New("scan state changed outside this scanner")

type ScanDetail struct {
	Scan       db.DataDiscoveryScan
	PolicyName string
	Targets    []db.DataDiscoveryScanTarget
}

type ScanSummary struct {
	Scan       db.DataDiscoveryScan
	PolicyName string
}

type ScanListing struct {
	Items    []ScanSummary
	Page     int
	PageSize int
	Total    int64
}

type FileResultListing struct {
	Items    []db.DataDiscoveryFileResult
	Page     int
	PageSize int
	Total    int64
}

type ScanDefinition struct {
	Scan          db.DataDiscoveryScan
	Policy        db.DataDiscoveryPolicy
	Configuration db.DataDiscoveryConfiguration
	Credential    []byte
	Rules         []db.Rule
}

type ScanService struct {
	db             *gorm.DB
	repo           *repo.ScanRepository
	policies       *repo.PolicyRepository
	configurations *ConfigurationService
	registry       *strategy.PolicyRegistry
	created        chan struct{}
}

func NewScanService(
	database *gorm.DB,
	repository *repo.ScanRepository,
	policies *repo.PolicyRepository,
	configurations *ConfigurationService,
	registry *strategy.PolicyRegistry,
) *ScanService {
	return &ScanService{
		db:             database,
		repo:           repository,
		policies:       policies,
		configurations: configurations,
		registry:       registry,
		created:        make(chan struct{}, 1),
	}
}

func (s *ScanService) Create(ctx context.Context, customerID, requestedBy, policyID int) (ScanDetail, error) {
	policy, err := s.policies.Find(ctx, customerID, policyID)
	if err != nil {
		if db.IsNotFound(err) {
			return ScanDetail{}, utils.ErrDiscoveryPolicyNotFound
		}

		return ScanDetail{}, err
	}

	if policy.Status != db.DiscoveryStatusActive {
		return ScanDetail{}, utils.ErrDiscoveryPolicyInactive
	}

	selected, ok := s.registry.For(policy.SourceType)
	if !ok || !selected.Scannable() {
		return ScanDetail{}, utils.ErrSourceNotScannable
	}

	row := db.DataDiscoveryScan{
		CustomerID: customerID,
		PolicyID:   policy.ID,
		Status:     db.ScanStatusPending,
	}

	if requestedBy > 0 {
		row.RequestedBy = &requestedBy
	}

	if err := s.repo.Insert(ctx, &row); err != nil {
		switch {
		case db.IsDuplicate(err):
			return ScanDetail{}, utils.ErrDiscoveryScanActive
		case db.IsMissingReference(err):
			return ScanDetail{}, utils.ErrDiscoveryPolicyNotFound
		}

		return ScanDetail{}, err
	}

	s.signal()

	return s.Get(ctx, customerID, row.ID)
}

func (s *ScanService) Get(ctx context.Context, customerID int, id int64) (ScanDetail, error) {
	row, err := s.repo.Find(ctx, customerID, id)
	if err != nil {
		if db.IsNotFound(err) {
			return ScanDetail{}, utils.ErrDiscoveryScanNotFound
		}

		return ScanDetail{}, err
	}

	targets, err := s.repo.Targets(ctx, customerID, row.ID)
	if err != nil {
		return ScanDetail{}, err
	}

	names, err := s.repo.PolicyNamesFor(ctx, customerID, []int{row.PolicyID})
	if err != nil {
		return ScanDetail{}, err
	}

	detail := ScanDetail{Scan: row, Targets: targets}
	if len(names) > 0 {
		detail.PolicyName = names[0].Name
	}

	return detail, nil
}

func (s *ScanService) List(ctx context.Context, customerID int, params repo.ScanListParams) (ScanListing, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	rows, total, err := s.repo.List(ctx, customerID, params)
	if err != nil {
		return ScanListing{}, err
	}

	policyIDs := make([]int, 0, len(rows))
	for _, row := range rows {
		policyIDs = append(policyIDs, row.PolicyID)
	}

	names, err := s.repo.PolicyNamesFor(ctx, customerID, utils.NormalizeIDs(policyIDs))
	if err != nil {
		return ScanListing{}, err
	}

	byID := make(map[int]string, len(names))
	for _, name := range names {
		byID[name.ID] = name.Name
	}

	items := make([]ScanSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, ScanSummary{Scan: row, PolicyName: byID[row.PolicyID]})
	}

	return ScanListing{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *ScanService) Files(
	ctx context.Context,
	customerID int,
	scanID int64,
	params repo.FileResultListParams,
) (FileResultListing, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	if _, err := s.repo.Find(ctx, customerID, scanID); err != nil {
		if db.IsNotFound(err) {
			return FileResultListing{}, utils.ErrDiscoveryScanNotFound
		}

		return FileResultListing{}, err
	}

	rows, total, err := s.repo.ListFiles(ctx, customerID, scanID, params)
	if err != nil {
		return FileResultListing{}, err
	}

	return FileResultListing{Items: rows, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *ScanService) Created() <-chan struct{} {
	return s.created
}

func (s *ScanService) Claim(ctx context.Context) (db.DataDiscoveryScan, bool, error) {
	return s.repo.Claim(ctx)
}

func (s *ScanService) Definition(ctx context.Context, scan db.DataDiscoveryScan) (ScanDefinition, error) {
	policy, err := s.policies.Find(ctx, scan.CustomerID, scan.PolicyID)
	if err != nil {
		if db.IsNotFound(err) {
			return ScanDefinition{}, utils.ErrDiscoveryPolicyNotFound
		}

		return ScanDefinition{}, err
	}

	if policy.Status != db.DiscoveryStatusActive {
		return ScanDefinition{}, utils.ErrDiscoveryPolicyInactive
	}

	configuration, err := s.configurations.Get(ctx, scan.CustomerID, policy.ConfigurationID)
	if err != nil {
		if errors.Is(err, utils.ErrDiscoveryConfigNotFound) {
			return ScanDefinition{}, utils.ErrUnknownConfiguration
		}

		return ScanDefinition{}, err
	}

	if configuration.Configuration.Status != db.DiscoveryStatusActive {
		return ScanDefinition{}, utils.ErrUnknownConfiguration
	}

	credential, err := s.configurations.OpenCredential(ctx, scan.CustomerID, policy.ConfigurationID)
	if err != nil {
		return ScanDefinition{}, err
	}

	rules, err := s.policies.RuleDefinitionsFor(ctx, scan.CustomerID, policy.ID)
	if err != nil {
		return ScanDefinition{}, err
	}

	return ScanDefinition{
		Scan:          scan,
		Policy:        policy,
		Configuration: configuration.Configuration,
		Credential:    credential,
		Rules:         rules,
	}, nil
}

func (s *ScanService) Start(ctx context.Context, scan db.DataDiscoveryScan, targets []string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		affected, err := repository.MarkRunning(ctx, scan.ID, len(targets))
		if err != nil {
			return err
		}
		if affected == 0 {
			return ErrScanStateChanged
		}

		rows := make([]db.DataDiscoveryScanTarget, 0, len(targets))
		for position, target := range targets {
			rows = append(rows, db.DataDiscoveryScanTarget{
				ScanID:     scan.ID,
				Position:   position,
				CustomerID: scan.CustomerID,
				Target:     target,
				Status:     db.TargetStatusPending,
			})
		}

		return repository.InsertTargets(ctx, rows)
	})
}

func (s *ScanService) Progress(ctx context.Context, scanID int64, counters db.ScanCounters) error {
	affected, err := s.repo.Progress(ctx, scanID, counters)
	if err != nil {
		return err
	}

	if affected == 0 {
		return ErrScanStateChanged
	}

	return nil
}

func (s *ScanService) UpdateTarget(ctx context.Context, target db.DataDiscoveryScanTarget) error {
	return s.repo.UpdateTarget(ctx, target)
}

func (s *ScanService) SaveFile(ctx context.Context, result db.DataDiscoveryFileResult) error {
	return s.repo.UpsertFileResult(ctx, &result)
}

func (s *ScanService) Finish(
	ctx context.Context,
	scanID int64,
	status, errorCode string,
	counters db.ScanCounters,
) error {
	var code *string
	if errorCode != "" {
		code = &errorCode
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		affected, err := repository.Finish(ctx, scanID, status, code, counters)
		if err != nil || affected == 0 || status != db.ScanStatusFailed {
			return err
		}

		return repository.CloseTargets(ctx, []int64{scanID}, errorCode)
	})
}

func (s *ScanService) FailStale(ctx context.Context, before time.Time) (int64, error) {
	var failed int64

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		ids, err := repository.FailStale(ctx, before)
		if err != nil {
			return err
		}

		failed = int64(len(ids))

		return repository.CloseTargets(ctx, ids, db.ScanErrorInterrupted)
	})

	return failed, err
}

func (s *ScanService) signal() {
	select {
	case s.created <- struct{}{}:
	default:
	}
}
