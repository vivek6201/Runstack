package keys

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"code-runner/dashboard/internal/auth"
	"code-runner/shared/dto"
	"code-runner/shared/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(c *fiber.Ctx) error {
	claims := auth.GetUserClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthorized")
	}

	var req dto.CreateAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		req = dto.CreateAPIKeyRequest{}
	}

	resp, err := h.service.CreateKey(c.Context(), claims.UserID, claims.Plan, req)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.Created(c, resp)
}

func (h *Handler) List(c *fiber.Ctx) error {
	claims := auth.GetUserClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthorized")
	}

	keys, err := h.service.ListKeys(c.Context(), claims.UserID)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.Success(c, keys)
}

func (h *Handler) Revoke(c *fiber.Ctx) error {
	claims := auth.GetUserClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthorized")
	}

	idParam := c.Params("id")
	keyID, err := uuid.Parse(idParam)
	if err != nil {
		return response.BadRequest(c, "invalid api key uuid")
	}

	if err := h.service.RevokeKey(c.Context(), claims.UserID, keyID); err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			return response.NotFound(c, err.Error())
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.Success(c, fiber.Map{"message": "api key revoked successfully"})
}
