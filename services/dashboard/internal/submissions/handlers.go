package submissions

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"code-runner/dashboard/internal/middleware"
	"code-runner/shared/response"
)

type ISubmissionsHandler interface {
	ListSubmissionsHandler(ctx fiber.Ctx) error
	GetSubmissionHandler(ctx fiber.Ctx) error
}

type SubmissionsHandler struct {
	service ISubmissionsService
}

func NewSubmissionsHandler(service ISubmissionsService) ISubmissionsHandler {
	return &SubmissionsHandler{service: service}
}

func (h *SubmissionsHandler) ListSubmissionsHandler(ctx fiber.Ctx) error {
	claims := middleware.GetUserClaims(ctx)
	if claims == nil {
		return response.Unauthorized(ctx, "unauthorized")
	}

	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "20"))
	status := ctx.Query("status", "")
	language := ctx.Query("language", "")

	query := SubmissionListQuery{
		Page:     page,
		Limit:    limit,
		Status:   status,
		Language: language,
	}

	items, total, err := h.service.List(ctx.Context(), claims.UserID, query)
	if err != nil {
		return response.InternalServerError(ctx, err.Error())
	}

	return response.Success(ctx, fiber.Map{
		"items": items,
		"total": total,
		"page":  query.Page,
		"limit": query.Limit,
	})
}

func (h *SubmissionsHandler) GetSubmissionHandler(ctx fiber.Ctx) error {
	claims := middleware.GetUserClaims(ctx)
	if claims == nil {
		return response.Unauthorized(ctx, "unauthorized")
	}

	subID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "invalid submission uuid")
	}

	detail, err := h.service.Get(ctx.Context(), claims.UserID, subID)
	if err != nil {
		if errors.Is(err, ErrSubmissionNotFound) {
			return response.NotFound(ctx, err.Error())
		}
		return response.InternalServerError(ctx, err.Error())
	}

	return response.Success(ctx, detail)
}
