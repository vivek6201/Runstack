package submissions

import (
	"github.com/redis/go-redis/v9"

	"code-runner/shared/cache"
)

// Caches holds typed Redis caches used by the submissions domain.
type Caches struct {
	List   *cache.TypedCache[PaginatedSubmissions]
	Detail *cache.TypedCache[SubmissionDetailResponse]
}

// NewCaches constructs nil-safe submission caches from a Redis client.
func NewCaches(rdb *redis.Client) *Caches {
	return &Caches{
		List:   cache.NewTypedCache[PaginatedSubmissions](rdb),
		Detail: cache.NewTypedCache[SubmissionDetailResponse](rdb),
	}
}
