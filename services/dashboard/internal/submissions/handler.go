package submissions

import (
	"errors"
	"strconv"

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

func (h *Handler) List(c *fiber.Ctx) error {
	claims := auth.GetUserClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthorized")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	status := c.Query("status", "")
	language := c.Query("language", "")

	query := dto.SubmissionListQuery{
		Page:     page,
		Limit:    limit,
		Status:   status,
		Language: language,
	}

	items, total, err := h.service.List(c.Context(), claims.UserID, query)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.Success(c, fiber.Map{
		"items": items,
		"total": total,
		"page":  query.Page,
		"limit": query.Limit,
	})
}

func (h *Handler) Get(c *fiber.Ctx) error {
	claims := auth.GetUserClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthorized")
	}

	idParam := c.Params("id")
	subID, err := uuid.Parse(idParam)
	if err != nil {
		return response.BadRequest(c, "invalid submission uuid")
	}

	detail, err := h.service.Get(c.Context(), claims.UserID, subID)
	if err != nil {
		if errors.Is(err, ErrSubmissionNotFound) {
			return response.NotFound(c, err.Error())
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.Success(c, detail)
}
