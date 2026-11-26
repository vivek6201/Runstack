package server

import (
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"code-runner/execution-engine/internal/adapters/auth"
	"code-runner/execution-engine/internal/adapters/queue"
	"code-runner/execution-engine/internal/api"
	"code-runner/execution-engine/internal/config"
	"code-runner/execution-engine/internal/middleware"
	"code-runner/shared/cache"
)

// SetupRoutes wires adapters, auth middleware, handlers, and API routes.
func SetupRoutes(app *fiber.App, redisClient *redis.Client, queueAdapter *queue.RedisQueueAdapter, cfg *config.ConfigVar) {
	authValidator := auth.NewRedisKeyValidator(redisClient)
	redisCache := cache.New(redisClient)
	handler := api.NewHandler(queueAdapter, queueAdapter)
	authMiddleware := middleware.APIKeyAuth(authValidator, cfg, redisCache)

	api.RegisterRoute(app, handler, authMiddleware)
}
