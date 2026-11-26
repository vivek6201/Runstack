package profile

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"code-runner/dashboard/internal/middleware"
	"code-runner/shared/response"
)

type IProfileHandler interface {
	GetProfileHandler(ctx fiber.Ctx) error
	UpdateProfileHandler(ctx fiber.Ctx) error
}

type ProfileHandler struct {
	service IProfileService
}

func NewProfileHandler(service IProfileService) IProfileHandler {
	return &ProfileHandler{service: service}
}

func (h *ProfileHandler) GetProfileHandler(ctx fiber.Ctx) error {
	claims := middleware.GetUserClaims(ctx)
	if claims == nil {
		return response.Unauthorized(ctx, "unauthorized")
	}

	prof, err := h.service.GetByUserID(ctx.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			return response.NotFound(ctx, err.Error())
		}
		return response.InternalServerError(ctx, err.Error())
	}

	return response.Success(ctx, prof)
}

func (h *ProfileHandler) UpdateProfileHandler(ctx fiber.Ctx) error {
	claims := middleware.GetUserClaims(ctx)
	if claims == nil {
		return response.Unauthorized(ctx, "unauthorized")
	}

	var req UpdateProfileRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return response.BadRequest(ctx, "invalid profile payload: "+err.Error())
	}

	prof, err := h.service.UpdateByUserID(ctx.Context(), claims.UserID, req)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			return response.NotFound(ctx, err.Error())
		}
		if errors.Is(err, ErrUsernameTaken) {
			return response.Error(ctx, fiber.StatusConflict, err.Error())
		}
		return response.BadRequest(ctx, err.Error())
	}

	return response.Success(ctx, prof)
}
