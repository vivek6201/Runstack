package submissions

import "github.com/gofiber/fiber/v3"

// RegisterRoute registers submission history routes onto the router group.
func RegisterRoute(r fiber.Router, handler ISubmissionsHandler, authMiddleware fiber.Handler) {
	subRoutes := r.Group("/submissions", authMiddleware)
	{
		subRoutes.Get("/", handler.ListSubmissionsHandler)
		subRoutes.Get("/:id", handler.GetSubmissionHandler)
	}
}
