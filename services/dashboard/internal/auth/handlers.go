package auth

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"code-runner/dashboard/internal/config"
	"code-runner/shared/response"
)

type IAuthHandler interface {
	RegisterHandler(ctx fiber.Ctx) error
	LoginHandler(ctx fiber.Ctx) error
	LogoutHandler(ctx fiber.Ctx) error
	MeHandler(ctx fiber.Ctx) error
}

type AuthHandler struct {
	service IAuthService
	cfg     *config.ConfigVar
}

func NewAuthHandler(service IAuthService, cfg *config.ConfigVar) IAuthHandler {
	return &AuthHandler{
		service: service,
		cfg:     cfg,
	}
}

func (h *AuthHandler) RegisterHandler(ctx fiber.Ctx) error {
	var req RegisterRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return response.BadRequest(ctx, "invalid registration payload: "+err.Error())
	}

	resp, err := h.service.Register(ctx.Context(), req)
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			return response.Error(ctx, fiber.StatusConflict, err.Error())
		}
		return response.BadRequest(ctx, err.Error())
	}

	h.setAuthCookie(ctx, resp.Token, resp.ExpiresAt)
	return response.Created(ctx, resp)
}

func (h *AuthHandler) LoginHandler(ctx fiber.Ctx) error {
	var req LoginRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return response.BadRequest(ctx, "invalid login payload")
	}

	resp, err := h.service.Login(ctx.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return response.Unauthorized(ctx, err.Error())
		}
		return response.InternalServerError(ctx, err.Error())
	}

	h.setAuthCookie(ctx, resp.Token, resp.ExpiresAt)
	return response.Success(ctx, resp)
}

func (h *AuthHandler) LogoutHandler(ctx fiber.Ctx) error {
	ctx.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Secure:   h.cfg != nil && h.cfg.CookieSecure,
		Domain:   h.cookieDomain(),
		SameSite: "Lax",
		Path:     "/",
	})
	return response.Success(ctx, fiber.Map{"message": "logged out successfully"})
}

func (h *AuthHandler) MeHandler(ctx fiber.Ctx) error {
	claims, ok := ctx.Locals("claims").(*Claims)
	if !ok || claims == nil || claims.UserID == uuid.Nil {
		return response.Unauthorized(ctx, "unauthorized")
	}

	user, err := h.service.GetUser(ctx.Context(), claims.UserID)
	if err != nil {
		return response.NotFound(ctx, "user not found")
	}

	return response.Success(ctx, user)
}

func (h *AuthHandler) setAuthCookie(ctx fiber.Ctx, token string, expiresAt time.Time) {
	ctx.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  expiresAt,
		HTTPOnly: true,
		Secure:   h.cfg != nil && h.cfg.CookieSecure,
		Domain:   h.cookieDomain(),
		SameSite: "Lax",
		Path:     "/",
	})
}

func (h *AuthHandler) cookieDomain() string {
	if h.cfg == nil {
		return ""
	}
	return h.cfg.CookieDomain
}
