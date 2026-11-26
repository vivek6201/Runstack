package playground

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"code-runner/shared/cache"
)

// Caches encapsulates Redis quota updates and async dashboard cache invalidation after runs.
type Caches struct {
	raw *cache.Cache
}

// NewCaches constructs nil-safe cache helpers for the playground domain.
func NewCaches(rdb *redis.Client) *Caches {
	if rdb == nil {
		return &Caches{}
	}
	return &Caches{raw: cache.New(rdb)}
}

// RecordRunAndInvalidateAsync increments the monthly run quota and invalidates cached stats/lists asynchronously.
func (c *Caches) RecordRunAndInvalidateAsync(userID uuid.UUID) {
	if c == nil || c.raw == nil || c.raw.Client() == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		now := time.Now().UTC()
		ym := fmt.Sprintf("%04d-%02d", now.Year(), now.Month())
		uid := userID.String()

		_ = c.raw.Client().Incr(ctx, cache.Key("user", "quota", uid, ym, "runs")).Err()
		_ = c.raw.Revoke(ctx, cache.Key("dashboard", "stats", uid))
		_ = c.raw.RevokePattern(ctx, cache.Key("dashboard", "submissions", uid, "list")+"*")
		_ = c.raw.RevokePattern(ctx, cache.Key("dashboard", "stats", "daily", uid)+"*")
	}()
}
