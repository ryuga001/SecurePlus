package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv             string
	AppPort            string
	DatabaseURL        string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	SMTPHost           string
	SMTPPort           string
	CORSAllowedOrigins []string
}

func Load() *Config {
	godotenv.Load()

	redisDB, _ := strconv.Atoi(os.Getenv("REDIS_DB"))

	return &Config{
		AppEnv:             os.Getenv("APP_ENV"),
		AppPort:            os.Getenv("APP_PORT"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		RedisAddr:          os.Getenv("REDIS_ADDR"),
		RedisPassword:      os.Getenv("REDIS_PASSWORD"),
		RedisDB:            redisDB,
		SMTPHost:           os.Getenv("SMTP_HOST"),
		SMTPPort:           os.Getenv("SMTP_PORT"),
		CORSAllowedOrigins: strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ","),
	}
}
