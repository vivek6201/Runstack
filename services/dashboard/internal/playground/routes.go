package playground

import "github.com/gofiber/fiber/v3"

// RegisterRoute registers interactive playground execution routes onto the router group.
func RegisterRoute(r fiber.Router, handler IPlaygroundHandler, authMiddleware fiber.Handler) {
	playRoutes := r.Group("/playground", authMiddleware)
	{
		playRoutes.Post("/execute", handler.ExecuteHandler)
	}
}
