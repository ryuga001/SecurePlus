package datadiscovery

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	repo "dpdp-backend/internal/admin/repositories/datadiscovery"
	"dpdp-backend/internal/admin/utils"
	appcrypto "dpdp-backend/internal/crypto"
	"dpdp-backend/internal/datadiscovery/provider"
	"dpdp-backend/internal/datadiscovery/strategy"
	"dpdp-backend/internal/db"
)

type ConfigurationDetail struct {
	Configuration db.DataDiscoveryConfiguration
	HasCredential bool
	PolicyCount   int64
}

type ConfigurationListing struct {
	Items    []ConfigurationDetail
	Page     int
	PageSize int
	Total    int64
}

type ConfigurationInput struct {
	Name              string
	Description       string
	ConfigurationType string
	Status            string
	Config            map[string]string
	Secret            map[string]string
}

type TestInput struct {
	ConfigurationID int
	ConfigurationInput
}

type ConfigurationService struct {
	db          *gorm.DB
	repo        *repo.ConfigurationRepository
	registry    *strategy.ConfigurationRegistry
	box         *appcrypto.SecretBox
	testTimeout time.Duration
}

func NewConfigurationService(
	database *gorm.DB,
	repository *repo.ConfigurationRepository,
	registry *strategy.ConfigurationRegistry,
	box *appcrypto.SecretBox,
	testTimeout time.Duration,
) *ConfigurationService {
	return &ConfigurationService{
		db:          database,
		repo:        repository,
		registry:    registry,
		box:         box,
		testTimeout: testTimeout,
	}
}

func (s *ConfigurationService) List(
	ctx context.Context,
	customerID int,
	params repo.ConfigurationListParams,
) (ConfigurationListing, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	rows, total, err := s.repo.List(ctx, customerID, params)
	if err != nil {
		return ConfigurationListing{}, err
	}

	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	counts, err := s.repo.PolicyCountsFor(ctx, customerID, ids)
	if err != nil {
		return ConfigurationListing{}, err
	}

	byConfiguration := make(map[int]int64, len(counts))
	for _, count := range counts {
		byConfiguration[count.ConfigurationID] = count.Total
	}

	items := make([]ConfigurationDetail, 0, len(rows))
	for _, row := range rows {
		items = append(items, ConfigurationDetail{
			Configuration: row,
			PolicyCount:   byConfiguration[row.ID],
		})
	}

	return ConfigurationListing{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *ConfigurationService) Get(
	ctx context.Context,
	customerID, id int,
) (ConfigurationDetail, error) {
	row, err := s.repo.Find(ctx, customerID, id)
	if err != nil {
		return ConfigurationDetail{}, configurationError(err, "")
	}

	return s.detail(ctx, s.repo, customerID, row)
}

func (s *ConfigurationService) Create(
	ctx context.Context,
	customerID int,
	in ConfigurationInput,
) (ConfigurationDetail, error) {
	input, err := NormalizeConfigurationInput(in)
	if err != nil {
		return ConfigurationDetail{}, err
	}

	selected, ok := s.registry.For(input.ConfigurationType)
	if !ok {
		return ConfigurationDetail{}, utils.ErrInvalidConfigurationType
	}

	config, credential, err := selected.Normalize(strategy.Input{
		Config: input.Config,
		Secret: input.Secret,
	})
	if err != nil {
		return ConfigurationDetail{}, err
	}

	if err := s.test(ctx, customerID, selected, config, credential); err != nil {
		return ConfigurationDetail{}, err
	}

	row := db.DataDiscoveryConfiguration{
		CustomerID:        customerID,
		Name:              input.Name,
		Description:       optionalText(input.Description),
		ConfigurationType: input.ConfigurationType,
		Config:            config,
		Status:            input.Status,
		LastTestedAt:      time.Now(),
	}

	var detail ConfigurationDetail

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		if err := repository.Insert(ctx, &row); err != nil {
			return err
		}

		sealed, err := s.box.Seal(credential,
			appcrypto.CredentialAAD(s.box.Version(), customerID, row.ID))
		if err != nil {
			return err
		}

		if err := repository.UpsertCredential(ctx, customerID, row.ID, sealed, s.box.Version()); err != nil {
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
		return ConfigurationDetail{}, configurationError(err, input.Name)
	}

	return detail, nil
}

func (s *ConfigurationService) Update(
	ctx context.Context,
	customerID, id int,
	in ConfigurationInput,
) (ConfigurationDetail, error) {
	current, err := s.repo.Find(ctx, customerID, id)
	if err != nil {
		return ConfigurationDetail{}, configurationError(err, "")
	}

	if in.ConfigurationType != "" && in.ConfigurationType != current.ConfigurationType {
		return ConfigurationDetail{}, utils.ErrConfigurationTypeImmutable
	}

	in.ConfigurationType = current.ConfigurationType

	input, err := NormalizeConfigurationInput(in)
	if err != nil {
		return ConfigurationDetail{}, err
	}

	selected, ok := s.registry.For(current.ConfigurationType)
	if !ok {
		return ConfigurationDetail{}, utils.ErrInvalidConfigurationType
	}

	config, credential, err := s.resolveUpdate(ctx, customerID, id, current, selected, input)
	if err != nil {
		return ConfigurationDetail{}, err
	}

	updates := map[string]any{
		"name":        input.Name,
		"description": optionalText(input.Description),
		"status":      input.Status,
		"config":      config,
		"updated_at":  time.Now(),
	}

	var sealed []byte

	if credential != nil {
		if err := s.test(ctx, customerID, selected, config, credential); err != nil {
			return ConfigurationDetail{}, err
		}

		updates["last_tested_at"] = time.Now()

		sealed, err = s.box.Seal(credential,
			appcrypto.CredentialAAD(s.box.Version(), customerID, id))
		if err != nil {
			return ConfigurationDetail{}, err
		}
	}

	var detail ConfigurationDetail

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		if _, err := repository.Lock(ctx, customerID, id); err != nil {
			return err
		}

		if _, err := repository.Update(ctx, customerID, id, updates); err != nil {
			return err
		}

		if sealed != nil {
			if err := repository.UpsertCredential(ctx, customerID, id, sealed, s.box.Version()); err != nil {
				return err
			}
		}

		updated, err := repository.Find(ctx, customerID, id)
		if err != nil {
			return err
		}

		detail, err = s.detail(ctx, repository, customerID, updated)

		return err
	})

	if err != nil {
		return ConfigurationDetail{}, configurationError(err, input.Name)
	}

	return detail, nil
}

func (s *ConfigurationService) resolveUpdate(
	ctx context.Context,
	customerID, id int,
	current db.DataDiscoveryConfiguration,
	selected strategy.ConfigurationStrategy,
	input ConfigurationInput,
) (db.StringMap, []byte, error) {
	secretProvided := false
	for _, key := range selected.SecretKeys() {
		if strings.TrimSpace(input.Secret[key]) != "" {
			secretProvided = true
			break
		}
	}

	if secretProvided {
		return selected.Normalize(strategy.Input{Config: input.Config, Secret: input.Secret})
	}

	config, err := selected.Refresh(current.Config, strategy.Input{Config: input.Config})
	if err != nil {
		return nil, nil, err
	}

	for _, key := range selected.ConnectionKeys() {
		if current.Config[key] != config[key] {
			stored, err := s.openCredential(ctx, customerID, id)
			if err != nil {
				return nil, nil, err
			}

			return config, stored, nil
		}
	}

	return config, nil, nil
}

func (s *ConfigurationService) Delete(ctx context.Context, customerID, id int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		if _, err := repository.Lock(ctx, customerID, id); err != nil {
			return configurationError(err, "")
		}

		count, err := repository.CountPolicies(ctx, customerID, id)
		if err != nil {
			return err
		}
		if count > 0 {
			return utils.ErrDiscoveryConfigInUse
		}

		affected, err := repository.Delete(ctx, customerID, id)
		if err != nil {
			return deleteConfigurationError(err)
		}
		if affected == 0 {
			return utils.ErrDiscoveryConfigNotFound
		}

		return nil
	})
}

func (s *ConfigurationService) Test(ctx context.Context, customerID int, in TestInput) error {
	configurationType := in.ConfigurationType

	if in.ConfigurationID > 0 {
		current, err := s.repo.Find(ctx, customerID, in.ConfigurationID)
		if err != nil {
			return configurationError(err, "")
		}

		configurationType = current.ConfigurationType
	}

	selected, ok := s.registry.For(configurationType)
	if !ok {
		return utils.ErrInvalidConfigurationType
	}

	secretProvided := false
	for _, key := range selected.SecretKeys() {
		if strings.TrimSpace(in.Secret[key]) != "" {
			secretProvided = true
			break
		}
	}

	if secretProvided {
		config, credential, err := selected.Normalize(
			strategy.Input{Config: in.Config, Secret: in.Secret})
		if err != nil {
			return err
		}

		return s.test(ctx, customerID, selected, config, credential)
	}

	if in.ConfigurationID == 0 {
		return utils.ErrSecretNeeded
	}

	current, err := s.repo.Find(ctx, customerID, in.ConfigurationID)
	if err != nil {
		return configurationError(err, "")
	}

	config, err := selected.Refresh(current.Config, strategy.Input{Config: in.Config})
	if err != nil {
		return err
	}

	credential, err := s.openCredential(ctx, customerID, in.ConfigurationID)
	if err != nil {
		return err
	}

	return s.test(ctx, customerID, selected, config, credential)
}

func (s *ConfigurationService) Capabilities(ctx context.Context) ([]db.DataDiscoverySourceCapability, error) {
	return s.repo.Capabilities(ctx)
}

func (s *ConfigurationService) test(
	ctx context.Context,
	customerID int,
	selected strategy.ConfigurationStrategy,
	config db.StringMap,
	credential []byte,
) error {
	testCtx, cancel := context.WithTimeout(ctx, s.testTimeout)
	defer cancel()

	err := selected.Test(testCtx, config, credential)
	if err == nil {
		return nil
	}

	var providerErr *provider.Error
	if !errors.As(err, &providerErr) {
		return err
	}

	slog.WarnContext(ctx, "data discovery connection test failed",
		"customer_id", customerID,
		"configuration_type", selected.Type(),
		"provider", providerErr.Provider,
		"stage", providerErr.Stage,
		"reason", providerErr.Reason,
		"http_status", providerErr.HTTPStatus,
		"provider_code", providerErr.Code(),
		"request_id", providerErr.RequestID(),
	)

	if providerErr.Reason == provider.ReasonUnavailable ||
		providerErr.Reason == provider.ReasonRateLimited {
		return utils.ErrProviderUnavailable
	}

	return utils.ConnectionFailed(providerErr.Reason)
}

func (s *ConfigurationService) openCredential(
	ctx context.Context,
	customerID, configurationID int,
) ([]byte, error) {
	stored, err := s.repo.LoadSealed(ctx, customerID, configurationID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, utils.ErrSecretNeeded
		}

		return nil, err
	}

	credential, err := s.box.Open(stored.Secret,
		appcrypto.CredentialAAD(stored.KeyVersion, customerID, configurationID))
	if err != nil {
		return nil, utils.ErrCredentialUnavailable
	}

	return credential, nil
}

func (s *ConfigurationService) detail(
	ctx context.Context,
	repository *repo.ConfigurationRepository,
	customerID int,
	row db.DataDiscoveryConfiguration,
) (ConfigurationDetail, error) {
	hasCredential, err := repository.HasCredential(ctx, customerID, row.ID)
	if err != nil {
		return ConfigurationDetail{}, err
	}

	count, err := repository.CountPolicies(ctx, customerID, row.ID)
	if err != nil {
		return ConfigurationDetail{}, err
	}

	return ConfigurationDetail{Configuration: row, HasCredential: hasCredential, PolicyCount: count}, nil
}

func optionalText(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func configurationError(err error, name string) error {
	switch {
	case db.IsNotFound(err):
		return utils.ErrDiscoveryConfigNotFound
	case db.IsDuplicate(err):
		return utils.Taken(utils.ErrDiscoveryConfigNameTaken, name)
	}

	return err
}

func deleteConfigurationError(err error) error {
	if db.IsMissingReference(err) {
		return utils.ErrDiscoveryConfigInUse
	}

	return configurationError(err, "")
}

func NormalizeConfigurationInput(in ConfigurationInput) (ConfigurationInput, error) {
	name := utils.NormalizeName(in.Name)
	if length := utf8.RuneCountInString(name); length < 2 || length > 100 {
		return ConfigurationInput{}, utils.ErrDiscoveryNameNeeded
	}

	configurationType := strings.ToUpper(strings.TrimSpace(in.ConfigurationType))
	if !validConfigurationType(configurationType) {
		return ConfigurationInput{}, utils.ErrInvalidConfigurationType
	}

	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = db.DiscoveryStatusActive
	}
	if status != db.DiscoveryStatusActive && status != db.DiscoveryStatusInactive {
		return ConfigurationInput{}, utils.ErrConfigFieldNeeded
	}

	description := strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(description) > 255 {
		return ConfigurationInput{}, utils.ErrConfigFieldNeeded
	}

	config := in.Config
	if config == nil {
		config = map[string]string{}
	}

	secret := in.Secret
	if secret == nil {
		secret = map[string]string{}
	}

	return ConfigurationInput{
		Name:              name,
		Description:       description,
		ConfigurationType: configurationType,
		Status:            status,
		Config:            config,
		Secret:            secret,
	}, nil
}

func validConfigurationType(value string) bool {
	switch value {
	case db.ConfigurationTypeEntra, db.ConfigurationTypeAzureStorage,
		db.ConfigurationTypeGoogleSA, db.ConfigurationTypeAWSIAM:
		return true
	}

	return false
}
