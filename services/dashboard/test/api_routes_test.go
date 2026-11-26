package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"code-runner/dashboard/internal/auth"
	"code-runner/dashboard/internal/config"
	"code-runner/dashboard/internal/keys"
	"code-runner/dashboard/internal/middleware"
	"code-runner/dashboard/internal/playground"
	"code-runner/dashboard/internal/profile"
	"code-runner/dashboard/internal/stats"
	"code-runner/dashboard/internal/submissions"
	"code-runner/shared/dto"
	"code-runner/shared/models"
	"code-runner/shared/response"
	sharedValidator "code-runner/shared/validator"
)

type inMemoryKeysRepo struct {
	mu   sync.Mutex
	keys map[uuid.UUID]*models.APIKey
}

func newInMemoryKeysRepo() *inMemoryKeysRepo {
	return &inMemoryKeysRepo{keys: make(map[uuid.UUID]*models.APIKey)}
}

func (r *inMemoryKeysRepo) Create(ctx context.Context, key *models.APIKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := *key
	copied.CreatedAt = time.Now().UTC()
	r.keys[key.ID] = &copied
	return nil
}

func (r *inMemoryKeysRepo) FindByID(ctx context.Context, userID, keyID uuid.UUID) (*models.APIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k, ok := r.keys[keyID]
	if !ok || k.UserID != userID || !k.IsActive {
		return nil, keys.ErrKeyNotFound
	}
	copied := *k
	return &copied, nil
}

func (r *inMemoryKeysRepo) FindByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, k := range r.keys {
		if k.KeyHash == hash && k.IsActive {
			copied := *k
			return &copied, nil
		}
	}
	return nil, keys.ErrKeyNotFound
}

func (r *inMemoryKeysRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]models.APIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var list []models.APIKey
	for _, k := range r.keys {
		if k.UserID == userID {
			list = append(list, *k)
		}
	}
	return list, nil
}

func (r *inMemoryKeysRepo) Revoke(ctx context.Context, userID, keyID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k, ok := r.keys[keyID]
	if !ok || k.UserID != userID || !k.IsActive {
		return keys.ErrKeyNotFound
	}
	now := time.Now().UTC()
	k.IsActive = false
	k.RevokedAt = &now
	return nil
}

type inMemorySubmissionsRepo struct {
	mu   sync.Mutex
	subs map[uuid.UUID]*models.Submission
}

func newInMemorySubmissionsRepo() *inMemorySubmissionsRepo {
	return &inMemorySubmissionsRepo{subs: make(map[uuid.UUID]*models.Submission)}
}

func (r *inMemorySubmissionsRepo) Create(ctx context.Context, sub *models.Submission) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := *sub
	if copied.CreatedAt.IsZero() {
		copied.CreatedAt = time.Now().UTC()
	}
	r.subs[sub.ID] = &copied
	return nil
}

func (r *inMemorySubmissionsRepo) FindByID(ctx context.Context, userID, subID uuid.UUID) (*models.Submission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sub, ok := r.subs[subID]
	if !ok || sub.UserID != userID {
		return nil, submissions.ErrSubmissionNotFound
	}
	copied := *sub
	return &copied, nil
}

func (r *inMemorySubmissionsRepo) List(ctx context.Context, userID uuid.UUID, status, language string, limit, offset int) ([]models.Submission, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var matched []models.Submission
	for _, s := range r.subs {
		if s.UserID != userID {
			continue
		}
		if status != "" && string(s.Status) != status {
			continue
		}
		if language != "" && s.Language != language {
			continue
		}
		matched = append(matched, *s)
	}
	return matched, int64(len(matched)), nil
}

func (r *inMemorySubmissionsRepo) GetStats(ctx context.Context, userID uuid.UUID) (*submissions.StatsAggregate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var agg submissions.StatsAggregate
	for _, s := range r.subs {
		if s.UserID != userID {
			continue
		}
		agg.TotalSubmissions++
		switch s.Status {
		case models.SubmissionCompleted:
			agg.SuccessfulRuns++
		case models.SubmissionFailed:
			agg.FailedRuns++
		case models.SubmissionTimedOut:
			agg.TimeoutRuns++
		}
		agg.TotalTimeMs += s.TimeTakenMs
	}
	if agg.TotalSubmissions > 0 {
		agg.AvgTimeMs = float64(agg.TotalTimeMs) / float64(agg.TotalSubmissions)
	}
	return &agg, nil
}

func (r *inMemorySubmissionsRepo) GetMonthlyRuns(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var count int64
	for _, s := range r.subs {
		if s.UserID == userID && !s.CreatedAt.Before(since) {
			count++
		}
	}
	return count, nil
}

func (r *inMemorySubmissionsRepo) GetDailyRows(ctx context.Context, userID uuid.UUID, days int) ([]submissions.DailyStatRow, error) {
	return []submissions.DailyStatRow{}, nil
}

func TestDashboard_AllAPIRoutes_Integration(t *testing.T) {
	cfg := &config.ConfigVar{
		AppEnv:       "development",
		TokenTTL:     time.Hour,
		EngineURL:    "http://engine.local",
		DevBypassKey: "cr_dev_bypass",
	}

	sharedAuthRepo := newInMemoryAuthRepo()
	kRepo := newInMemoryKeysRepo()
	sRepo := newInMemorySubmissionsRepo()

	authSvc := auth.NewAuthService(sharedAuthRepo, nil, cfg)
	authHandler := auth.NewAuthHandler(authSvc, cfg)
	authMw := middleware.AuthRequired(authSvc)

	profileSvc := profile.NewProfileService(sharedAuthRepo, nil)
	profileHandler := profile.NewProfileHandler(profileSvc)

	keysSvc := keys.NewKeysService(kRepo, nil)
	keysHandler := keys.NewKeysHandler(keysSvc)

	subSvc := submissions.NewSubmissionsService(sRepo, nil)
	subHandler := submissions.NewSubmissionsHandler(subSvc)

	statsSvc := stats.NewStatsService(sRepo, nil, nil)
	statsHandler := stats.NewStatsHandler(statsSvc)

	jobID := uuid.New().String()
	playSvc := playground.NewPlaygroundService(sRepo, nil, cfg)
	playSvc.SetHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			rec.Header().Set("Content-Type", "application/json")
			switch req.URL.Path {
			case "/v1/submit":
				rec.WriteHeader(http.StatusAccepted)
				_ = json.NewEncoder(rec).Encode(response.Response[dto.SubmitResponse]{
					Success: true,
					Data:    dto.SubmitResponse{JobID: jobID, Status: dto.StatusQueued},
				})
			case "/v1/status/" + jobID:
				now := time.Now().UTC()
				_ = json.NewEncoder(rec).Encode(response.Response[dto.Job]{
					Success: true,
					Data: dto.Job{
						ID:          jobID,
						Language:    "python",
						Code:        "print('hello')",
						Status:      dto.StatusCompleted,
						CreatedAt:   now,
						CompletedAt: &now,
						Result: &dto.ExecutionResult{
							Stdout:      "hello\n",
							ExitCode:    0,
							TimeTakenMs: 25,
						},
					},
				})
			default:
				rec.WriteHeader(http.StatusNotFound)
			}
			return rec.Result(), nil
		}),
	})
	playHandler := playground.NewPlaygroundHandler(playSvc)

	app := fiber.New(fiber.Config{
		CaseSensitive:   true,
		StructValidator: sharedValidator.NewStructValidator(),
		ErrorHandler:    response.FiberErrorHandler,
	})
	app.Get("/health", func(c fiber.Ctx) error {
		return response.Success(c, fiber.Map{"status": "healthy"})
	})
	apiGroup := app.Group("/api")
	auth.RegisterRoute(apiGroup, authHandler, authMw)
	profile.RegisterRoute(apiGroup, profileHandler, authMw)
	keys.RegisterRoute(apiGroup, keysHandler, authMw)
	submissions.RegisterRoute(apiGroup, subHandler, authMw)
	stats.RegisterRoute(apiGroup, statsHandler, authMw)
	playground.RegisterRoute(apiGroup, playHandler, authMw)

	doReq := func(method, path string, body any, cookie *http.Cookie, ua string) (*http.Response, []byte) {
		var reader *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			reader = bytes.NewReader(b)
		} else {
			reader = bytes.NewReader(nil)
		}
		req := httptest.NewRequest(method, path, reader)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if ua != "" {
			req.Header.Set("User-Agent", ua)
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
		if err != nil {
			t.Fatalf("%s %s error: %v", method, path, err)
		}
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		_ = resp.Body.Close()
		return resp, buf.Bytes()
	}

	extractTokenCookie := func(r *http.Response) *http.Cookie {
		for _, c := range r.Cookies() {
			if c.Name == "token" {
				return c
			}
		}
		return nil
	}

	// 1. Health & Unauthenticated guard
	r, _ := doReq("GET", "/health", nil, nil, "")
	if r.StatusCode != 200 {
		t.Fatalf("GET /health = %d, want 200", r.StatusCode)
	}
	r, _ = doReq("GET", "/api/auth/me", nil, nil, "")
	if r.StatusCode != 401 {
		t.Fatalf("unauthenticated GET /api/auth/me = %d, want 401", r.StatusCode)
	}

	// 2. Register validation + creation (creates profile + session A)
	r, _ = doReq("POST", "/api/auth/register", map[string]string{"email": "dev@runstack.io", "password": "short"}, nil, "")
	if r.StatusCode != 400 {
		t.Fatalf("weak password register = %d, want 400", r.StatusCode)
	}
	r, _ = doReq("POST", "/api/auth/register", map[string]string{"email": "dev@runstack.io", "password": "Password123"}, nil, "Mozilla/5.0 (Macintosh) Chrome/120")
	if r.StatusCode != 201 {
		t.Fatalf("valid register = %d, want 201", r.StatusCode)
	}
	cookieA := extractTokenCookie(r)
	if cookieA == nil || cookieA.Value == "" || !cookieA.HttpOnly {
		t.Fatalf("expected HttpOnly token cookie from register")
	}
	r, _ = doReq("POST", "/api/auth/register", map[string]string{"email": "dev@runstack.io", "password": "Password123"}, nil, "")
	if r.StatusCode != 409 {
		t.Fatalf("duplicate register = %d, want 409", r.StatusCode)
	}

	// 3. Login from second device -> creates session B and evicts session A immediately
	r, _ = doReq("POST", "/api/auth/login", map[string]string{"email": "dev@runstack.io", "password": "wrong"}, nil, "")
	if r.StatusCode != 401 {
		t.Fatalf("bad login = %d, want 401", r.StatusCode)
	}
	r, _ = doReq("POST", "/api/auth/login", map[string]string{"email": "dev@runstack.io", "password": "Password123"}, nil, "curl/8.4.0")
	if r.StatusCode != 200 {
		t.Fatalf("valid login = %d, want 200", r.StatusCode)
	}
	cookieB := extractTokenCookie(r)
	if cookieB == nil || cookieB.Value == "" || !cookieB.HttpOnly {
		t.Fatalf("expected HttpOnly token cookie from login")
	}

	// Session A must now be rejected (single session at a time)
	r, _ = doReq("GET", "/api/auth/me", nil, cookieA, "")
	if r.StatusCode != 401 {
		t.Fatalf("expected stale session A rejected (401), got %d", r.StatusCode)
	}

	// Session B succeeds on /me & /auth/session metadata endpoint
	r, raw := doReq("GET", "/api/auth/me", nil, cookieB, "")
	if r.StatusCode != 200 {
		t.Fatalf("GET /api/auth/me with cookie B = %d, want 200", r.StatusCode)
	}
	r, raw = doReq("GET", "/api/auth/session", nil, cookieB, "")
	if r.StatusCode != 200 {
		t.Fatalf("GET /api/auth/session = %d, want 200", r.StatusCode)
	}
	var sessEnv response.Response[auth.SessionInfoResponse]
	_ = json.Unmarshal(raw, &sessEnv)
	if sessEnv.Data.DeviceName != "cURL CLI" {
		t.Fatalf("expected cURL CLI device_name in session metadata, got %q", sessEnv.Data.DeviceName)
	}

	// 4. Profile GET & PATCH
	r, raw = doReq("GET", "/api/profile", nil, cookieB, "")
	if r.StatusCode != 200 {
		t.Fatalf("GET /api/profile = %d, want 200 (%s)", r.StatusCode, string(raw))
	}
	r, raw = doReq("PATCH", "/api/profile", map[string]string{
		"username":         "vivek_dev",
		"display_name":     "Vivek",
		"default_language": "go",
		"github_url":       "https://github.com/vivek6201",
	}, cookieB, "")
	if r.StatusCode != 200 {
		t.Fatalf("PATCH /api/profile = %d, want 200 (%s)", r.StatusCode, string(raw))
	}
	var profEnv response.Response[profile.ProfileResponse]
	_ = json.Unmarshal(raw, &profEnv)
	if profEnv.Data.Username != "vivek_dev" || profEnv.Data.DefaultLanguage != "go" {
		t.Fatalf("unexpected updated profile: %+v", profEnv.Data)
	}

	// 5. API Keys CRUD
	r, raw = doReq("POST", "/api/keys", map[string]any{"name": "CI Key", "rate_limit_rpm": 500}, cookieB, "")
	if r.StatusCode != 201 {
		t.Fatalf("POST /api/keys = %d, want 201 (%s)", r.StatusCode, string(raw))
	}
	var createdKeyEnv response.Response[keys.CreateAPIKeyResponse]
	_ = json.Unmarshal(raw, &createdKeyEnv)
	if createdKeyEnv.Data.RateLimitRPM != 60 {
		t.Fatalf("expected free tier cap 60 RPM, got %d", createdKeyEnv.Data.RateLimitRPM)
	}
	r, raw = doReq("GET", "/api/keys", nil, cookieB, "")
	if r.StatusCode != 200 {
		t.Fatalf("GET /api/keys = %d, want 200", r.StatusCode)
	}

	// 6. Playground Execute + Submissions + Stats
	r, _ = doReq("POST", "/api/playground/execute", map[string]string{"language": "python"}, cookieB, "")
	if r.StatusCode != 400 {
		t.Fatalf("invalid playground execute = %d, want 400", r.StatusCode)
	}
	r, raw = doReq("POST", "/api/playground/execute", map[string]string{"language": "python", "code": "print('hello')"}, cookieB, "")
	if r.StatusCode != 200 {
		t.Fatalf("POST /api/playground/execute = %d, want 200 (%s)", r.StatusCode, string(raw))
	}
	r, _ = doReq("GET", "/api/submissions?page=1&limit=10", nil, cookieB, "")
	if r.StatusCode != 200 {
		t.Fatalf("GET /api/submissions = %d, want 200", r.StatusCode)
	}
	r, _ = doReq("GET", "/api/submissions/"+jobID, nil, cookieB, "")
	if r.StatusCode != 200 {
		t.Fatalf("GET /api/submissions/:id = %d, want 200", r.StatusCode)
	}
	r, raw = doReq("GET", "/api/stats", nil, cookieB, "")
	if r.StatusCode != 200 {
		t.Fatalf("GET /api/stats = %d, want 200", r.StatusCode)
	}
	var statsEnv response.Response[stats.DashboardStatsResponse]
	_ = json.Unmarshal(raw, &statsEnv)
	if statsEnv.Data.TotalSubmissions != 1 || statsEnv.Data.SuccessfulRuns != 1 {
		t.Fatalf("unexpected stats after 1 run: %+v", statsEnv.Data)
	}
	r, _ = doReq("GET", "/api/stats/daily?days=7", nil, cookieB, "")
	if r.StatusCode != 200 {
		t.Fatalf("GET /api/stats/daily = %d, want 200", r.StatusCode)
	}

	// 7. Revoke Key & Logout (server-side revocation verifies cookieB fails immediately afterwards)
	r, _ = doReq("DELETE", "/api/keys/invalid-uuid", nil, cookieB, "")
	if r.StatusCode != 400 {
		t.Fatalf("DELETE bad key UUID = %d, want 400", r.StatusCode)
	}
	r, _ = doReq("DELETE", "/api/keys/"+createdKeyEnv.Data.ID.String(), nil, cookieB, "")
	if r.StatusCode != 200 {
		t.Fatalf("DELETE /api/keys/:id = %d, want 200", r.StatusCode)
	}
	r, _ = doReq("POST", "/api/auth/logout", nil, cookieB, "")
	if r.StatusCode != 200 {
		t.Fatalf("POST /api/auth/logout = %d, want 200", r.StatusCode)
	}
	r, _ = doReq("GET", "/api/auth/me", nil, cookieB, "")
	if r.StatusCode != 401 {
		t.Fatalf("expected logged out cookie B rejected (401), got %d", r.StatusCode)
	}
}
