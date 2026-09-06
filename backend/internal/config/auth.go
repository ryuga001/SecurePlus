package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

const (
	RotationGrace  = 10 * time.Second
	CodeTTL        = 10 * time.Minute
	CodeMaxAttempt = 5
	SignupTTL      = 30 * time.Minute
	ResendCooldown = time.Minute
	PrivCacheTTL   = 10 * time.Minute
)

type Auth struct {
	Pepper       string
	CSRFSecret   string
	Issuer       string
	BcryptCost   int
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
	CookieSecure bool
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

	auth := Auth{
		Pepper:       os.Getenv("PASSWORD_PEPPER"),
		CSRFSecret:   os.Getenv("CSRF_SECRET"),
		Issuer:       issuer,
		BcryptCost:   cost,
		AccessTTL:    accessTTL,
		RefreshTTL:   refreshTTL,
		CookieSecure: secure,
	}

	if auth.Pepper == "" {
		log.Fatal("PASSWORD_PEPPER is required")
	}
	if auth.CSRFSecret == "" {
		log.Fatal("CSRF_SECRET is required")
	}

	return auth
}
