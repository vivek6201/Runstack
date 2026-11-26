package profile

import "github.com/gofiber/fiber/v3"

// RegisterRoute registers developer profile endpoints onto the router group.
func RegisterRoute(r fiber.Router, handler IProfileHandler, authMiddleware fiber.Handler) {
	profileRoutes := r.Group("/profile", authMiddleware)
	{
		profileRoutes.Get("/", handler.GetProfileHandler)
		profileRoutes.Patch("/", handler.UpdateProfileHandler)
	}
}
