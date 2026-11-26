package auth

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"code-runner/shared/cache"
)

// Caches holds session cache state with synchronous eviction to guarantee single active login.
type Caches struct {
	Sessions *cache.TypedCache[SessionContext]
}

// NewCaches constructs nil-safe auth caches from a Redis client.
func NewCaches(rdb *redis.Client) *Caches {
	return &Caches{
		Sessions: cache.NewTypedCache[SessionContext](rdb),
	}
}

func sessionCacheKey(sessionHash string) string {
	return cache.Key("auth", "session", sessionHash)
}

func (c *Caches) GetSession(ctx context.Context, sessionHash string) (*SessionContext, bool) {
	if c == nil || c.Sessions == nil || sessionHash == "" {
		return nil, false
	}
	val, err := c.Sessions.Get(ctx, sessionCacheKey(sessionHash))
	if err != nil || val == nil {
		return nil, false
	}
	return val, true
}

func (c *Caches) SetSession(ctx context.Context, sc *SessionContext) {
	if c == nil || c.Sessions == nil || sc == nil {
		return
	}
	ttl := time.Until(sc.ExpiresAt)
	if ttl <= 0 {
		return
	}
	_ = c.Sessions.Set(ctx, sessionCacheKey(sc.SessionID), sc, ttl)
}

// EvictSessionsSync removes session keys synchronously so evicted logins fail immediately.
func (c *Caches) EvictSessionsSync(ctx context.Context, sessionHashes ...string) {
	if c == nil || c.Sessions == nil || len(sessionHashes) == 0 {
		return
	}
	keys := make([]string, 0, len(sessionHashes))
	for _, hash := range sessionHashes {
		if hash != "" {
			keys = append(keys, sessionCacheKey(hash))
		}
	}
	if len(keys) > 0 {
		_ = c.Sessions.Delete(ctx, keys...)
	}
}
