package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"dpdp-backend/internal/config"
)

const (
	TypeAccess  = "access"
	TypeRefresh = "refresh"

	pepperVersion = "v1"
	codeLength    = 6
)

var codeMax = big.NewInt(1000000)

var dummyHash = "$2a$12$C6UzMDM.H6dfI/f/IKcEe.eS/vDtCn.mYFbYlPvXQMXKxRe3wZ0dW"

type Claims struct {
	jwt.RegisteredClaims
	CustomerID int    `json:"cid"`
	RoleID     *int   `json:"rid,omitempty"`
	Version    int    `json:"ver"`
	Typ        string `json:"typ"`
}

type SecretFunc func(ctx context.Context, customerID int) (string, error)

type Issued struct {
	Token     string
	ID        string
	ExpiresAt time.Time
}

type Subject struct {
	UserID     int
	CustomerID int
	RoleID     *int
	Version    int
}

func NewSalt() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

func prehash(pepper, salt, password string) []byte {
	mac := hmac.New(sha256.New, []byte(pepper))
	mac.Write([]byte(salt))
	mac.Write([]byte{0})
	mac.Write([]byte(password))

	out := make([]byte, base64.RawStdEncoding.EncodedLen(sha256.Size))
	base64.RawStdEncoding.Encode(out, mac.Sum(nil))

	return out
}

func HashPassword(pepper, password, salt string, cost int) (string, error) {
	digest, err := bcrypt.GenerateFromPassword(prehash(pepper, salt, password), cost)
	if err != nil {
		return "", err
	}

	return pepperVersion + "$" + string(digest), nil
}

func VerifyPassword(pepper, stored, salt, password string) error {
	_, digest, found := strings.Cut(stored, "$")
	if !found || digest == "" {
		bcrypt.CompareHashAndPassword([]byte(dummyHash), prehash(pepper, salt, password))
		return ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(digest), prehash(pepper, salt, password)); err != nil {
		return ErrInvalidCredentials
	}

	return nil
}

func BurnTime(pepper, password string) {
	bcrypt.CompareHashAndPassword([]byte(dummyHash), prehash(pepper, "", password))
}

func GenerateCode() (string, error) {
	value, err := rand.Int(rand.Reader, codeMax)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%0*d", codeLength, value.Int64()), nil
}

func EqualCode(stored, candidate string) bool {
	return hmac.Equal([]byte(stored), []byte(strings.TrimSpace(candidate)))
}

func NewToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))

	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func Issue(cfg config.Auth, secret string, subject Subject, typ string) (Issued, error) {
	ttl := cfg.AccessTTL
	if typ == TypeRefresh {
		ttl = cfg.RefreshTTL
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	id, err := NewSalt()
	if err != nil {
		return Issued{}, err
	}

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(subject.UserID),
			Issuer:    cfg.Issuer,
			ID:        id,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		CustomerID: subject.CustomerID,
		RoleID:     subject.RoleID,
		Version:    subject.Version,
		Typ:        typ,
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return Issued{}, err
	}

	return Issued{Token: signed, ID: id, ExpiresAt: expiresAt}, nil
}

func Verify(ctx context.Context, raw, issuer, want string, secretFn SecretFunc) (*Claims, error) {
	claims := &Claims{}

	parsed, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		inner, ok := t.Claims.(*Claims)
		if !ok || inner.CustomerID <= 0 {
			return nil, ErrInvalidToken
		}

		secret, err := secretFn(ctx, inner.CustomerID)
		if err != nil {
			return nil, err
		}

		return []byte(secret), nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
	)
	if errors.Is(err, ErrUnavailable) {
		return nil, ErrUnavailable
	}

	if err != nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}

	if claims.Typ != want || claims.ID == "" || claims.CustomerID <= 0 {
		return nil, ErrInvalidToken
	}

	if _, err := strconv.Atoi(claims.Subject); err != nil {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func UserID(claims *Claims) int {
	id, _ := strconv.Atoi(claims.Subject)
	return id
}
