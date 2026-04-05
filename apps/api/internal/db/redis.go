package db

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewRedis(redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}

	rdb := redis.NewClient(opts)

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("unable to ping redis: %w", err)
	}

	return rdb, nil
}
