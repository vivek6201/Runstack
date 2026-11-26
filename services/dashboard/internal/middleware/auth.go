package middleware

import (
	"github.com/gofiber/fiber/v3"

	"code-runner/dashboard/internal/auth"
	"code-runner/shared/response"
)

// AuthRequired protects dashboard routes via server-side sessions backed by Redis + PostgreSQL.
func AuthRequired(authSvc auth.IAuthService) fiber.Handler {
	return func(c fiber.Ctx) error {
		rawToken := auth.ExtractSessionToken(c)
		if rawToken == "" {
			return response.Unauthorized(c, "missing authentication token")
		}

		sessionCtx, err := authSvc.ValidateSession(c.Context(), rawToken)
		if err != nil {
			return response.Unauthorized(c, "invalid, expired, or superseded session")
		}

		c.Locals("session", sessionCtx)
		c.Locals("claims", sessionCtx.ToClaims())
		c.Locals("user_id", sessionCtx.UserID)
		return c.Next()
	}
}

// GetUserClaims extracts authenticated user claims from the Fiber v3 request context.
func GetUserClaims(c fiber.Ctx) *auth.Claims {
	claims, ok := c.Locals("claims").(*auth.Claims)
	if !ok {
		return nil
	}
	return claims
}
