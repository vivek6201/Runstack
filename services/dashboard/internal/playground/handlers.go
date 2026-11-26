package playground

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"code-runner/dashboard/internal/middleware"
	"code-runner/shared/response"
)

type IPlaygroundHandler interface {
	ExecuteHandler(ctx fiber.Ctx) error
}

type PlaygroundHandler struct {
	service IPlaygroundService
}

func NewPlaygroundHandler(service IPlaygroundService) IPlaygroundHandler {
	return &PlaygroundHandler{service: service}
}

func (h *PlaygroundHandler) ExecuteHandler(ctx fiber.Ctx) error {
	claims := middleware.GetUserClaims(ctx)
	if claims == nil {
		return response.Unauthorized(ctx, "unauthorized")
	}

	var req ExecuteRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return response.BadRequest(ctx, "invalid execution request: language and code are required")
	}

	job, err := h.service.Execute(ctx.Context(), claims.UserID, &req)
	if err != nil {
		var httpErr *EngineHTTPError
		switch {
		case errors.As(err, &httpErr):
			return response.Error(ctx, httpErr.StatusCode, httpErr.Body)
		case errors.Is(err, ErrEngineUnreachable):
			return response.BadGateway(ctx, err.Error())
		case errors.Is(err, ErrExecutionTimeout):
			return response.GatewayTimeout(ctx, err.Error())
		default:
			return response.InternalServerError(ctx, err.Error())
		}
	}

	return response.Success(ctx, job)
}
