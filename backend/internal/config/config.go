package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	App        App
	Auth       Auth
	Postgres   Postgres
	Redis      Redis
	Mongo      Mongo
	Storage    Storage
	SMTP       SMTP
	SMTPServer SMTPServer
	Relay      Relay
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
		App:        loadApp(),
		Auth:       loadAuth(),
		Postgres:   loadPostgres(),
		Redis:      loadRedis(),
		Mongo:      loadMongo(),
		Storage:    loadStorage(),
		SMTP:       loadSMTP(),
		SMTPServer: loadSMTPServer(),
		Relay:      loadRelay(),
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
