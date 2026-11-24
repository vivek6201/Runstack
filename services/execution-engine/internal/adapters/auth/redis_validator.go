package auth

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"code-runner/shared/cache"
	"code-runner/shared/models"
)

// RedisKeyValidator implements ports.KeyValidator using sub-millisecond Redis cache.
type RedisKeyValidator struct {
	cache *cache.Cache
}

func NewRedisKeyValidator(client *redis.Client) *RedisKeyValidator {
	return &RedisKeyValidator{cache: cache.New(client)}
}

func (v *RedisKeyValidator) ValidateKey(ctx context.Context, keyHash string) (*models.APIKeyCachedMetadata, error) {
	cacheKey := cache.Key("apikey", "cache", keyHash)
	var meta models.APIKeyCachedMetadata
	found, err := v.cache.Get(ctx, cacheKey, &meta)
	if err != nil {
		return nil, fmt.Errorf("failed to check api key cache: %w", err)
	}

	if !found || !meta.IsActive {
		return nil, nil // Key not found or marked inactive/revoked
	}

	return &meta, nil
}
