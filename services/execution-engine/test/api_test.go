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

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"code-runner/execution-engine/internal/api"
	"code-runner/execution-engine/internal/config"
	"code-runner/execution-engine/internal/middleware"
	"code-runner/shared/crypto"
	"code-runner/shared/dto"
	"code-runner/shared/models"
	"code-runner/shared/response"
	sharedValidator "code-runner/shared/validator"
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

func (m *mockQueueStore) Ack(ctx context.Context, jobID string) error {
	return nil
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

	app := fiber.New(fiber.Config{
		StructValidator: sharedValidator.NewStructValidator(),
	})
	app.Get("/health", h.HealthHandler)

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

	app := fiber.New(fiber.Config{
		StructValidator: sharedValidator.NewStructValidator(),
	})
	app.Post("/v1/submit", h.SubmitHandler)
	app.Get("/v1/status/:id", h.StatusHandler)

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

	app := fiber.New(fiber.Config{
		StructValidator: sharedValidator.NewStructValidator(),
	})
	// Middleware injecting User B (attacker) auth metadata
	attackerUUID := uuid.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("auth", &models.APIKeyCachedMetadata{
			UserID:   attackerUUID,
			Plan:     models.PlanPro,
			IsActive: true,
		})
		return c.Next()
	})
	app.Get("/v1/status/:id", h.StatusHandler)

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

type mockKeyValidator struct {
	keys map[string]*models.APIKeyCachedMetadata
}

func (m *mockKeyValidator) ValidateKey(ctx context.Context, keyHash string) (*models.APIKeyCachedMetadata, error) {
	return m.keys[keyHash], nil
}

func TestAPI_MiddlewareAndRoutes_FullFlow(t *testing.T) {
	mockQS := newMockQueueStore()
	userID := uuid.New()
	keyID := uuid.New()
	rawAPIKey := "cr_live_integration_test_key"

	validator := &mockKeyValidator{
		keys: map[string]*models.APIKeyCachedMetadata{
			crypto.HashAPIKey(rawAPIKey): {
				APIKeyID:       keyID,
				UserID:         userID,
				Plan:           models.PlanPro,
				RateLimitRPM:   60,
				MaxConcurrency: 5,
				IsActive:       true,
			},
		},
	}

	// 1. Development app tests health, key auth, validation, submit, and status
	devCfg := &config.ConfigVar{AppEnv: "development", DevBypassKey: "cr_dev_bypass"}
	app := fiber.New(fiber.Config{
		CaseSensitive:   true,
		StructValidator: sharedValidator.NewStructValidator(),
		ErrorHandler:    response.FiberErrorHandler,
	})
	api.RegisterRoute(app, api.NewHandler(mockQS, mockQS), middleware.APIKeyAuth(validator, devCfg, nil))

	// Public health
	healthResp, _ := app.Test(httptest.NewRequest("GET", "/health", nil))
	if healthResp.StatusCode != 200 {
		t.Fatalf("GET /health = %d, want 200", healthResp.StatusCode)
	}

	// Missing API key -> 401
	unauthReq := httptest.NewRequest("POST", "/v1/submit", strings.NewReader(`{"language":"python","code":"print(1)"}`))
	unauthReq.Header.Set("Content-Type", "application/json")
	if resp, _ := app.Test(unauthReq); resp.StatusCode != 401 {
		t.Fatalf("missing key = %d, want 401", resp.StatusCode)
	}

	// Missing required code -> 400 via StructValidator
	invalidReq := httptest.NewRequest("POST", "/v1/submit", strings.NewReader(`{"language":"python"}`))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidReq.Header.Set("X-API-Key", rawAPIKey)
	if resp, _ := app.Test(invalidReq); resp.StatusCode != 400 {
		t.Fatalf("invalid payload = %d, want 400", resp.StatusCode)
	}

	// Valid API key -> 202 Accepted, then owner polls 200 OK
	validReq := httptest.NewRequest("POST", "/v1/submit", strings.NewReader(`{"language":"python","code":"print(99)"}`))
	validReq.Header.Set("Content-Type", "application/json")
	validReq.Header.Set("X-API-Key", rawAPIKey)
	subResp, _ := app.Test(validReq)
	if subResp.StatusCode != 202 {
		t.Fatalf("valid submit = %d, want 202", subResp.StatusCode)
	}
	var env response.Response[dto.SubmitResponse]
	_ = json.NewDecoder(subResp.Body).Decode(&env)

	statusReq := httptest.NewRequest("GET", "/v1/status/"+env.Data.JobID, nil)
	statusReq.Header.Set("Authorization", "Bearer "+rawAPIKey)
	statusResp, _ := app.Test(statusReq)
	if statusResp.StatusCode != 200 {
		t.Fatalf("owner GET status = %d, want 200", statusResp.StatusCode)
	}

	// 2. Production mode rejects dev bypass key with 401
	prodCfg := &config.ConfigVar{AppEnv: "production", DevBypassKey: "cr_dev_bypass"}
	prodApp := fiber.New(fiber.Config{StructValidator: sharedValidator.NewStructValidator()})
	api.RegisterRoute(prodApp, api.NewHandler(mockQS, mockQS), middleware.APIKeyAuth(validator, prodCfg, nil))

	bypassReq := httptest.NewRequest("POST", "/v1/submit", strings.NewReader(`{"language":"python","code":"print(1)"}`))
	bypassReq.Header.Set("Content-Type", "application/json")
	bypassReq.Header.Set("X-API-Key", "cr_dev_bypass")
	if resp, _ := prodApp.Test(bypassReq); resp.StatusCode != 401 {
		t.Fatalf("expected dev bypass rejected in production (401), got %d", resp.StatusCode)
	}
}
