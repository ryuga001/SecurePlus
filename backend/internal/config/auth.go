package config

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	RotationGrace  = 10 * time.Second
	CodeTTL        = 10 * time.Minute
	CodeMaxAttempt = 5
	SignupTTL      = 30 * time.Minute
	ResendCooldown = time.Minute
	PrivCacheTTL   = 10 * time.Minute

	ProviderTokenTTL = 365 * 24 * time.Hour
)

type Auth struct {
	Pepper         string
	CSRFSecret     string
	Issuer         string
	BcryptCost     int
	AccessTTL      time.Duration
	RefreshTTL     time.Duration
	CookieSecure   bool
	CookieSameSite http.SameSite
}

func loadAuth() Auth {
	cost, _ := strconv.Atoi(os.Getenv("BCRYPT_COST"))
	if cost == 0 {
		cost = 12
	}

	accessTTL, _ := time.ParseDuration(os.Getenv("ACCESS_TTL"))
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}

	refreshTTL, _ := time.ParseDuration(os.Getenv("REFRESH_TTL"))
	if refreshTTL <= 0 {
		refreshTTL = 720 * time.Hour
	}

	issuer := os.Getenv("JWT_ISSUER")
	if issuer == "" {
		issuer = "dpdp"
	}

	secure, _ := strconv.ParseBool(os.Getenv("COOKIE_SECURE"))
	sameSite := cookieSameSite(os.Getenv("COOKIE_SAMESITE"))

	auth := Auth{
		Pepper:         os.Getenv("PASSWORD_PEPPER"),
		CSRFSecret:     os.Getenv("CSRF_SECRET"),
		Issuer:         issuer,
		BcryptCost:     cost,
		AccessTTL:      accessTTL,
		RefreshTTL:     refreshTTL,
		CookieSecure:   secure,
		CookieSameSite: sameSite,
	}

	if auth.Pepper == "" {
		auth.Pepper = "default-pepper"
	}
	if auth.CSRFSecret == "" {
		auth.CSRFSecret = "default-csrf-secret"
	}

	if auth.CookieSameSite == http.SameSiteNoneMode && !auth.CookieSecure {
		log.Println("COOKIE_SAMESITE=none requires COOKIE_SECURE=true, browsers will reject every auth cookie")
	}

	return auth
}

func cookieSameSite(raw string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "none":
		return http.SameSiteNoneMode
	case "strict":
		return http.SameSiteStrictMode
	default:
		return http.SameSiteLaxMode
	}
}
