package keys

import (
	"github.com/redis/go-redis/v9"

	"code-runner/shared/cache"
	"code-runner/shared/models"
)

// Caches holds typed Redis caches used by the API keys domain.
type Caches struct {
	Keys        *cache.TypedCache[[]APIKeyItemResponse]
	KeyMetadata *cache.TypedCache[models.APIKeyCachedMetadata]
}

// NewCaches constructs nil-safe key caches from a Redis client.
func NewCaches(rdb *redis.Client) *Caches {
	return &Caches{
		Keys:        cache.NewTypedCache[[]APIKeyItemResponse](rdb),
		KeyMetadata: cache.NewTypedCache[models.APIKeyCachedMetadata](rdb),
	}
}
