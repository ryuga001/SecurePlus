package config

import (
	"context"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Postgres struct {
	URL             string
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

func loadPostgres() Postgres {
	maxConns, _ := strconv.Atoi(os.Getenv("POSTGRES_MAX_CONNS"))
	minConns, _ := strconv.Atoi(os.Getenv("POSTGRES_MIN_CONNS"))
	maxConnLifetime, _ := time.ParseDuration(os.Getenv("POSTGRES_MAX_CONN_LIFETIME"))
	maxConnIdleTime, _ := time.ParseDuration(os.Getenv("POSTGRES_MAX_CONN_IDLE_TIME"))

	return Postgres{
		URL:             os.Getenv("DATABASE_URL"),
		MaxConns:        maxConns,
		MinConns:        minConns,
		MaxConnLifetime: maxConnLifetime,
		MaxConnIdleTime: maxConnIdleTime,
	}
}

func (p Postgres) Connect(ctx context.Context, env string) (*gorm.DB, error) {
	level := logger.Warn
	if env != "production" {
		level = logger.Info
	}

	db, err := gorm.Open(postgres.Open(p.URL), &gorm.Config{
		Logger:                 logger.Default.LogMode(level),
		TranslateError:         true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, err
	}

	pool, err := db.DB()
	if err != nil {
		return nil, err
	}

	if p.MaxConns > 0 {
		pool.SetMaxOpenConns(p.MaxConns)
	}
	if p.MinConns > 0 {
		pool.SetMaxIdleConns(p.MinConns)
	}
	if p.MaxConnLifetime > 0 {
		pool.SetConnMaxLifetime(p.MaxConnLifetime)
	}
	if p.MaxConnIdleTime > 0 {
		pool.SetConnMaxIdleTime(p.MaxConnIdleTime)
	}

	if err := pool.PingContext(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return db, nil
}
