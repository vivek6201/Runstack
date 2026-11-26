package api

import "github.com/gofiber/fiber/v3"

// RegisterRoute mounts public and protected execution routes onto the Fiber app.
func RegisterRoute(app *fiber.App, handler IHandler, authMiddleware fiber.Handler) {
	app.Get("/health", handler.HealthHandler)

	v1 := app.Group("/v1", authMiddleware)
	{
		v1.Post("/submit", handler.SubmitHandler)
		v1.Get("/status/:id", handler.StatusHandler)
	}
}
