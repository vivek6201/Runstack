package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"code-runner/dashboard/internal/auth"
	"code-runner/shared/response"
)

// AuthRequired protects dashboard routes using Bearer tokens or HttpOnly cookies.
func AuthRequired(jwtSecret string) fiber.Handler {
	return func(c fiber.Ctx) error {
		var tokenStr string

		authHeader := c.Get("Authorization")
		if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "bearer ") {
			tokenStr = strings.TrimSpace(authHeader[7:])
		}

		if tokenStr == "" {
			tokenStr = c.Cookies("token")
		}

		if tokenStr == "" {
			return response.Unauthorized(c, "missing authentication token")
		}

		claims, err := auth.ParseToken(tokenStr, jwtSecret)
		if err != nil {
			return response.Unauthorized(c, "invalid or expired authentication token")
		}

		c.Locals("claims", claims)
		c.Locals("user_id", claims.UserID)
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
