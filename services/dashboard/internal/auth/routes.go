package auth

import "github.com/gofiber/fiber/v3"

// RegisterRoute registers authentication and session lifecycle routes onto the router group.
func RegisterRoute(r fiber.Router, handler IAuthHandler, authMiddleware fiber.Handler) {
	authRoutes := r.Group("/auth")
	{
		authRoutes.Post("/register", handler.RegisterHandler)
		authRoutes.Post("/login", handler.LoginHandler)
		authRoutes.Post("/logout", handler.LogoutHandler)
		authRoutes.Get("/me", authMiddleware, handler.MeHandler)
		authRoutes.Get("/session", authMiddleware, handler.SessionHandler)
	}
}
