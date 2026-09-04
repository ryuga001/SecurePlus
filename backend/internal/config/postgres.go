package config

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	URL             string
	MaxConns        int32
	MinConns        int32
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
		MaxConns:        int32(maxConns),
		MinConns:        int32(minConns),
		MaxConnLifetime: maxConnLifetime,
		MaxConnIdleTime: maxConnIdleTime,
	}
}

func (p Postgres) Connect(ctx context.Context) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(p.URL)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
