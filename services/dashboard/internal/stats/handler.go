package stats

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"code-runner/dashboard/internal/auth"
	"code-runner/shared/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetStats(c *fiber.Ctx) error {
	claims := auth.GetUserClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthorized")
	}

	stats, err := h.service.GetDashboardStats(c.Context(), claims.UserID, claims.Plan)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.Success(c, stats)
}

func (h *Handler) GetDaily(c *fiber.Ctx) error {
	claims := auth.GetUserClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthorized")
	}

	days, _ := strconv.Atoi(c.Query("days", "30"))
	series, err := h.service.GetDailyStats(c.Context(), claims.UserID, days)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.Success(c, series)
}
