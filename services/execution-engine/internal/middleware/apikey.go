package middleware

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"code-runner/execution-engine/internal/config"
	"code-runner/execution-engine/internal/ports"
	"code-runner/shared/cache"
	"code-runner/shared/crypto"
	"code-runner/shared/models"
	"code-runner/shared/response"
)

// APIKeyAuth validates incoming X-API-Key headers or Bearer tokens and enforces rate limits via Redis.
func APIKeyAuth(validator ports.KeyValidator, cfg *config.ConfigVar, cCache *cache.Cache) fiber.Handler {
	appEnv, devBypassKey := resolveAuthPolicy(cfg)

	return func(c fiber.Ctx) error {
		rawKey := extractAPIKey(c)
		if rawKey == "" {
			return response.Unauthorized(c, "missing X-API-Key header or Bearer token")
		}

		if appEnv != "production" && devBypassKey != "" && rawKey == devBypassKey {
			return authenticateDevBypass(c, cCache)
		}

		keyHash := crypto.HashAPIKey(rawKey)
		meta, err := validator.ValidateKey(c.Context(), keyHash)
		if err != nil {
			return response.InternalServerError(c, "failed to validate api key")
		}
		if meta == nil || !meta.IsActive {
			return response.Unauthorized(c, "invalid or revoked api key")
		}

		if err := enforceRateLimit(c, cCache, meta.APIKeyID.String(), meta.RateLimitRPM, "rate limit exceeded, please retry in 60 seconds"); err != nil {
			return err
		}

		c.Locals("auth", meta)
		return c.Next()
	}
}

func resolveAuthPolicy(cfg *config.ConfigVar) (appEnv, devBypassKey string) {
	if cfg == nil {
		return "development", ""
	}
	if cfg.AppEnv == "" {
		return "development", cfg.DevBypassKey
	}
	return cfg.AppEnv, cfg.DevBypassKey
}

func authenticateDevBypass(c fiber.Ctx, cCache *cache.Cache) error {
	devUserID := parseHeaderUUID(c, "X-User-ID")
	meta := &models.APIKeyCachedMetadata{
		UserID:         devUserID,
		Plan:           models.PlanPro,
		RateLimitRPM:   600,
		MaxConcurrency: 10,
		IsActive:       true,
	}
	c.Locals("auth", meta)

	subject := "dev_bypass:" + devUserID.String()
	if err := enforceRateLimit(c, cCache, subject, meta.RateLimitRPM, "dev bypass rate limit exceeded, retry in 60s"); err != nil {
		return err
	}
	return c.Next()
}

func parseHeaderUUID(c fiber.Ctx, header string) uuid.UUID {
	rawUID := c.Get(header)
	if rawUID == "" {
		return uuid.Nil
	}
	parsedUID, err := uuid.Parse(rawUID)
	if err != nil {
		return uuid.Nil
	}
	return parsedUID
}

func extractAPIKey(c fiber.Ctx) string {
	if rawKey := c.Get("X-API-Key"); rawKey != "" {
		return rawKey
	}
	authHeader := c.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}

func enforceRateLimit(c fiber.Ctx, cCache *cache.Cache, subject string, rpmLimit int, exceededMsg string) error {
	if cCache == nil || rpmLimit <= 0 {
		return nil
	}

	window := time.Now().Unix() / 60
	rateLimitKey := cache.Key("ratelimit", subject, window)
	rpm := int64(rpmLimit)
	count, err := cCache.Incr(c.Context(), rateLimitKey, 60*time.Second)
	if err != nil {
		return nil
	}

	remaining := rpm - count
	if remaining < 0 {
		remaining = 0
	}
	c.Set("X-RateLimit-Limit", strconv.FormatInt(rpm, 10))
	c.Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

	if count > rpm {
		c.Set("Retry-After", "60")
		return response.TooManyRequests(c, exceededMsg)
	}
	return nil
}
