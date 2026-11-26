package stats

import "github.com/gofiber/fiber/v3"

// RegisterRoute registers dashboard analytics routes onto the router group.
func RegisterRoute(r fiber.Router, handler IStatsHandler, authMiddleware fiber.Handler) {
	statsRoutes := r.Group("/stats", authMiddleware)
	{
		statsRoutes.Get("/", handler.GetStatsHandler)
		statsRoutes.Get("/daily", handler.GetDailyStatsHandler)
	}
}
