package api

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"code-runner/execution-engine/internal/ports"
	"code-runner/shared/dto"
	"code-runner/shared/models"
	"code-runner/shared/response"
)

type IHandler interface {
	SubmitHandler(c fiber.Ctx) error
	StatusHandler(c fiber.Ctx) error
	HealthHandler(c fiber.Ctx) error
}

type Handler struct {
	queue ports.JobQueue
	store ports.StateStore
}

func NewHandler(queue ports.JobQueue, store ports.StateStore) IHandler {
	return &Handler{
		queue: queue,
		store: store,
	}
}

// SubmitHandler handles POST /v1/submit.
func (h *Handler) SubmitHandler(c fiber.Ctx) error {
	var req dto.SubmitRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.BadRequest(c, "invalid JSON payload: language and code are required")
	}

	if strings.TrimSpace(req.Code) == "" {
		return response.BadRequest(c, "code cannot be empty")
	}
	if strings.TrimSpace(req.Language) == "" {
		return response.BadRequest(c, "language is required")
	}

	applyDefaultBounds(&req)
	userID, apiKeyID := extractAuthIdentifiers(c)

	jobID := uuid.New().String()
	job := &dto.Job{
		ID:            jobID,
		UserID:        userID,
		APIKeyID:      apiKeyID,
		Language:      req.Language,
		Code:          req.Code,
		TestCases:     req.TestCases,
		TimeLimitMs:   req.TimeLimitMs,
		MemoryLimitMB: req.MemoryLimitMB,
		Status:        dto.StatusQueued,
		CreatedAt:     time.Now(),
	}

	if err := h.queue.Enqueue(c.Context(), job); err != nil {
		return response.InternalServerError(c, "failed to enqueue job")
	}

	return response.Accepted(c, dto.SubmitResponse{
		JobID:  jobID,
		Status: dto.StatusQueued,
	})
}

// StatusHandler handles GET /v1/status/:id.
func (h *Handler) StatusHandler(c fiber.Ctx) error {
	jobID := c.Params("id")
	if jobID == "" {
		return response.BadRequest(c, "missing job id")
	}

	job, err := h.store.GetJob(c.Context(), jobID)
	if err != nil {
		return response.InternalServerError(c, "failed to query job state")
	}
	if job == nil {
		return response.NotFound(c, "job not found or expired")
	}

	if !canAccessJob(c, job) {
		return response.Forbidden(c, "access denied: you do not have permission to view this job")
	}

	c.Set("Cache-Control", "private, no-cache, no-store, must-revalidate")
	return response.Success(c, job)
}

// HealthHandler handles GET /health.
func (h *Handler) HealthHandler(c fiber.Ctx) error {
	return response.Success(c, fiber.Map{
		"status":    "healthy",
		"service":   "execution-engine",
		"timestamp": time.Now().UTC(),
	})
}

func applyDefaultBounds(req *dto.SubmitRequest) {
	if req.TimeLimitMs <= 0 {
		req.TimeLimitMs = 3000
	} else if req.TimeLimitMs > 15000 {
		req.TimeLimitMs = 15000
	}
	if req.MemoryLimitMB <= 0 {
		req.MemoryLimitMB = 128
	}
}

func extractAuthIdentifiers(c fiber.Ctx) (userID, apiKeyID string) {
	authMeta, ok := c.Locals("auth").(*models.APIKeyCachedMetadata)
	if !ok || authMeta == nil {
		return "", ""
	}
	if authMeta.UserID != uuid.Nil {
		userID = authMeta.UserID.String()
	}
	if authMeta.APIKeyID != uuid.Nil {
		apiKeyID = authMeta.APIKeyID.String()
	}
	return userID, apiKeyID
}

func canAccessJob(c fiber.Ctx, job *dto.Job) bool {
	if job.UserID == "" {
		return true
	}
	authMeta, ok := c.Locals("auth").(*models.APIKeyCachedMetadata)
	if !ok || authMeta == nil {
		return true
	}
	return authMeta.UserID != uuid.Nil && job.UserID == authMeta.UserID.String()
}
