package strategy

import (
	"context"
	"encoding/json"
	"strings"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/datadiscovery/provider"
	"dpdp-backend/internal/db"
)

const serviceAccountKeyType = "service_account"

type googleCredential struct {
	PrivateKeyID string `json:"privateKeyId"`
	PrivateKey   string `json:"privateKey"`
}

type GoogleStrategy struct {
	client *provider.Client
}

func (GoogleStrategy) Type() string { return db.ConfigurationTypeGoogleSA }

func (GoogleStrategy) ConnectionKeys() []string { return []string{"subject", "accessMode"} }

func (GoogleStrategy) SecretKeys() []string { return []string{"serviceAccountKey"} }

func (s GoogleStrategy) Normalize(in Input) (db.StringMap, []byte, error) {
	raw := in.secret("serviceAccountKey")
	if raw == "" {
		return nil, nil, utils.ErrSecretNeeded
	}

	var key provider.ServiceAccountKey
	if err := json.Unmarshal([]byte(raw), &key); err != nil {
		return nil, nil, utils.ErrInvalidServiceAccountKey
	}

	if key.Type != serviceAccountKeyType ||
		key.ProjectID == "" || key.ClientEmail == "" ||
		key.ClientID == "" || key.PrivateKey == "" || key.PrivateKeyID == "" {
		return nil, nil, utils.ErrInvalidServiceAccountKey
	}

	tokenURI, err := provider.TrustedTokenURI(strings.TrimSpace(key.TokenURI))
	if err != nil {
		return nil, nil, utils.ErrInvalidServiceAccountKey
	}

	subject := in.config("subject")

	accessMode := db.GoogleAccessModeSharedDrive
	if subject != "" {
		accessMode = db.GoogleAccessModeDelegation
	}

	credential, err := json.Marshal(googleCredential{
		PrivateKeyID: key.PrivateKeyID,
		PrivateKey:   key.PrivateKey,
	})
	if err != nil {
		return nil, nil, err
	}

	config := db.StringMap{
		"projectId":   key.ProjectID,
		"clientEmail": key.ClientEmail,
		"clientId":    key.ClientID,
		"tokenUri":    tokenURI,
		"accessMode":  accessMode,
	}

	if subject != "" {
		config["subject"] = subject
	}

	return config, credential, nil
}

func (s GoogleStrategy) Test(ctx context.Context, config db.StringMap, credential []byte) error {
	var secret googleCredential
	if err := json.Unmarshal(credential, &secret); err != nil {
		return utils.ErrCredentialUnavailable
	}

	token, err := s.client.GoogleToken(
		ctx, config["clientEmail"], config["subject"], config["tokenUri"], secret.PrivateKey)
	if err != nil {
		return err
	}

	return s.client.GoogleDriveAbout(ctx, token)
}

func (s GoogleStrategy) Refresh(current db.StringMap, in Input) (db.StringMap, error) {
	merged := overlay(current, in, "subject")

	if merged["subject"] == "" {
		delete(merged, "subject")
		merged["accessMode"] = db.GoogleAccessModeSharedDrive
	} else {
		merged["accessMode"] = db.GoogleAccessModeDelegation
	}

	if merged["clientEmail"] == "" || merged["tokenUri"] == "" {
		return nil, utils.ErrInvalidServiceAccountKey
	}

	return merged, nil
}
