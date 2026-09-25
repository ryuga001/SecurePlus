package strategy

import (
	"context"
	"strings"

	"dpdp-backend/internal/datadiscovery/provider"
	"dpdp-backend/internal/db"
)

type Input struct {
	Config map[string]string
	Secret map[string]string
}

func (in Input) config(key string) string {
	return strings.TrimSpace(in.Config[key])
}

func (in Input) secret(key string) string {
	return strings.TrimSpace(in.Secret[key])
}

type ConfigurationStrategy interface {
	Type() string
	Normalize(in Input) (db.StringMap, []byte, error)
	Refresh(current db.StringMap, in Input) (db.StringMap, error)
	Test(ctx context.Context, config db.StringMap, credential []byte) error
	ConnectionKeys() []string
	SecretKeys() []string
}

func overlay(current db.StringMap, in Input, keys ...string) db.StringMap {
	merged := make(db.StringMap, len(current)+len(keys))
	for key, value := range current {
		merged[key] = value
	}

	for _, key := range keys {
		if _, present := in.Config[key]; present {
			merged[key] = in.config(key)
		}
	}

	return merged
}

type PolicyStrategy interface {
	SourceType() string
	ConfigurationType() string
	NormalizeTargets(targets []string) ([]string, error)
	Scannable() bool
	Connect(ctx context.Context, client *provider.Client, config db.StringMap, credential []byte) (Source, error)
}

type ConfigurationRegistry struct {
	strategies map[string]ConfigurationStrategy
}

func NewConfigurationRegistry(items ...ConfigurationStrategy) *ConfigurationRegistry {
	registry := make(map[string]ConfigurationStrategy, len(items))
	for _, item := range items {
		registry[item.Type()] = item
	}

	return &ConfigurationRegistry{strategies: registry}
}

func DefaultConfigurationRegistry(client *provider.Client) *ConfigurationRegistry {
	return NewConfigurationRegistry(
		EntraStrategy{client: client},
		AzureStorageStrategy{client: client},
		GoogleStrategy{client: client},
		AWSStrategy{client: client},
	)
}

func (r *ConfigurationRegistry) For(configurationType string) (ConfigurationStrategy, bool) {
	strategy, ok := r.strategies[configurationType]

	return strategy, ok
}

type PolicyRegistry struct {
	strategies map[string]PolicyStrategy
}

func NewPolicyRegistry(items ...PolicyStrategy) *PolicyRegistry {
	registry := make(map[string]PolicyStrategy, len(items))
	for _, item := range items {
		registry[item.SourceType()] = item
	}

	return &PolicyRegistry{strategies: registry}
}

func DefaultPolicyRegistry() *PolicyRegistry {
	return NewPolicyRegistry(
		sharePointStrategy(),
		oneDriveStrategy(),
		azureBlobStrategy(),
		googleDriveStrategy(),
		awsS3Strategy(),
	)
}

func (r *PolicyRegistry) For(sourceType string) (PolicyStrategy, bool) {
	strategy, ok := r.strategies[sourceType]

	return strategy, ok
}
