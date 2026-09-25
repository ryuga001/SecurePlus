package strategy

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/datadiscovery/provider"
	"dpdp-backend/internal/db"
)

var ErrSourceUnsupported = errors.New("this source type cannot be scanned yet")

type Source interface {
	List(ctx context.Context, target string, emit func(provider.File) error) error
	Open(ctx context.Context, file provider.File) (io.ReadCloser, error)
}

type connectFunc func(
	ctx context.Context,
	client *provider.Client,
	config db.StringMap,
	credential []byte,
) (Source, error)

func connectAzureBlob(
	ctx context.Context,
	client *provider.Client,
	config db.StringMap,
	credential []byte,
) (Source, error) {
	var secret entraCredential
	if err := json.Unmarshal(credential, &secret); err != nil || secret.ClientSecret == "" {
		return nil, utils.ErrCredentialUnavailable
	}

	account := config["storageAccount"]
	tenantID := config["tenantId"]
	clientID := config["clientId"]

	if account == "" || tenantID == "" || clientID == "" {
		return nil, utils.ErrConfigFieldNeeded
	}

	return provider.NewBlobSource(ctx, client, account, tenantID, clientID, secret.ClientSecret)
}
