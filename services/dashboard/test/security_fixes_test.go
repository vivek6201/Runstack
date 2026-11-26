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
	"code-runner/dashboard/internal/profile"
	"code-runner/shared/dto"
	"code-runner/shared/models"
	"code-runner/shared/response"
	sharedValidator "code-runner/shared/validator"
)

type inMemoryAuthRepo struct {
	mu       sync.Mutex
	users    map[string]*models.User
	profiles map[uuid.UUID]*models.Profile
	sessions map[string]*models.Session
}

func newInMemoryAuthRepo() *inMemoryAuthRepo {
	return &inMemoryAuthRepo{
		users:    make(map[string]*models.User),
		profiles: make(map[uuid.UUID]*models.Profile),
		sessions: make(map[string]*models.Session),
	}
}

func (m *inMemoryAuthRepo) CreateWithProfile(ctx context.Context, user *models.User, prof *models.Profile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	uCopy := *user
	pCopy := *prof
	m.users[user.Email] = &uCopy
	m.profiles[user.ID] = &pCopy
	return nil
}

func (m *inMemoryAuthRepo) FindByIDWithProfile(ctx context.Context, id uuid.UUID) (*models.User, *models.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.ID == id {
			uCopy := *u
			if p, ok := m.profiles[id]; ok {
				pCopy := *p
				return &uCopy, &pCopy, nil
			}
			return &uCopy, nil, nil
		}
	}
	return nil, nil, auth.ErrUserNotFound
}

func (m *inMemoryAuthRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[email]; ok {
		uCopy := *u
		return &uCopy, nil
	}
	return nil, auth.ErrUserNotFound
}

func (m *inMemoryAuthRepo) CreateSingleSession(ctx context.Context, sess *models.Session) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var evicted []string
	for id, existing := range m.sessions {
		if existing.UserID == sess.UserID {
			evicted = append(evicted, id)
			delete(m.sessions, id)
		}
	}
	sCopy := *sess
	m.sessions[sess.ID] = &sCopy
	return evicted, nil
}

func (m *inMemoryAuthRepo) FindSessionByID(ctx context.Context, sessionHash string) (*models.Session, *models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sess, ok := m.sessions[sessionHash]
	if !ok {
		return nil, nil, auth.ErrSessionInvalid
	}
	for _, u := range m.users {
		if u.ID == sess.UserID {
			sCopy := *sess
			uCopy := *u
			return &sCopy, &uCopy, nil
		}
	}
	return nil, nil, auth.ErrSessionInvalid
}

func (m *inMemoryAuthRepo) DeleteSession(ctx context.Context, sessionHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionHash)
	return nil
}

func (m *inMemoryAuthRepo) TouchSession(ctx context.Context, sessionHash string, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sess, ok := m.sessions[sessionHash]; ok {
		sess.LastActiveAt = now
	}
	return nil
}

// Adapter exposing inMemoryAuthRepo profiles map as profile.IProfileRepository
func (m *inMemoryAuthRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*models.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.profiles[userID]
	if !ok {
		return nil, profile.ErrProfileNotFound
	}
	pCopy := *p
	return &pCopy, nil
}

func (m *inMemoryAuthRepo) FindByUsername(ctx context.Context, username string) (*models.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.profiles {
		if p.Username == username {
			pCopy := *p
			return &pCopy, nil
		}
	}
	return nil, profile.ErrProfileNotFound
}

func (m *inMemoryAuthRepo) Update(ctx context.Context, prof *models.Profile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	pCopy := *prof
	pCopy.UpdatedAt = time.Now().UTC()
	m.profiles[prof.UserID] = &pCopy
	return nil
}

func TestAuthService_PasswordAndEmailPolicy(t *testing.T) {
	repo := newInMemoryAuthRepo()
	cfg := &config.ConfigVar{
		TokenTTL: time.Hour,
	}
	svc := auth.NewAuthService(repo, nil, cfg)

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
			}, auth.SessionMetadata{})
			if err == nil {
				t.Errorf("expected Register to reject %s, but succeeded", tc.name)
			}
		})
	}

	resp, err := svc.Register(context.Background(), auth.RegisterRequest{
		Email:    "valid@example.com",
		Password: "StrongPassword123",
	}, auth.SessionMetadata{
		IPAddress: "203.0.113.10",
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X) Chrome/120.0",
	})
	if err != nil {
		t.Fatalf("expected valid registration to succeed, got error: %v", err)
	}
	if resp.User.Email != "valid@example.com" || resp.Token == "" || resp.User.Profile == nil {
		t.Fatalf("unexpected registration response: %+v", resp)
	}
	if resp.Session == nil || resp.Session.DeviceName != "macOS • Chrome" {
		t.Errorf("expected parsed session device metadata, got %+v", resp.Session)
	}
}

func TestAuthService_SingleActiveSession_LatestLoginWins(t *testing.T) {
	repo := newInMemoryAuthRepo()
	svc := auth.NewAuthService(repo, nil, &config.ConfigVar{TokenTTL: time.Hour})

	regResp, err := svc.Register(context.Background(), auth.RegisterRequest{
		Email:    "solo@runstack.io",
		Password: "Password123",
	}, auth.SessionMetadata{IPAddress: "10.0.0.1", UserAgent: "Mozilla/5.0 (Macintosh) Chrome/120"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	tokenDeviceA := regResp.Token

	// Device A session works before Device B logs in
	scA, err := svc.ValidateSession(context.Background(), tokenDeviceA)
	if err != nil || scA.IPAddress != "10.0.0.1" {
		t.Fatalf("expected Device A session valid, err=%v", err)
	}

	// Device B logs in -> must evict Device A
	loginResp, err := svc.Login(context.Background(), auth.LoginRequest{
		Email:    "solo@runstack.io",
		Password: "Password123",
	}, auth.SessionMetadata{IPAddress: "10.0.0.2", UserAgent: "Mozilla/5.0 (Windows NT) Firefox/121"})
	if err != nil {
		t.Fatalf("device B login failed: %v", err)
	}
	tokenDeviceB := loginResp.Token

	// Device A must immediately fail validation
	if _, err := svc.ValidateSession(context.Background(), tokenDeviceA); err == nil {
		t.Fatalf("SECURITY FAILURE: Device A session remained active after Device B login")
	}

	// Device B must succeed with updated metadata
	scB, err := svc.ValidateSession(context.Background(), tokenDeviceB)
	if err != nil || scB.IPAddress != "10.0.0.2" || scB.DeviceName != "Windows • Firefox" {
		t.Fatalf("expected Device B session with updated metadata, got sc=%+v err=%v", scB, err)
	}

	// Explicit logout invalidates Device B
	if err := svc.Logout(context.Background(), tokenDeviceB); err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	if _, err := svc.ValidateSession(context.Background(), tokenDeviceB); err == nil {
		t.Fatalf("expected Device B session invalidated after logout")
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
