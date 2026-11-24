package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"code-runner/dashboard/internal/auth"
	"code-runner/dashboard/internal/keys"
	"code-runner/dashboard/internal/playground"
	"code-runner/dashboard/internal/stats"
	"code-runner/dashboard/internal/submissions"
)

type RouterConfig struct {
	AuthHandler        *auth.Handler
	KeysHandler        *keys.Handler
	SubmissionsHandler *submissions.Handler
	StatsHandler       *stats.Handler
	PlaygroundHandler  *playground.Handler
	JWTSecret          string
}

// RegisterRoutes sets up all public and protected HTTP routes on the Fiber application
func RegisterRoutes(app *fiber.App, cfg RouterConfig) {
	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"service":   "dashboard-api",
			"timestamp": time.Now().UTC(),
		})
	})

	apiGroup := app.Group("/api")

	// Public Auth Routes
	apiGroup.Post("/auth/register", cfg.AuthHandler.Register)
	apiGroup.Post("/auth/login", cfg.AuthHandler.Login)
	apiGroup.Post("/auth/logout", cfg.AuthHandler.Logout)

	// Protected Routes (JWT required)
	protected := apiGroup.Group("", auth.JWTMiddleware(cfg.JWTSecret))
	protected.Get("/auth/me", cfg.AuthHandler.Me)

	// API Keys Management
	protected.Post("/keys", cfg.KeysHandler.Create)
	protected.Get("/keys", cfg.KeysHandler.List)
	protected.Delete("/keys/:id", cfg.KeysHandler.Revoke)

	// Submissions & Execution History
	protected.Get("/submissions", cfg.SubmissionsHandler.List)
	protected.Get("/submissions/:id", cfg.SubmissionsHandler.Get)

	// Analytics & Dashboard Stats
	protected.Get("/stats", cfg.StatsHandler.GetStats)
	protected.Get("/stats/daily", cfg.StatsHandler.GetDaily)

	// Playground Direct Execution
	protected.Post("/playground/execute", cfg.PlaygroundHandler.Execute)
}
