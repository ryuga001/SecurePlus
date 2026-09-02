package database

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(ctx context.Context, addr, password, db string) (*redis.Client, error) {
	dbIndex, err := strconv.Atoi(db)
	if err != nil {
		dbIndex = 0
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       dbIndex,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}
