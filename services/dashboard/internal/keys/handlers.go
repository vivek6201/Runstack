package keys

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"code-runner/dashboard/internal/middleware"
	"code-runner/shared/response"
)

type IKeysHandler interface {
	CreateKeyHandler(ctx fiber.Ctx) error
	ListKeysHandler(ctx fiber.Ctx) error
	RevokeKeyHandler(ctx fiber.Ctx) error
}

type KeysHandler struct {
	service IKeysService
}

func NewKeysHandler(service IKeysService) IKeysHandler {
	return &KeysHandler{service: service}
}

func (h *KeysHandler) CreateKeyHandler(ctx fiber.Ctx) error {
	claims := middleware.GetUserClaims(ctx)
	if claims == nil {
		return response.Unauthorized(ctx, "unauthorized")
	}

	var req CreateAPIKeyRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		req = CreateAPIKeyRequest{}
	}

	resp, err := h.service.CreateKey(ctx.Context(), claims.UserID, claims.Plan, req)
	if err != nil {
		return response.InternalServerError(ctx, err.Error())
	}

	return response.Created(ctx, resp)
}

func (h *KeysHandler) ListKeysHandler(ctx fiber.Ctx) error {
	claims := middleware.GetUserClaims(ctx)
	if claims == nil {
		return response.Unauthorized(ctx, "unauthorized")
	}

	list, err := h.service.ListKeys(ctx.Context(), claims.UserID)
	if err != nil {
		return response.InternalServerError(ctx, err.Error())
	}

	return response.Success(ctx, list)
}

func (h *KeysHandler) RevokeKeyHandler(ctx fiber.Ctx) error {
	claims := middleware.GetUserClaims(ctx)
	if claims == nil {
		return response.Unauthorized(ctx, "unauthorized")
	}

	keyID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "invalid api key uuid")
	}

	if err := h.service.RevokeKey(ctx.Context(), claims.UserID, keyID); err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			return response.NotFound(ctx, err.Error())
		}
		return response.InternalServerError(ctx, err.Error())
	}

	return response.Success(ctx, fiber.Map{"message": "api key revoked successfully"})
}
