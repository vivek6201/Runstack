package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// QueueJobs is the Redis list/stream for pending execution jobs.
	QueueJobs = "queue:jobs"
)

// Cache provides high-performance, reusable Redis caching functions:
// dynamic key generation, typed JSON get/set/update, atomic counters,
// cache-aside retrieval, and exact or pattern-based key revocation.
type Cache struct {
	client *redis.Client
}

// New creates a new Cache helper wrapping a redis.Client.
// If client is nil, methods safely degrade to no-op or cache misses without panicking.
func New(client *redis.Client) *Cache {
	return &Cache{client: client}
}

// Client returns the underlying *redis.Client instance.
func (c *Cache) Client() *redis.Client {
	if c == nil {
		return nil
	}
	return c.client
}

// Key dynamically formats a colon-delimited Redis key from arbitrary parts.
// Example: Key("dashboard", "stats", userID) -> "dashboard:stats:<userID>"
func Key(namespace string, parts ...interface{}) string {
	segments := make([]string, 0, len(parts)+1)
	if namespace != "" {
		segments = append(segments, namespace)
	}
	for _, p := range parts {
		segments = append(segments, fmt.Sprintf("%v", p))
	}
	return strings.Join(segments, ":")
}

// BuildKey is an instance method convenience for Key.
func (c *Cache) BuildKey(namespace string, parts ...interface{}) string {
	return Key(namespace, parts...)
}

// Get retrieves a cached JSON value by key and unmarshals it into dest.
// Returns (true, nil) on cache hit, (false, nil) on cache miss (or if cache is nil),
// or (false, err) on network/decoding errors.
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	if c == nil || c.client == nil {
		return false, nil
	}

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, fmt.Errorf("redis get error for key %s: %w", key, err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("failed to unmarshal cached data for key %s: %w", key, err)
	}

	return true, nil
}

// Set serializes value to JSON and stores it with the specified expiration TTL.
// Passing ttl = 0 keeps the key indefinitely without an expiration.
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value for key %s: %w", key, err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set error for key %s: %w", key, err)
	}

	return nil
}

// GetOrSet implements the Cache-Aside pattern.
// If the key exists in cache, it unmarshals into dest and returns nil.
// If it misses, fetch() is called, the result is cached with ttl, and dest is populated.
func (c *Cache) GetOrSet(ctx context.Context, key string, dest interface{}, ttl time.Duration, fetch func() (interface{}, error)) error {
	if c != nil && c.client != nil {
		found, err := c.Get(ctx, key, dest)
		if err == nil && found {
			return nil
		}
	}

	val, err := fetch()
	if err != nil {
		return err
	}

	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to marshal fetched value for key %s: %w", key, err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal fetched data for key %s: %w", key, err)
	}

	if c != nil && c.client != nil {
		_ = c.client.Set(ctx, key, data, ttl).Err()
	}

	return nil
}

// Update retrieves the existing raw JSON bytes (if any), executes updateFn,
// and saves the updated value with the specified ttl.
func (c *Cache) Update(ctx context.Context, key string, ttl time.Duration, updateFn func(existing []byte) (interface{}, error)) error {
	if c == nil || c.client == nil {
		return nil
	}

	var existing []byte
	data, err := c.client.Get(ctx, key).Bytes()
	if err == nil {
		existing = data
	} else if !errors.Is(err, redis.Nil) {
		return fmt.Errorf("redis error reading key for update %s: %w", key, err)
	}

	newVal, err := updateFn(existing)
	if err != nil {
		return err
	}

	return c.Set(ctx, key, newVal, ttl)
}

// Revoke deletes one or more exact keys from the cache.
func (c *Cache) Revoke(ctx context.Context, keys ...string) error {
	if c == nil || c.client == nil || len(keys) == 0 {
		return nil
	}

	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("redis revoke error: %w", err)
	}
	return nil
}

// RevokePattern safely scans and revokes all keys matching a wildcard pattern
// (e.g. "dashboard:submissions:user-uuid:*") without blocking the Redis server.
func (c *Cache) RevokePattern(ctx context.Context, pattern string) error {
	if c == nil || c.client == nil || pattern == "" {
		return nil
	}

	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed scanning pattern %s: %w", pattern, err)
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("failed deleting keys matching %s: %w", pattern, err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// Incr atomically increments a counter by 1 with a sliding TTL window.
// Ideal for rate-limiting and quota tracking.
func (c *Cache) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if c == nil || c.client == nil {
		return 1, nil
	}

	pipe := c.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("redis incr pipeline error on key %s: %w", key, err)
	}

	return incr.Val(), nil
}

// Decr atomically decrements a counter by 1.
func (c *Cache) Decr(ctx context.Context, key string) (int64, error) {
	if c == nil || c.client == nil {
		return 0, nil
	}
	return c.client.Decr(ctx, key).Result()
}

// Exists checks whether a specific key exists in Redis.
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	if c == nil || c.client == nil {
		return false, nil
	}
	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// -----------------------------------------------------------------------------
// Generic Cache Wrapper Functions (hits cache first, on miss calls real db func)
// -----------------------------------------------------------------------------

// Fetch is a generic wrapper function that hits the Redis cache first.
// On a cache hit, it unmarshals into T and immediately returns without touching the DB.
// On a cache miss (or if cache is offline), it invokes the real dbCall() function,
// saves the result in Redis with the specified ttl, and returns the result.
func Fetch[T any](ctx context.Context, c *Cache, key string, ttl time.Duration, dbCall func() (T, error)) (T, error) {
	if c != nil && c.client != nil && key != "" {
		var cached T
		found, err := c.Get(ctx, key, &cached)
		if err == nil && found {
			return cached, nil
		}
	}

	result, err := dbCall()
	if err != nil {
		var zero T
		return zero, err
	}

	if c != nil && c.client != nil && key != "" {
		_ = c.Set(ctx, key, result, ttl)
	}

	return result, nil
}

// WithCache is an alias for Fetch, providing a descriptive wrapper name.
func WithCache[T any](ctx context.Context, c *Cache, key string, ttl time.Duration, dbCall func() (T, error)) (T, error) {
	return Fetch(ctx, c, key, ttl, dbCall)
}

// PaginatedResult wraps a slice of items and a total count for cache storage.
type PaginatedResult[T any] struct {
	Items T     `json:"items"`
	Total int64 `json:"total"`
}

// FetchPaginated is a generic wrapper for paginated queries returning (items T, total int64, err error).
// It checks cache first, and on miss calls dbCall(), caches the items and total count, and returns them.
func FetchPaginated[T any](ctx context.Context, c *Cache, key string, ttl time.Duration, dbCall func() (T, int64, error)) (T, int64, error) {
	res, err := Fetch(ctx, c, key, ttl, func() (PaginatedResult[T], error) {
		items, total, err := dbCall()
		if err != nil {
			var zero T
			return PaginatedResult[T]{Items: zero, Total: 0}, err
		}
		return PaginatedResult[T]{Items: items, Total: total}, nil
	})
	if err != nil {
		var zero T
		return zero, 0, err
	}
	return res.Items, res.Total, nil
}
