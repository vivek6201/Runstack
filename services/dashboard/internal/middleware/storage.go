package middleware

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStorageAdapter adapts *redis.Client to Fiber v3's Storage interface for rate limiting.
type RedisStorageAdapter struct {
	rdb *redis.Client
}

func NewRedisStorageAdapter(rdb *redis.Client) *RedisStorageAdapter {
	return &RedisStorageAdapter{rdb: rdb}
}

func (s *RedisStorageAdapter) Get(key string) ([]byte, error) {
	return s.GetWithContext(context.Background(), key)
}

func (s *RedisStorageAdapter) GetWithContext(ctx context.Context, key string) ([]byte, error) {
	val, err := s.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

func (s *RedisStorageAdapter) Set(key string, val []byte, exp time.Duration) error {
	return s.SetWithContext(context.Background(), key, val, exp)
}

func (s *RedisStorageAdapter) SetWithContext(ctx context.Context, key string, val []byte, exp time.Duration) error {
	return s.rdb.Set(ctx, key, val, exp).Err()
}

func (s *RedisStorageAdapter) Delete(key string) error {
	return s.DeleteWithContext(context.Background(), key)
}

func (s *RedisStorageAdapter) DeleteWithContext(ctx context.Context, key string) error {
	return s.rdb.Del(ctx, key).Err()
}

func (s *RedisStorageAdapter) Reset() error {
	return s.ResetWithContext(context.Background())
}

func (s *RedisStorageAdapter) ResetWithContext(ctx context.Context) error {
	return nil
}

func (s *RedisStorageAdapter) Close() error {
	return nil
}
