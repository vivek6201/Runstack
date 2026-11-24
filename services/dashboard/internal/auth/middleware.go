package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"code-runner/shared/response"
)

func JWTMiddleware(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var tokenStr string

		// 1. Check Authorization Header: Bearer <token>
		authHeader := c.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			tokenStr = strings.TrimSpace(authHeader[7:])
		}

		// 2. Fallback to HttpOnly cookie: "token"
		if tokenStr == "" {
			tokenStr = c.Cookies("token")
		}

		if tokenStr == "" {
			return response.Unauthorized(c, "missing authentication token")
		}

		claims, err := ParseToken(tokenStr, secret)
		if err != nil {
			return response.Unauthorized(c, "invalid or expired authentication token")
		}

		// Set claims in request context locals
		c.Locals("claims", claims)
		c.Locals("user_id", claims.UserID)
		return c.Next()
	}
}

// GetUserClaims extracts the authenticated Claims from Fiber context locals.
func GetUserClaims(c *fiber.Ctx) *Claims {
	claims, ok := c.Locals("claims").(*Claims)
	if !ok {
		return nil
	}
	return claims
}
