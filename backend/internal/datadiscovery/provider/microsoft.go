package provider

import (
	"context"
	"fmt"
	"net/url"
)

const (
	ProviderEntra        = "microsoft-entra"
	ProviderAzureStorage = "azure-storage"
)

func (c *Client) microsoftToken(
	ctx context.Context,
	provider, tenantID, clientID, clientSecret, scope string,
) (string, error) {
	token, err := c.microsoftAccessToken(ctx, provider, tenantID, clientID, clientSecret, scope)

	return token.Value, err
}

func (c *Client) microsoftAccessToken(
	ctx context.Context,
	provider, tenantID, clientID, clientSecret, scope string,
) (Token, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", clientID)
	form.Set("scope", scope)

	body := form.Encode() + "&client_secret=" + clientSecret

	endpoint := fmt.Sprintf(EntraTokenURLFormat, url.PathEscape(tenantID))

	return c.postFormToken(ctx, provider, StageToken, endpoint, body)
}

func (c *Client) EntraToken(ctx context.Context, tenantID, clientID, clientSecret string) (string, error) {
	return c.microsoftToken(ctx, ProviderEntra, tenantID, clientID, clientSecret, GraphScope)
}

func (c *Client) EntraAccessToken(ctx context.Context, tenantID, clientID, clientSecret string) (Token, error) {
	return c.microsoftAccessToken(ctx, ProviderEntra, tenantID, clientID, clientSecret, GraphScope)
}

func (c *Client) AzureStorageToken(ctx context.Context, tenantID, clientID, clientSecret string) (string, error) {
	return c.microsoftToken(ctx, ProviderAzureStorage, tenantID, clientID, clientSecret, AzureStorageScope)
}

func (c *Client) AzureStorageAccessToken(
	ctx context.Context,
	tenantID, clientID, clientSecret string,
) (Token, error) {
	return c.microsoftAccessToken(ctx, ProviderAzureStorage, tenantID, clientID, clientSecret, AzureStorageScope)
}

func (c *Client) AzureListContainers(ctx context.Context, storageAccount, token string) error {
	endpoint := fmt.Sprintf(AzureBlobURLFormat, storageAccount) + "/" + AzureBlobListQuery

	return c.get(ctx, ProviderAzureStorage, StageProbe, endpoint, token, map[string]string{
		"x-ms-version": AzureStorageAPIVersion,
	})
}
