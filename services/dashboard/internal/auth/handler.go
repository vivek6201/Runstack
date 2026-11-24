package auth

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"

	"code-runner/shared/dto"
	"code-runner/shared/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request payload")
	}

	resp, err := h.service.Register(req)
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			return response.Error(c, fiber.StatusConflict, err.Error())
		}
		return response.BadRequest(c, err.Error())
	}

	setAuthCookie(c, resp.Token, resp.ExpiresAt)
	return response.Created(c, resp)
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request payload")
	}

	resp, err := h.service.Login(req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return response.Unauthorized(c, err.Error())
		}
		return response.InternalServerError(c, err.Error())
	}

	setAuthCookie(c, resp.Token, resp.ExpiresAt)
	return response.Success(c, resp)
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		SameSite: "Lax",
		Path:     "/",
	})
	return response.Success(c, fiber.Map{"message": "logged out successfully"})
}

func (h *Handler) Me(c *fiber.Ctx) error {
	claims := GetUserClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthorized")
	}

	user, err := h.service.GetUser(claims.UserID)
	if err != nil {
		return response.NotFound(c, "user not found")
	}

	return response.Success(c, user)
}

func setAuthCookie(c *fiber.Ctx, token string, expiresAt time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  expiresAt,
		HTTPOnly: true,
		SameSite: "Lax",
		Path:     "/",
	})
}
