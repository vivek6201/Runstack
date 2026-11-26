package profile

import (
	"github.com/redis/go-redis/v9"

	"code-runner/shared/cache"
)

// Caches holds typed Redis caches for the developer profile domain.
type Caches struct {
	ByUserID *cache.TypedCache[ProfileResponse]
}

// NewCaches constructs nil-safe profile caches from a Redis client.
func NewCaches(rdb *redis.Client) *Caches {
	return &Caches{
		ByUserID: cache.NewTypedCache[ProfileResponse](rdb),
	}
}
