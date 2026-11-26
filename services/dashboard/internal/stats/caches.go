package stats

import (
	"github.com/redis/go-redis/v9"

	"code-runner/shared/cache"
)

// Caches holds typed Redis caches used by the analytics/stats domain.
type Caches struct {
	Overview *cache.TypedCache[DashboardStatsResponse]
	Daily    *cache.TypedCache[[]DailyStatPoint]
}

// NewCaches constructs nil-safe stats caches from a Redis client.
func NewCaches(rdb *redis.Client) *Caches {
	return &Caches{
		Overview: cache.NewTypedCache[DashboardStatsResponse](rdb),
		Daily:    cache.NewTypedCache[[]DailyStatPoint](rdb),
	}
}
