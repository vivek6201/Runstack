package routes

import (
	"github.com/gofiber/fiber/v2"

	"code-runner/execution-engine/internal/api"
	"code-runner/execution-engine/internal/middleware"
	"code-runner/execution-engine/internal/ports"
	"code-runner/shared/cache"
)

type RouterConfig struct {
	Handler       *api.Handler
	AuthValidator ports.KeyValidator
	DevBypassKey  string
	Cache         *cache.Cache
}

// RegisterRoutes registers public and protected API routes for the Execution Engine
func RegisterRoutes(app *fiber.App, cfg RouterConfig) {
	// Public Health Route
	app.Get("/health", cfg.Handler.Health)

	// Protected v1 API Routes with Authentication & Rate-Limiting
	v1 := app.Group("/v1", middleware.APIKeyAuth(cfg.AuthValidator, cfg.DevBypassKey, cfg.Cache))
	v1.Post("/submit", cfg.Handler.Submit)
	v1.Get("/status/:id", cfg.Handler.Status)
}
