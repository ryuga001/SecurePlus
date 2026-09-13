package config

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	URL string
}

func loadRedis() Redis {
	return Redis{URL: os.Getenv("REDIS_URL")}
}

func (r Redis) Connect(ctx context.Context) (*redis.Client, error) {
	options, err := redis.ParseURL(r.URL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(options)

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}
