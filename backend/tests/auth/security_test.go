package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/config"
)

func issued(t *testing.T, typ string) string {
	t.Helper()

	cfg := config.Auth{Issuer: "dpdp", AccessTTL: time.Minute, RefreshTTL: time.Hour}

	token, err := auth.Issue(cfg, "tenant-secret", auth.Subject{UserID: 7, CustomerID: 3, Version: 1}, typ)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	return token.Token
}

func secret(value string, err error) auth.SecretFunc {
	return func(context.Context, int) (string, error) {
		return value, err
	}
}

func TestVerifyAcceptsValidRefreshToken(t *testing.T) {
	claims, err := auth.Verify(context.Background(), issued(t, auth.TypeRefresh), "dpdp", auth.TypeRefresh, secret("tenant-secret", nil))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}

	if auth.UserID(claims) != 7 || claims.CustomerID != 3 {
		t.Fatalf("claims = %+v", claims)
	}
}

func TestVerifyReportsUnavailableSecretStore(t *testing.T) {
	_, err := auth.Verify(context.Background(), issued(t, auth.TypeRefresh), "dpdp", auth.TypeRefresh, secret("", auth.ErrUnavailable))

	if !errors.Is(err, auth.ErrUnavailable) || errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("err = %v, a store outage must not look like an invalid token", err)
	}
}

func TestVerifyRejectsInvalidTokens(t *testing.T) {
	cases := map[string]struct {
		raw    string
		want   string
		secret auth.SecretFunc
	}{
		"wrong secret":   {issued(t, auth.TypeRefresh), auth.TypeRefresh, secret("other", nil)},
		"wrong type":     {issued(t, auth.TypeAccess), auth.TypeRefresh, secret("tenant-secret", nil)},
		"unknown tenant": {issued(t, auth.TypeRefresh), auth.TypeRefresh, secret("", auth.ErrInvalidToken)},
		"garbage":        {"not-a-token", auth.TypeRefresh, secret("tenant-secret", nil)},
	}

	for name, tc := range cases {
		if _, err := auth.Verify(context.Background(), tc.raw, "dpdp", tc.want, tc.secret); !errors.Is(err, auth.ErrInvalidToken) {
			t.Fatalf("%s: err = %v", name, err)
		}
	}
}
