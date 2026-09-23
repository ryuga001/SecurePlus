package datadiscovery

import (
	"context"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	repo "dpdp-backend/internal/admin/repositories/datadiscovery"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/datadiscovery/strategy"
	"dpdp-backend/internal/db"
)

type PolicyDetail struct {
	Policy        db.DataDiscoveryPolicy
	Configuration utils.ReferenceItem
	Rules         []repo.PolicyReference
}

type PolicySummary struct {
	Policy        db.DataDiscoveryPolicy
	Configuration utils.ReferenceItem
	RuleCount     int
}

type PolicyListing struct {
	Items    []PolicySummary
	Page     int
	PageSize int
	Total    int64
}

type PolicyInput struct {
	Name            string
	Description     string
	SourceType      string
	Status          string
	ConfigurationID int
	TargetList      []string
	FileTypes       []string
	RuleIDs         []int
}

type PolicyService struct {
	db       *gorm.DB
	repo     *repo.PolicyRepository
	registry *strategy.PolicyRegistry
}

func NewPolicyService(
	database *gorm.DB,
	repository *repo.PolicyRepository,
	registry *strategy.PolicyRegistry,
) *PolicyService {
	return &PolicyService{db: database, repo: repository, registry: registry}
}

func (s *PolicyService) List(
	ctx context.Context,
	customerID int,
	params repo.PolicyListParams,
) (PolicyListing, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	rows, total, err := s.repo.List(ctx, customerID, params)
	if err != nil {
		return PolicyListing{}, err
	}

	policyIDs := make([]int, 0, len(rows))
	configurationIDs := make([]int, 0, len(rows))

	for _, row := range rows {
		policyIDs = append(policyIDs, row.ID)
		configurationIDs = append(configurationIDs, row.ConfigurationID)
	}

	rules, err := s.repo.RulesFor(ctx, customerID, policyIDs)
	if err != nil {
		return PolicyListing{}, err
	}

	configurations, err := s.repo.ConfigurationsFor(ctx, customerID, configurationIDs)
	if err != nil {
		return PolicyListing{}, err
	}

	ruleCounts := make(map[int]int, len(rules))
	for _, rule := range rules {
		ruleCounts[rule.PolicyID]++
	}

	byID := make(map[int]utils.ReferenceItem, len(configurations))
	for _, configuration := range configurations {
		byID[configuration.ID] = configuration
	}

	items := make([]PolicySummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, PolicySummary{
			Policy:        row,
			Configuration: byID[row.ConfigurationID],
			RuleCount:     ruleCounts[row.ID],
		})
	}

	return PolicyListing{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *PolicyService) Get(ctx context.Context, customerID, id int) (PolicyDetail, error) {
	row, err := s.repo.Find(ctx, customerID, id)
	if err != nil {
		return PolicyDetail{}, policyError(err, "")
	}

	return s.detail(ctx, s.repo, customerID, row)
}

func (s *PolicyService) Create(
	ctx context.Context,
	customerID int,
	in PolicyInput,
) (PolicyDetail, error) {
	input, selected, err := s.prepare(in)
	if err != nil {
		return PolicyDetail{}, err
	}

	row := db.DataDiscoveryPolicy{
		CustomerID:      customerID,
		Name:            input.Name,
		Description:     optionalText(input.Description),
		ConfigurationID: input.ConfigurationID,
		SourceType:      input.SourceType,
		TargetList:      db.StringList(input.TargetList),
		FileTypes:       db.StringList(input.FileTypes),
		Status:          input.Status,
	}

	var detail PolicyDetail

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		configurationType, err := ensureConfiguration(
			ctx, repository, customerID, input.ConfigurationID, selected)
		if err != nil {
			return err
		}

		if err := ensureRules(ctx, repository, customerID, input.RuleIDs); err != nil {
			return err
		}

		if err := ensureFileTypes(ctx, repository, input.FileTypes); err != nil {
			return err
		}

		row.ConfigurationType = configurationType

		if err := repository.Insert(ctx, &row); err != nil {
			return err
		}

		if err := repository.ReplaceRules(ctx, customerID, row.ID, input.RuleIDs); err != nil {
			return err
		}

		created, err := repository.Find(ctx, customerID, row.ID)
		if err != nil {
			return err
		}

		detail, err = s.detail(ctx, repository, customerID, created)

		return err
	})

	if err != nil {
		return PolicyDetail{}, policyError(err, input.Name)
	}

	return detail, nil
}

func (s *PolicyService) Update(
	ctx context.Context,
	customerID, id int,
	in PolicyInput,
) (PolicyDetail, error) {
	input, selected, err := s.prepare(in)
	if err != nil {
		return PolicyDetail{}, err
	}

	var detail PolicyDetail

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		current, err := repository.Lock(ctx, customerID, id)
		if err != nil {
			return err
		}

		configurationType, err := ensureConfiguration(
			ctx, repository, customerID, input.ConfigurationID, selected)
		if err != nil {
			return err
		}

		if err := ensureRules(ctx, repository, customerID, input.RuleIDs); err != nil {
			return err
		}

		if err := ensureFileTypes(ctx, repository, input.FileTypes); err != nil {
			return err
		}

		updates := map[string]any{
			"name":               input.Name,
			"description":        optionalText(input.Description),
			"configuration_id":   input.ConfigurationID,
			"configuration_type": configurationType,
			"source_type":        input.SourceType,
			"target_list":        db.StringList(input.TargetList),
			"file_types":         db.StringList(input.FileTypes),
			"status":             input.Status,
			"updated_at":         time.Now(),
		}

		if _, err := repository.Update(ctx, customerID, id, updates); err != nil {
			return err
		}

		if err := repository.ReplaceRules(ctx, customerID, id, input.RuleIDs); err != nil {
			return err
		}

		current.Name = input.Name
		current.Description = optionalText(input.Description)
		current.ConfigurationID = input.ConfigurationID
		current.ConfigurationType = configurationType
		current.SourceType = input.SourceType
		current.TargetList = db.StringList(input.TargetList)
		current.FileTypes = db.StringList(input.FileTypes)
		current.Status = input.Status

		detail, err = s.detail(ctx, repository, customerID, current)

		return err
	})

	if err != nil {
		return PolicyDetail{}, policyError(err, input.Name)
	}

	return detail, nil
}

func (s *PolicyService) Delete(ctx context.Context, customerID, id int) error {
	affected, err := s.repo.Delete(ctx, customerID, id)
	if err != nil {
		return policyError(err, "")
	}
	if affected == 0 {
		return utils.ErrDiscoveryPolicyNotFound
	}

	return nil
}

func (s *PolicyService) prepare(in PolicyInput) (PolicyInput, strategy.PolicyStrategy, error) {
	input, err := NormalizePolicyInput(in)
	if err != nil {
		return PolicyInput{}, nil, err
	}

	selected, ok := s.registry.For(input.SourceType)
	if !ok {
		return PolicyInput{}, nil, utils.ErrInvalidSourceType
	}

	targets, err := selected.NormalizeTargets(input.TargetList)
	if err != nil {
		return PolicyInput{}, nil, err
	}

	input.TargetList = targets

	return input, selected, nil
}

func (s *PolicyService) detail(
	ctx context.Context,
	repository *repo.PolicyRepository,
	customerID int,
	row db.DataDiscoveryPolicy,
) (PolicyDetail, error) {
	rules, err := repository.RulesFor(ctx, customerID, []int{row.ID})
	if err != nil {
		return PolicyDetail{}, err
	}

	configurations, err := repository.ConfigurationsFor(ctx, customerID, []int{row.ConfigurationID})
	if err != nil {
		return PolicyDetail{}, err
	}

	configuration := utils.ReferenceItem{}
	if len(configurations) > 0 {
		configuration = configurations[0]
	}

	return PolicyDetail{Policy: row, Configuration: configuration, Rules: rules}, nil
}

func ensureConfiguration(
	ctx context.Context,
	repository *repo.PolicyRepository,
	customerID, configurationID int,
	selected strategy.PolicyStrategy,
) (string, error) {
	configuration, err := repository.LockOwnedConfiguration(ctx, customerID, configurationID)
	if err != nil {
		if db.IsNotFound(err) {
			return "", utils.ErrUnknownConfiguration
		}

		return "", err
	}

	if configuration.ConfigurationType != selected.ConfigurationType() {
		return "", utils.ErrIncompatibleSource
	}

	if configuration.Status != db.DiscoveryStatusActive {
		return "", utils.ErrUnknownConfiguration
	}

	return configuration.ConfigurationType, nil
}

func ensureRules(
	ctx context.Context,
	repository *repo.PolicyRepository,
	customerID int,
	ids []int,
) error {
	found, err := repository.LockOwnedRuleIDs(ctx, customerID, ids)
	if err != nil {
		return err
	}
	if len(found) != len(ids) {
		return utils.ErrUnknownRule
	}

	return nil
}

func ensureFileTypes(
	ctx context.Context,
	repository *repo.PolicyRepository,
	extensions []string,
) error {
	if len(extensions) == 0 {
		return nil
	}

	found, err := repository.KnownExtensions(ctx, extensions)
	if err != nil {
		return err
	}
	if len(found) != len(extensions) {
		return utils.ErrUnknownFileType
	}

	return nil
}

func policyError(err error, name string) error {
	switch {
	case db.IsNotFound(err):
		return utils.ErrDiscoveryPolicyNotFound
	case db.IsDuplicate(err):
		return utils.Taken(utils.ErrDiscoveryPolicyNameTaken, name)
	case db.IsMissingReference(err):
		return utils.ErrIncompatibleSource
	}

	return err
}

func NormalizePolicyInput(in PolicyInput) (PolicyInput, error) {
	name := utils.NormalizeName(in.Name)
	if length := utf8.RuneCountInString(name); length < 2 || length > 100 {
		return PolicyInput{}, utils.ErrDiscoveryNameNeeded
	}

	description := strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(description) > 255 {
		return PolicyInput{}, utils.ErrConfigFieldNeeded
	}

	sourceType := strings.ToUpper(strings.TrimSpace(in.SourceType))
	if !validSourceType(sourceType) {
		return PolicyInput{}, utils.ErrInvalidSourceType
	}

	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = db.DiscoveryStatusActive
	}
	if status != db.DiscoveryStatusActive && status != db.DiscoveryStatusInactive {
		return PolicyInput{}, utils.ErrConfigFieldNeeded
	}

	if in.ConfigurationID < 1 {
		return PolicyInput{}, utils.ErrUnknownConfiguration
	}

	ruleIDs := utils.NormalizeIDs(in.RuleIDs)
	if len(ruleIDs) == 0 {
		return PolicyInput{}, utils.ErrRulesNeeded
	}
	if len(ruleIDs) > utils.MaxBatchIDs {
		return PolicyInput{}, utils.ErrTooManyItems
	}

	fileTypes, err := normalizeFileTypes(in.FileTypes)
	if err != nil {
		return PolicyInput{}, err
	}

	return PolicyInput{
		Name:            name,
		Description:     description,
		SourceType:      sourceType,
		Status:          status,
		ConfigurationID: in.ConfigurationID,
		TargetList:      in.TargetList,
		FileTypes:       fileTypes,
		RuleIDs:         ruleIDs,
	}, nil
}

func normalizeFileTypes(raw []string) ([]string, error) {
	seen := make(map[string]bool, len(raw))
	values := make([]string, 0, len(raw))

	for _, entry := range raw {
		value := strings.ToLower(strings.TrimSpace(entry))
		value = strings.TrimPrefix(value, ".")

		if value == "" || len(value) > 20 || strings.ContainsAny(value, " .\t/\\") {
			return nil, utils.ErrUnknownFileType
		}
		if seen[value] {
			continue
		}

		seen[value] = true
		values = append(values, value)
	}

	if len(values) > utils.MaxBatchIDs {
		return nil, utils.ErrTooManyItems
	}

	slices.Sort(values)

	return values, nil
}

func validSourceType(value string) bool {
	switch value {
	case db.SourceTypeSharePoint, db.SourceTypeOneDrive, db.SourceTypeAzureBlob,
		db.SourceTypeGoogleDrive, db.SourceTypeAWSS3:
		return true
	}

	return false
}
