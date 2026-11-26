package keys

import "github.com/gofiber/fiber/v3"

// RegisterRoute registers API key management endpoints onto the router group.
func RegisterRoute(r fiber.Router, handler IKeysHandler, authMiddleware fiber.Handler) {
	keysRoutes := r.Group("/keys", authMiddleware)
	{
		keysRoutes.Post("/", handler.CreateKeyHandler)
		keysRoutes.Get("/", handler.ListKeysHandler)
		keysRoutes.Delete("/:id", handler.RevokeKeyHandler)
	}
}
