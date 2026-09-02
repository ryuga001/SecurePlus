package main

import (
	"context"
	"log"

	"dpdp-backend/internal/config"
	"dpdp-backend/internal/database"
	"dpdp-backend/internal/server"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	db, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	redisClient, err := database.NewRedisClient(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	defer redisClient.Close()

	srv := server.New(cfg, db, redisClient)
	router := srv.RegisterRoutes()

	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
