package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"code-runner/execution-engine/internal/api"
	"code-runner/shared/dto"
	"code-runner/shared/models"
	"code-runner/shared/response"
)

// in-memory mock queue and state store for isolated API testing
type mockQueueStore struct {
	mu   sync.Mutex
	jobs map[string]*dto.Job
}

func newMockQueueStore() *mockQueueStore {
	return &mockQueueStore{jobs: make(map[string]*dto.Job)}
}

func (m *mockQueueStore) Enqueue(ctx context.Context, job *dto.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[job.ID] = job
	return nil
}

func (m *mockQueueStore) Dequeue(ctx context.Context) (*dto.Job, error) {
	return nil, nil
}

func (m *mockQueueStore) SetStatus(ctx context.Context, jobID string, status dto.JobStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j, ok := m.jobs[jobID]; ok {
		j.Status = status
	}
	return nil
}

func (m *mockQueueStore) SetResult(ctx context.Context, jobID string, result *dto.ExecutionResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j, ok := m.jobs[jobID]; ok {
		j.Status = dto.StatusCompleted
		j.Result = result
	}
	return nil
}

func (m *mockQueueStore) SetError(ctx context.Context, jobID string, errMessage string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j, ok := m.jobs[jobID]; ok {
		j.Status = dto.StatusFailed
		j.Error = errMessage
	}
	return nil
}

func (m *mockQueueStore) GetJob(ctx context.Context, jobID string) (*dto.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.jobs[jobID], nil
}

func TestAPI_Health(t *testing.T) {
	mock := newMockQueueStore()
	h := api.NewHandler(mock, mock)

	app := fiber.New()
	app.Get("/health", h.Health)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestAPI_Submit_And_Status(t *testing.T) {
	mock := newMockQueueStore()
	h := api.NewHandler(mock, mock)

	app := fiber.New()
	app.Post("/v1/submit", h.Submit)
	app.Get("/v1/status/:id", h.Status)

	submitBody := dto.SubmitRequest{
		Language:    "python",
		Code:        "print('hello from api')",
		TimeLimitMs: 2000,
	}
	bodyBytes, _ := json.Marshal(submitBody)

	req := httptest.NewRequest("POST", "/v1/submit", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusAccepted {
		t.Fatalf("expected status 202 Accepted, got %d", resp.StatusCode)
	}

	var env response.Response[dto.SubmitResponse]
	json.NewDecoder(resp.Body).Decode(&env)

	if !env.Success || env.Data.JobID == "" || env.Data.Status != dto.StatusQueued {
		t.Fatalf("unexpected submit response: %+v", env)
	}

	// Poll status
	statusReq := httptest.NewRequest("GET", "/v1/status/"+env.Data.JobID, nil)
	statusResp, err := app.Test(statusReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if statusResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", statusResp.StatusCode)
	}
}

func TestAPI_Status_IDOR_Protection(t *testing.T) {
	mock := newMockQueueStore()
	h := api.NewHandler(mock, mock)

	// User A owns this job
	ownerUUID := uuid.New()
	jobA := &dto.Job{
		ID:        "job-user-a",
		UserID:    ownerUUID.String(),
		Language:  "python",
		Code:      "secret_api_token = 'SUPER_SECRET_123'",
		Status:    dto.StatusCompleted,
		CreatedAt: time.Now(),
	}
	_ = mock.Enqueue(context.Background(), jobA)

	app := fiber.New()
	// Middleware injecting User B (attacker) auth metadata
	attackerUUID := uuid.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("auth", &models.APIKeyCachedMetadata{
			UserID:   attackerUUID,
			Plan:     models.PlanPro,
			IsActive: true,
		})
		return c.Next()
	})
	app.Get("/v1/status/:id", h.Status)

	// Attacker tries to query User A's job
	req := httptest.NewRequest("GET", "/v1/status/job-user-a", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must be blocked with 403 Forbidden!
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("SECURITY VULNERABILITY: expected status 403 Forbidden on cross-tenant job access, got %d", resp.StatusCode)
	}

	// Verify Cache-Control header is private
	cacheControl := resp.Header.Get("Cache-Control")
	if strings.Contains(cacheControl, "public") {
		t.Errorf("SECURITY VULNERABILITY: Cache-Control should be private, got %s", cacheControl)
	}
}
