package test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"code-runner/dashboard/internal/auth"
	"code-runner/dashboard/internal/config"
	"code-runner/dashboard/internal/playground"
	"code-runner/shared/dto"
	"code-runner/shared/models"
	"code-runner/shared/response"
	sharedValidator "code-runner/shared/validator"
)

type mockUserRepo struct {
	users map[string]*models.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*models.User)}
}

func (m *mockUserRepo) Create(ctx context.Context, user *models.User) error {
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, auth.ErrUserNotFound
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, auth.ErrUserNotFound
}

func (m *mockUserRepo) Update(ctx context.Context, user *models.User) error {
	m.users[user.Email] = user
	return nil
}

func TestAuthService_PasswordAndEmailPolicy(t *testing.T) {
	repo := newMockUserRepo()
	cfg := &config.ConfigVar{
		JWTSecret: "test-jwt-secret",
		TokenTTL:  time.Hour,
	}
	svc := auth.NewAuthService(repo, cfg)

	invalidCases := []struct {
		name     string
		email    string
		password string
	}{
		{"too short", "a@example.com", "abc1234"},
		{"no digit", "b@example.com", "OnlyLettersHere"},
		{"no letter", "c@example.com", "123456789"},
		{"blank whitespace", "d@example.com", "        "},
		{"exceeds bcrypt 72 bytes", "e@example.com", strings.Repeat("a1", 37)},
		{"invalid email", "not-an-email", "SecurePass123"},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Register(context.Background(), auth.RegisterRequest{
				Email:    tc.email,
				Password: tc.password,
			})
			if err == nil {
				t.Errorf("expected Register to reject %s, but succeeded", tc.name)
			}
		})
	}

	resp, err := svc.Register(context.Background(), auth.RegisterRequest{
		Email:    "valid@example.com",
		Password: "StrongPassword123",
	})
	if err != nil {
		t.Fatalf("expected valid registration to succeed, got error: %v", err)
	}
	if resp.User.Email != "valid@example.com" || resp.Token == "" {
		t.Errorf("unexpected registration response: %+v", resp)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestPlaygroundHandler_ForwardsUserIDOnSubmitAndStatus(t *testing.T) {
	expectedUserID := uuid.New()
	expectedBypassKey := "cr_dev_test_secret"
	jobID := uuid.New().String()

	var mu sync.Mutex
	var submitUserID, statusUserID string
	var submitAPIKey, statusAPIKey string

	engineHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/submit":
			submitUserID = r.Header.Get("X-User-ID")
			submitAPIKey = r.Header.Get("X-API-Key")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(response.Response[dto.SubmitResponse]{
				Success: true,
				Data: dto.SubmitResponse{
					JobID:  jobID,
					Status: dto.StatusQueued,
				},
			})
		case "/v1/status/" + jobID:
			statusUserID = r.Header.Get("X-User-ID")
			statusAPIKey = r.Header.Get("X-API-Key")
			if statusUserID != submitUserID || statusUserID == "" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			now := time.Now()
			_ = json.NewEncoder(w).Encode(response.Response[dto.Job]{
				Success: true,
				Data: dto.Job{
					ID:          jobID,
					UserID:      expectedUserID.String(),
					Language:    "python",
					Code:        "print(42)",
					Status:      dto.StatusCompleted,
					CreatedAt:   now,
					CompletedAt: &now,
					Result: &dto.ExecutionResult{
						Stdout:   "42\n",
						ExitCode: 0,
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	svc := playground.NewPlaygroundService(nil, nil, &config.ConfigVar{
		EngineURL:    "http://execution-engine.local",
		DevBypassKey: expectedBypassKey,
	})
	svc.SetHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			engineHandler.ServeHTTP(rec, req)
			return rec.Result(), nil
		}),
	})
	h := playground.NewPlaygroundHandler(svc)

	app := fiber.New(fiber.Config{
		StructValidator: sharedValidator.NewStructValidator(),
	})
	app.Post("/api/playground/execute", func(c fiber.Ctx) error {
		c.Locals("claims", &auth.Claims{
			UserID: expectedUserID,
			Email:  "user@example.com",
			Role:   models.RoleMember,
			Plan:   models.PlanFree,
		})
		return h.ExecuteHandler(c)
	})

	payload, _ := json.Marshal(playground.ExecuteRequest{
		Language: "python",
		Code:     "print(42)",
	})
	req := httptest.NewRequest("POST", "/api/playground/execute", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 OK from playground execute, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	mu.Lock()
	defer mu.Unlock()
	if submitUserID != expectedUserID.String() {
		t.Errorf("submit X-User-ID = %q, want %q", submitUserID, expectedUserID.String())
	}
	if statusUserID != expectedUserID.String() {
		t.Errorf("status polling X-User-ID = %q, want %q", statusUserID, expectedUserID.String())
	}
	if submitAPIKey != expectedBypassKey || statusAPIKey != expectedBypassKey {
		t.Errorf("API keys mismatch: submit=%q, status=%q", submitAPIKey, statusAPIKey)
	}
}
