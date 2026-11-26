package stats

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"code-runner/dashboard/internal/middleware"
	"code-runner/shared/response"
)

type IStatsHandler interface {
	GetStatsHandler(ctx fiber.Ctx) error
	GetDailyStatsHandler(ctx fiber.Ctx) error
}

type StatsHandler struct {
	service IStatsService
}

func NewStatsHandler(service IStatsService) IStatsHandler {
	return &StatsHandler{service: service}
}

func (h *StatsHandler) GetStatsHandler(ctx fiber.Ctx) error {
	claims := middleware.GetUserClaims(ctx)
	if claims == nil {
		return response.Unauthorized(ctx, "unauthorized")
	}

	overview, err := h.service.GetDashboardStats(ctx.Context(), claims.UserID, claims.Plan)
	if err != nil {
		return response.InternalServerError(ctx, err.Error())
	}

	return response.Success(ctx, overview)
}

func (h *StatsHandler) GetDailyStatsHandler(ctx fiber.Ctx) error {
	claims := middleware.GetUserClaims(ctx)
	if claims == nil {
		return response.Unauthorized(ctx, "unauthorized")
	}

	days, _ := strconv.Atoi(ctx.Query("days", "30"))
	series, err := h.service.GetDailyStats(ctx.Context(), claims.UserID, days)
	if err != nil {
		return response.InternalServerError(ctx, err.Error())
	}

	return response.Success(ctx, series)
}
