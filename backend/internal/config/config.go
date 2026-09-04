package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	App      App
	Postgres Postgres
	Redis    Redis
	Storage  Storage
	SMTP     SMTP
}

type App struct {
	Env             string
	Port            string
	FrontendBaseURL string
	AllowedOrigin   string
}

func Load() *Config {
	godotenv.Load()

	return &Config{
		App:      loadApp(),
		Postgres: loadPostgres(),
		Redis:    loadRedis(),
		Storage:  loadStorage(),
		SMTP:     loadSMTP(),
	}
}

func loadApp() App {
	return App{
		Env:             os.Getenv("APP_ENV"),
		Port:            os.Getenv("APP_PORT"),
		FrontendBaseURL: os.Getenv("FRONTEND_BASE_URL"),
		AllowedOrigin:   os.Getenv("CORS_ALLOWED_ORIGIN"),
	}
}
