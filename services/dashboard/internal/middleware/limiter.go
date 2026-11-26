package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/redis/go-redis/v9"

	"code-runner/dashboard/internal/config"
	"code-runner/shared/response"
)

func ipKeyGenerator(c fiber.Ctx) string {
	if forwarded := c.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := c.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	return c.IP()
}

// NewRateLimiter constructs a Redis-backed (or in-memory fallback) Fiber v3 rate limiter.
func NewRateLimiter(cfg *config.ConfigVar, rdb *redis.Client, maxReq int, expiration time.Duration, errMsg string) fiber.Handler {
	if cfg != nil && cfg.AppEnv != "production" {
		maxReq *= 10
	}

	var storage fiber.Storage
	if rdb != nil {
		storage = NewRedisStorageAdapter(rdb)
	}

	return limiter.New(limiter.Config{
		Max:          maxReq,
		Expiration:   expiration,
		KeyGenerator: ipKeyGenerator,
		LimitReached: func(c fiber.Ctx) error {
			return response.TooManyRequests(c, errMsg)
		},
		Storage: storage,
	})
}
