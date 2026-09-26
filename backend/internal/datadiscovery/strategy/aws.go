package strategy

import (
	"context"
	"encoding/json"
	"strings"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/datadiscovery/provider"
	"dpdp-backend/internal/db"
)

type awsCredential struct {
	SecretAccessKey string `json:"secretAccessKey"`
}

type AWSStrategy struct {
	client *provider.Client
}

func (AWSStrategy) Type() string { return db.ConfigurationTypeAWSIAM }

func (AWSStrategy) ConnectionKeys() []string { return []string{"accessKeyId", "region"} }

func (AWSStrategy) SecretKeys() []string { return []string{"secretAccessKey"} }

func (s AWSStrategy) Normalize(in Input) (db.StringMap, []byte, error) {
	accessKeyID := strings.ToUpper(in.config("accessKeyId"))
	region := strings.ToLower(in.config("region"))

	if accessKeyID == "" || region == "" {
		return nil, nil, utils.ErrConfigFieldNeeded
	}

	if len(accessKeyID) < 16 || len(accessKeyID) > 128 || len(region) < 2 || len(region) > 30 {
		return nil, nil, utils.ErrConfigFieldNeeded
	}

	secret := in.secret("secretAccessKey")
	if secret == "" {
		return nil, nil, utils.ErrSecretNeeded
	}

	credential, err := json.Marshal(awsCredential{SecretAccessKey: secret})
	if err != nil {
		return nil, nil, err
	}

	return db.StringMap{"accessKeyId": accessKeyID, "region": region}, credential, nil
}

func (s AWSStrategy) Test(ctx context.Context, config db.StringMap, credential []byte) error {
	var secret awsCredential
	if err := json.Unmarshal(credential, &secret); err != nil {
		return utils.ErrCredentialUnavailable
	}

	return s.client.AWSCallerIdentity(
		ctx, config["accessKeyId"], secret.SecretAccessKey, config["region"])
}

func (s AWSStrategy) Refresh(current db.StringMap, in Input) (db.StringMap, error) {
	merged := overlay(current, in, "accessKeyId", "region")
	merged["accessKeyId"] = strings.ToUpper(merged["accessKeyId"])
	merged["region"] = strings.ToLower(merged["region"])

	if merged["accessKeyId"] == "" || merged["region"] == "" {
		return nil, utils.ErrConfigFieldNeeded
	}

	return merged, nil
}

func (s AWSStrategy) BrowseTargets(
	ctx context.Context,
	sourceType string,
	config db.StringMap,
	credential []byte,
	query provider.TargetQuery,
) (provider.TargetPage, error) {
	if sourceType != db.SourceTypeAWSS3 {
		return provider.TargetPage{}, utils.ErrIncompatibleSource
	}

	var secret awsCredential
	if err := json.Unmarshal(credential, &secret); err != nil || secret.SecretAccessKey == "" {
		return provider.TargetPage{}, utils.ErrCredentialUnavailable
	}

	source, err := provider.NewS3Source(ctx, s.client, config["accessKeyId"], secret.SecretAccessKey, config["region"])
	if err != nil {
		return provider.TargetPage{}, err
	}

	return source.Buckets(ctx, query)
}
