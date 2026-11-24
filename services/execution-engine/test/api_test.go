package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v2"

	"code-runner/execution-engine/internal/api"
	"code-runner/shared/dto"
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
