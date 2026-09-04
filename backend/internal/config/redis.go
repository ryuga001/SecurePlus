package config

import (
	"context"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	Addr     string
	Password string
	DB       int
}

func loadRedis() Redis {
	db, _ := strconv.Atoi(os.Getenv("REDIS_DB"))

	return Redis{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	}
}

func (r Redis) Connect(ctx context.Context) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     r.Addr,
		Password: r.Password,
		DB:       r.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}
