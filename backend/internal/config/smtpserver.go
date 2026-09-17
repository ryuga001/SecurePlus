package config

import (
	"os"
	"strconv"
	"time"
)

type SMTPServer struct {
	Addr            string
	MaxSize         int64
	MaxRecipients   int
	MaxConnections  int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

func loadSMTPServer() SMTPServer {
	return SMTPServer{
		Addr:            envString("SMTP_SERVER_ADDR", ":2525"),
		MaxSize:         int64(envInt("SMTP_MAX_SIZE", 10*1024*1024)),
		MaxRecipients:   envInt("SMTP_MAX_RECIPIENTS", 100),
		MaxConnections:  envInt("SMTP_MAX_CONNECTIONS", 100),
		ReadTimeout:     envDuration("SMTP_READ_TIMEOUT", time.Minute),
		WriteTimeout:    envDuration("SMTP_WRITE_TIMEOUT", time.Minute),
		ShutdownTimeout: envDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
	}
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}

func envBool(key string, fallback bool) bool {
	value, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		return fallback
	}

	return value
}
