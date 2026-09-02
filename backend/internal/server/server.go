package server

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"dpdp-backend/internal/config"
)

type Server struct {
	Config *config.Config
	DB     *pgxpool.Pool
	Redis  *redis.Client
}

func New(cfg *config.Config, db *pgxpool.Pool, redisClient *redis.Client) *Server {
	return &Server{
		Config: cfg,
		DB:     db,
		Redis:  redisClient,
	}
}
