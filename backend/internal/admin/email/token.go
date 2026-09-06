package email

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/db"
)

const TokenType = "provider_access"

type AccessClaims struct {
	jwt.RegisteredClaims
	CustomerID int    `json:"cid"`
	ConfigID   int    `json:"config_id"`
	Domain     string `json:"domain"`
	Typ        string `json:"typ"`
}

func IssueAccessToken(secret, issuer string, row db.EmailProviderConfiguration, ttl time.Duration) (string, time.Time, error) {
	if !ValidDomain(row.Domain) {
		return "", time.Time{}, ErrInvalidDomain
	}

	id, err := auth.NewSalt()
	if err != nil {
		return "", time.Time{}, err
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	claims := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        id,
			Issuer:    issuer,
			Subject:   strconv.Itoa(row.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		CustomerID: row.CustomerID,
		ConfigID:   row.ID,
		Domain:     row.Domain,
		Typ:        TokenType,
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

func ParseAccessToken(raw, secret, issuer string) (*AccessClaims, error) {
	claims := &AccessClaims{}

	parsed, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(issuer), jwt.WithExpirationRequired())

	if err != nil || !parsed.Valid {
		return nil, errors.New(TokenType + " token is invalid")
	}
	if claims.Typ != TokenType || claims.Domain == "" {
		return nil, errors.New(TokenType + " token is invalid")
	}

	return claims, nil
}
