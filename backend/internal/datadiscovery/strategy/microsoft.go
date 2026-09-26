package strategy

import (
	"context"
	"encoding/json"
	"strings"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/datadiscovery/provider"
	"dpdp-backend/internal/db"
)

type entraConfig struct {
	TenantID string
	ClientID string
}

type entraCredential struct {
	ClientSecret string `json:"clientSecret"`
}

type EntraStrategy struct {
	client *provider.Client
}

func (EntraStrategy) Type() string { return db.ConfigurationTypeEntra }

func (EntraStrategy) ConnectionKeys() []string { return []string{"tenantId", "clientId"} }

func (EntraStrategy) SecretKeys() []string { return []string{"clientSecret"} }

func (s EntraStrategy) Normalize(in Input) (db.StringMap, []byte, error) {
	tenantID := in.config("tenantId")
	clientID := in.config("clientId")

	if tenantID == "" || clientID == "" {
		return nil, nil, utils.ErrConfigFieldNeeded
	}

	secret := in.secret("clientSecret")
	if secret == "" {
		return nil, nil, utils.ErrSecretNeeded
	}

	credential, err := json.Marshal(entraCredential{ClientSecret: secret})
	if err != nil {
		return nil, nil, err
	}

	return db.StringMap{"tenantId": tenantID, "clientId": clientID}, credential, nil
}

func (s EntraStrategy) Test(ctx context.Context, config db.StringMap, credential []byte) error {
	parsed := entraConfig{TenantID: config["tenantId"], ClientID: config["clientId"]}

	var secret entraCredential
	if err := json.Unmarshal(credential, &secret); err != nil {
		return utils.ErrCredentialUnavailable
	}

	_, err := s.client.EntraToken(ctx, parsed.TenantID, parsed.ClientID, secret.ClientSecret)

	return err
}

type azureStorageConfig struct {
	StorageAccount string
	TenantID       string
	ClientID       string
}

type AzureStorageStrategy struct {
	client *provider.Client
}

func (AzureStorageStrategy) Type() string { return db.ConfigurationTypeAzureStorage }

func (AzureStorageStrategy) ConnectionKeys() []string {
	return []string{"storageAccount", "tenantId", "clientId"}
}

func (AzureStorageStrategy) SecretKeys() []string { return []string{"clientSecret"} }

func (s AzureStorageStrategy) Normalize(in Input) (db.StringMap, []byte, error) {
	account := strings.ToLower(in.config("storageAccount"))
	tenantID := in.config("tenantId")
	clientID := in.config("clientId")

	if account == "" || tenantID == "" || clientID == "" {
		return nil, nil, utils.ErrConfigFieldNeeded
	}

	if len(account) < 3 || len(account) > 24 || !isLowerAlphanumeric(account) {
		return nil, nil, utils.ErrConfigFieldNeeded
	}

	secret := in.secret("clientSecret")
	if secret == "" {
		return nil, nil, utils.ErrSecretNeeded
	}

	credential, err := json.Marshal(entraCredential{ClientSecret: secret})
	if err != nil {
		return nil, nil, err
	}

	config := db.StringMap{
		"storageAccount": account,
		"authMode":       db.AzureAuthModeServicePrincipal,
		"tenantId":       tenantID,
		"clientId":       clientID,
	}

	return config, credential, nil
}

func (s AzureStorageStrategy) Test(ctx context.Context, config db.StringMap, credential []byte) error {
	parsed := azureStorageConfig{
		StorageAccount: config["storageAccount"],
		TenantID:       config["tenantId"],
		ClientID:       config["clientId"],
	}

	var secret entraCredential
	if err := json.Unmarshal(credential, &secret); err != nil {
		return utils.ErrCredentialUnavailable
	}

	token, err := s.client.AzureStorageToken(ctx, parsed.TenantID, parsed.ClientID, secret.ClientSecret)
	if err != nil {
		return err
	}

	return s.client.AzureListContainers(ctx, parsed.StorageAccount, token)
}

func isLowerAlphanumeric(value string) bool {
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return false
		}
	}

	return true
}

func (s EntraStrategy) Refresh(current db.StringMap, in Input) (db.StringMap, error) {
	merged := overlay(current, in, "tenantId", "clientId")

	if merged["tenantId"] == "" || merged["clientId"] == "" {
		return nil, utils.ErrConfigFieldNeeded
	}

	return merged, nil
}

func (s AzureStorageStrategy) Refresh(current db.StringMap, in Input) (db.StringMap, error) {
	merged := overlay(current, in, "storageAccount", "tenantId", "clientId")
	merged["storageAccount"] = strings.ToLower(merged["storageAccount"])
	merged["authMode"] = db.AzureAuthModeServicePrincipal

	account := merged["storageAccount"]
	if account == "" || merged["tenantId"] == "" || merged["clientId"] == "" {
		return nil, utils.ErrConfigFieldNeeded
	}

	if len(account) < 3 || len(account) > 24 || !isLowerAlphanumeric(account) {
		return nil, utils.ErrConfigFieldNeeded
	}

	return merged, nil
}

func (s EntraStrategy) BrowseTargets(
	ctx context.Context,
	sourceType string,
	config db.StringMap,
	credential []byte,
	query provider.TargetQuery,
) (provider.TargetPage, error) {
	mode := ""

	switch sourceType {
	case db.SourceTypeSharePoint:
		mode = provider.GraphSharePoint
	case db.SourceTypeOneDrive:
		mode = provider.GraphOneDrive
	default:
		return provider.TargetPage{}, utils.ErrIncompatibleSource
	}

	var secret entraCredential
	if err := json.Unmarshal(credential, &secret); err != nil || secret.ClientSecret == "" {
		return provider.TargetPage{}, utils.ErrCredentialUnavailable
	}

	source, err := provider.NewGraphSource(ctx, s.client, config["tenantId"], config["clientId"], secret.ClientSecret, mode)
	if err != nil {
		return provider.TargetPage{}, err
	}

	switch {
	case mode == provider.GraphOneDrive:
		return source.Users(ctx, query)
	case query.Parent != "":
		return source.Libraries(ctx, query)
	}

	return source.Sites(ctx, query)
}

func (s AzureStorageStrategy) BrowseTargets(
	ctx context.Context,
	sourceType string,
	config db.StringMap,
	credential []byte,
	query provider.TargetQuery,
) (provider.TargetPage, error) {
	if sourceType != db.SourceTypeAzureBlob {
		return provider.TargetPage{}, utils.ErrIncompatibleSource
	}

	var secret entraCredential
	if err := json.Unmarshal(credential, &secret); err != nil || secret.ClientSecret == "" {
		return provider.TargetPage{}, utils.ErrCredentialUnavailable
	}

	source, err := provider.NewBlobSource(
		ctx, s.client, config["storageAccount"], config["tenantId"], config["clientId"], secret.ClientSecret)
	if err != nil {
		return provider.TargetPage{}, err
	}

	return source.Containers(ctx, query)
}
