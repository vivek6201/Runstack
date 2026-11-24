package middleware

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"code-runner/execution-engine/internal/ports"
	"code-runner/shared/cache"
	"code-runner/shared/crypto"
	"code-runner/shared/env"
	"code-runner/shared/models"
	"code-runner/shared/response"
)

// APIKeyAuth validates the incoming X-API-Key header against the KeyValidator port,
// and applies real-time rate limiting via Redis.
func APIKeyAuth(validator ports.KeyValidator, devBypassKey string, cCache *cache.Cache) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rawKey := c.Get("X-API-Key")
		if rawKey == "" {
			// Check Authorization: Bearer <key>
			authHeader := c.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				rawKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if rawKey == "" {
			return response.Unauthorized(c, "missing X-API-Key header or Bearer token")
		}

		// Dev bypass check (only permitted when APP_ENV is not production)
		appEnv := env.Get("APP_ENV", "development")
		if appEnv != "production" && devBypassKey != "" && rawKey == devBypassKey {
			var devUserID uuid.UUID
			if rawUID := c.Get("X-User-ID"); rawUID != "" {
				if parsedUID, err := uuid.Parse(rawUID); err == nil {
					devUserID = parsedUID
				}
			}
			meta := &models.APIKeyCachedMetadata{
				UserID:         devUserID,
				Plan:           models.PlanPro,
				RateLimitRPM:   600,
				MaxConcurrency: 10,
				IsActive:       true,
			}
			c.Locals("auth", meta)

			// Enforce rate limiting on bypass requests to prevent uncontrolled denial of service
			if cCache != nil && meta.RateLimitRPM > 0 {
				window := time.Now().Unix() / 60
				rateLimitKey := cache.Key("ratelimit", "dev_bypass", devUserID.String(), window)
				rpm := int64(meta.RateLimitRPM)
				count, err := cCache.Incr(c.Context(), rateLimitKey, 60*time.Second)
				if err == nil {
					remaining := rpm - count
					if remaining < 0 {
						remaining = 0
					}
					c.Set("X-RateLimit-Limit", strconv.FormatInt(rpm, 10))
					c.Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

					if count > rpm {
						c.Set("Retry-After", "60")
						return response.TooManyRequests(c, "dev bypass rate limit exceeded, retry in 60s")
					}
				}
			}

			return c.Next()
		}

		// Compute SHA-256 hash
		keyHash := crypto.HashAPIKey(rawKey)

		// Fast-path lookup via KeyValidator (Redis cache)
		meta, err := validator.ValidateKey(c.Context(), keyHash)
		if err != nil {
			return response.InternalServerError(c, "failed to validate api key")
		}

		if meta == nil || !meta.IsActive {
			return response.Unauthorized(c, "invalid or revoked api key")
		}

		// Enforce real-time rate limiting per minute window via Redis
		if cCache != nil && meta.RateLimitRPM > 0 {
			window := time.Now().Unix() / 60
			rateLimitKey := cache.Key("ratelimit", meta.APIKeyID.String(), window)
			rpm := int64(meta.RateLimitRPM)
			count, err := cCache.Incr(c.Context(), rateLimitKey, 60*time.Second)
			if err == nil {
				remaining := rpm - count
				if remaining < 0 {
					remaining = 0
				}
				c.Set("X-RateLimit-Limit", strconv.FormatInt(rpm, 10))
				c.Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

				if count > rpm {
					c.Set("Retry-After", "60")
					return response.TooManyRequests(c, "rate limit exceeded, please retry in 60 seconds")
				}
			}
		}

		// Inject metadata into context for downstream handlers
		c.Locals("auth", meta)
		return c.Next()
	}
}
