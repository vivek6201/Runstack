package api

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"code-runner/execution-engine/internal/ports"
	"code-runner/shared/dto"
	"code-runner/shared/models"
	"code-runner/shared/response"
)

type Handler struct {
	queue ports.JobQueue
	store ports.StateStore
}

func NewHandler(queue ports.JobQueue, store ports.StateStore) *Handler {
	return &Handler{
		queue: queue,
		store: store,
	}
}

// Submit handles POST /v1/submit
func (h *Handler) Submit(c *fiber.Ctx) error {
	var req dto.SubmitRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid JSON payload")
	}

	// Basic validation
	if strings.TrimSpace(req.Code) == "" {
		return response.BadRequest(c, "code cannot be empty")
	}
	if strings.TrimSpace(req.Language) == "" {
		return response.BadRequest(c, "language is required")
	}

	// Set reasonable default limits if not specified
	if req.TimeLimitMs <= 0 {
		req.TimeLimitMs = 3000 // 3 seconds default
	} else if req.TimeLimitMs > 15000 {
		req.TimeLimitMs = 15000 // 15 seconds max
	}

	if req.MemoryLimitMB <= 0 {
		req.MemoryLimitMB = 128
	}

	// Retrieve auth metadata from context
	var userID, apiKeyID string
	if authMeta, ok := c.Locals("auth").(*models.APIKeyCachedMetadata); ok && authMeta != nil {
		if authMeta.UserID != uuid.Nil {
			userID = authMeta.UserID.String()
		}
		if authMeta.APIKeyID != uuid.Nil {
			apiKeyID = authMeta.APIKeyID.String()
		}
	}

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

	// Push job into the queue
	if err := h.queue.Enqueue(c.Context(), job); err != nil {
		return response.InternalServerError(c, "failed to enqueue job")
	}

	return response.Accepted(c, dto.SubmitResponse{
		JobID:  jobID,
		Status: dto.StatusQueued,
	})
}

// Status handles GET /v1/status/:id
func (h *Handler) Status(c *fiber.Ctx) error {
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

	// Dynamic caching header:
	// Once execution is completed or failed, the result is immutable.
	if job.Status == dto.StatusCompleted || job.Status == dto.StatusFailed {
		c.Set("Cache-Control", "public, max-age=60, immutable")
	} else {
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	}

	return response.Success(c, job)
}

// Health handles GET /health
func (h *Handler) Health(c *fiber.Ctx) error {
	return response.Success(c, fiber.Map{
		"status":    "healthy",
		"service":   "execution-engine",
		"timestamp": time.Now().UTC(),
	})
}
