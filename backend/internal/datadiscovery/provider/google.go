package provider

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const ProviderGoogle = "google"

var ErrTokenURIUntrusted = errors.New("service account token_uri is not a trusted Google endpoint")

type ServiceAccountKey struct {
	Type         string `json:"type"`
	ProjectID    string `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	ClientID     string `json:"client_id"`
	TokenURI     string `json:"token_uri"`
}

func TrustedTokenURI(tokenURI string) (string, error) {
	if tokenURI == "" || tokenURI == GoogleTokenURL {
		return GoogleTokenURL, nil
	}

	return "", ErrTokenURIUntrusted
}

func (c *Client) GoogleToken(
	ctx context.Context,
	clientEmail, subject, tokenURI, privateKeyPEM string,
) (string, error) {
	token, err := c.GoogleAccessToken(ctx, clientEmail, subject, tokenURI, privateKeyPEM)

	return token.Value, err
}

func (c *Client) GoogleAccessToken(
	ctx context.Context,
	clientEmail, subject, tokenURI, privateKeyPEM string,
) (Token, error) {
	return c.GoogleScopedToken(ctx, clientEmail, subject, tokenURI, privateKeyPEM, GoogleDriveScope)
}

func (c *Client) GoogleScopedToken(
	ctx context.Context,
	clientEmail, subject, tokenURI, privateKeyPEM, scope string,
) (Token, error) {
	endpoint, err := TrustedTokenURI(tokenURI)
	if err != nil {
		return Token{}, err
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKeyPEM))
	if err != nil {
		return Token{}, err
	}

	now := time.Now()

	claims := jwt.MapClaims{
		"iss":   clientEmail,
		"scope": scope,
		"aud":   endpoint,
		"iat":   now.Unix(),
		"exp":   now.Add(GoogleAssertionTTL).Unix(),
	}

	if subject != "" {
		claims["sub"] = subject
	}

	assertion, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		return Token{}, err
	}

	form := url.Values{}
	form.Set("grant_type", GoogleGrantType)
	form.Set("assertion", assertion)

	return c.postFormToken(ctx, ProviderGoogle, StageToken, endpoint, form.Encode())
}

func (c *Client) GoogleDriveAbout(ctx context.Context, token string) error {
	return c.get(ctx, ProviderGoogle, StageProbe, GoogleDriveBaseURL+GoogleDriveAbout, token, nil)
}
