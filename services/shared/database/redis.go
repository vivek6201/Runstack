package database

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// ConnectRedis creates a Redis client and verifies connectivity with a short timeout ping.
func ConnectRedis(addr, password string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return rdb, err
	}

	return rdb, nil
}
