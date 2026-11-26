package server

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"code-runner/dashboard/internal/auth"
	"code-runner/dashboard/internal/config"
	"code-runner/dashboard/internal/keys"
	"code-runner/dashboard/internal/middleware"
	"code-runner/dashboard/internal/playground"
	"code-runner/dashboard/internal/stats"
	"code-runner/dashboard/internal/submissions"
)

// SetupRoutes wires repositories, caches, services, handlers, and domain route groups.
func SetupRoutes(app *fiber.App, db *gorm.DB, rdb *redis.Client, cfg *config.ConfigVar) {
	app.Get("/health", func(ctx fiber.Ctx) error {
		return ctx.JSON(fiber.Map{
			"status":    "healthy",
			"service":   "dashboard-api",
			"timestamp": time.Now().UTC(),
		})
	})

	// ── Auth ──────────────────────────────────────────────────────────────────
	authRepo := auth.NewAuthRepository(db)
	authService := auth.NewAuthService(authRepo, cfg)
	authHandler := auth.NewAuthHandler(authService, cfg)
	authMiddleware := middleware.AuthRequired(cfg.JWTSecret)

	// ── Keys ──────────────────────────────────────────────────────────────────
	keysRepo := keys.NewKeysRepository(db)
	keysService := keys.NewKeysService(keysRepo, keys.NewCaches(rdb))
	keysHandler := keys.NewKeysHandler(keysService)

	// ── Submissions & Stats ───────────────────────────────────────────────────
	subRepo := submissions.NewSubmissionsRepository(db)
	subService := submissions.NewSubmissionsService(subRepo, submissions.NewCaches(rdb))
	subHandler := submissions.NewSubmissionsHandler(subService)

	statsService := stats.NewStatsService(subRepo, rdb, stats.NewCaches(rdb))
	statsHandler := stats.NewStatsHandler(statsService)

	// ── Playground ────────────────────────────────────────────────────────────
	playService := playground.NewPlaygroundService(subRepo, playground.NewCaches(rdb), cfg)
	playHandler := playground.NewPlaygroundHandler(playService)

	// ── Rate Limiters ─────────────────────────────────────────────────────────
	authLimiter := middleware.NewRateLimiter(cfg, rdb, 20, 1*time.Minute, "too many authentication attempts, please try again shortly")
	generalLimiter := middleware.NewRateLimiter(cfg, rdb, 120, 1*time.Minute, "rate limit exceeded, please slow down")

	// ── Register Modular Routes ───────────────────────────────────────────────
	apiGroup := app.Group("/api")
	{
		authGroup := apiGroup.Group("", authLimiter)
		generalGroup := apiGroup.Group("", generalLimiter)

		auth.RegisterRoute(authGroup, authHandler, authMiddleware)
		keys.RegisterRoute(generalGroup, keysHandler, authMiddleware)
		submissions.RegisterRoute(generalGroup, subHandler, authMiddleware)
		stats.RegisterRoute(generalGroup, statsHandler, authMiddleware)
		playground.RegisterRoute(generalGroup, playHandler, authMiddleware)
	}
}
