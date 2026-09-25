package strategy

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/golang-jwt/jwt/v5"

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

	source, err := provider.NewBlobSource(ctx, client, account, tenantID, clientID, secret.ClientSecret)
	if err != nil {
		return nil, err
	}

	return source, nil
}

func connectSharePoint(
	ctx context.Context,
	client *provider.Client,
	config db.StringMap,
	credential []byte,
) (Source, error) {
	return connectGraph(ctx, client, config, credential, provider.GraphSharePoint)
}

func connectOneDrive(
	ctx context.Context,
	client *provider.Client,
	config db.StringMap,
	credential []byte,
) (Source, error) {
	return connectGraph(ctx, client, config, credential, provider.GraphOneDrive)
}

func connectGraph(
	ctx context.Context,
	client *provider.Client,
	config db.StringMap,
	credential []byte,
	mode string,
) (Source, error) {
	var secret entraCredential
	if err := json.Unmarshal(credential, &secret); err != nil || secret.ClientSecret == "" {
		return nil, utils.ErrCredentialUnavailable
	}

	tenantID := config["tenantId"]
	clientID := config["clientId"]

	if tenantID == "" || clientID == "" {
		return nil, utils.ErrConfigFieldNeeded
	}

	source, err := provider.NewGraphSource(ctx, client, tenantID, clientID, secret.ClientSecret, mode)
	if err != nil {
		return nil, err
	}

	return source, nil
}

func connectGoogleDrive(
	ctx context.Context,
	client *provider.Client,
	config db.StringMap,
	credential []byte,
) (Source, error) {
	var secret googleCredential
	if err := json.Unmarshal(credential, &secret); err != nil || secret.PrivateKey == "" {
		return nil, utils.ErrCredentialUnavailable
	}

	if _, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(secret.PrivateKey)); err != nil {
		return nil, utils.ErrCredentialUnavailable
	}

	clientEmail := config["clientEmail"]
	if clientEmail == "" {
		return nil, utils.ErrConfigFieldNeeded
	}

	source, err := provider.NewDriveSource(
		ctx, client, clientEmail, config["subject"], config["tokenUri"], secret.PrivateKey)
	if err != nil {
		return nil, err
	}

	return source, nil
}

func connectAWSS3(
	ctx context.Context,
	client *provider.Client,
	config db.StringMap,
	credential []byte,
) (Source, error) {
	var secret awsCredential
	if err := json.Unmarshal(credential, &secret); err != nil || secret.SecretAccessKey == "" {
		return nil, utils.ErrCredentialUnavailable
	}

	accessKeyID := config["accessKeyId"]
	region := config["region"]

	if accessKeyID == "" || region == "" {
		return nil, utils.ErrConfigFieldNeeded
	}

	source, err := provider.NewS3Source(ctx, client, accessKeyID, secret.SecretAccessKey, region)
	if err != nil {
		return nil, err
	}

	return source, nil
}
