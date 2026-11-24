package playground

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"code-runner/dashboard/internal/auth"
	subRepo "code-runner/dashboard/internal/repository/submission"
	"code-runner/shared/cache"
	"code-runner/shared/dto"
	"code-runner/shared/models"
	"code-runner/shared/response"
)

type Handler struct {
	subRepo       subRepo.Repository
	cache         *cache.Cache
	engineBaseURL string
	devBypassKey  string
	httpClient    *http.Client
}

func NewHandler(subRepo subRepo.Repository, cache *cache.Cache, engineBaseURL, devBypassKey string) *Handler {
	if engineBaseURL == "" {
		engineBaseURL = "http://localhost:8081"
	}
	return &Handler{
		subRepo:       subRepo,
		cache:         cache,
		engineBaseURL: engineBaseURL,
		devBypassKey:  devBypassKey,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

type ExecuteRequest struct {
	Language      string         `json:"language"`
	Code          string         `json:"code"`
	TestCases     []dto.TestCase `json:"test_cases"`
	TimeLimitMs   int64          `json:"time_limit_ms"`
	MemoryLimitMB int64          `json:"memory_limit_mb"`
}

func (h *Handler) Execute(c *fiber.Ctx) error {
	claims := auth.GetUserClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthorized")
	}

	var req ExecuteRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid execution request")
	}

	if req.Code == "" || req.Language == "" {
		return response.BadRequest(c, "language and code are required")
	}

	if req.TimeLimitMs <= 0 {
		req.TimeLimitMs = 3000
	}
	if req.MemoryLimitMB <= 0 {
		req.MemoryLimitMB = 128
	}

	// 1. Submit job to Execution Engine API
	submitPayload := dto.SubmitRequest{
		Language:      req.Language,
		Code:          req.Code,
		TestCases:     req.TestCases,
		TimeLimitMs:   req.TimeLimitMs,
		MemoryLimitMB: req.MemoryLimitMB,
	}

	payloadBytes, err := json.Marshal(submitPayload)
	if err != nil {
		return response.InternalServerError(c, "failed to encode job payload")
	}

	submitReq, err := http.NewRequestWithContext(c.Context(), "POST", h.engineBaseURL+"/v1/submit", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return response.InternalServerError(c, "failed to create engine request")
	}
	submitReq.Header.Set("Content-Type", "application/json")
	submitReq.Header.Set("X-API-Key", h.devBypassKey)
	submitReq.Header.Set("X-User-ID", claims.UserID.String())

	resp, err := h.httpClient.Do(submitReq)
	if err != nil {
		return response.BadGateway(c, fmt.Sprintf("execution engine unreachable at %s: %v", h.engineBaseURL, err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return response.Error(c, resp.StatusCode, string(body))
	}

	var env response.Response[dto.SubmitResponse]
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil || !env.Success {
		return response.InternalServerError(c, "failed to parse engine response")
	}

	jobID := env.Data.JobID

	// 2. Poll Execution Engine for completion (up to 15 seconds)
	deadline := time.Now().Add(15 * time.Second)
	var finalJob dto.Job
	completed := false

	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)

		statusReq, err := http.NewRequestWithContext(c.Context(), "GET", h.engineBaseURL+"/v1/status/"+jobID, nil)
		if err != nil {
			break
		}
		statusReq.Header.Set("X-API-Key", h.devBypassKey)

		statusResp, err := h.httpClient.Do(statusReq)
		if err == nil && statusResp.StatusCode == http.StatusOK {
			var statusEnv response.Response[dto.Job]
			if err := json.NewDecoder(statusResp.Body).Decode(&statusEnv); err == nil && statusEnv.Success {
				job := statusEnv.Data
				if job.Status == dto.StatusCompleted || job.Status == dto.StatusFailed || job.Status == dto.StatusCompileError {
					finalJob = job
					completed = true
					_ = statusResp.Body.Close()
					break
				}
			}
			_ = statusResp.Body.Close()
		}
	}

	if !completed {
		return response.GatewayTimeout(c, "execution timed out while waiting for worker response")
	}

	// 3. Map finalJob to models.Submission & persist via repository
	submissionID, err := uuid.Parse(jobID)
	if err != nil {
		submissionID = uuid.New()
	}

	subStatus := models.SubmissionCompleted
	switch finalJob.Status {
	case dto.StatusFailed:
		subStatus = models.SubmissionFailed
	case dto.StatusCompileError:
		subStatus = models.SubmissionFailed
	}

	var testRecords []models.TestCaseResultRecord
	var exitCode int
	var stdout, stderr, errStr string
	var timeTakenMs, memoryUsedKB int64

	if finalJob.Result != nil {
		exitCode = finalJob.Result.ExitCode
		stdout = finalJob.Result.Stdout
		stderr = finalJob.Result.Stderr
		timeTakenMs = finalJob.Result.TimeTakenMs
		memoryUsedKB = finalJob.Result.MemoryUsedKB
		for _, tc := range finalJob.Result.TestResults {
			testRecords = append(testRecords, models.TestCaseResultRecord{
				TestCaseID:     tc.TestCaseID,
				Passed:         tc.Passed,
				ActualOutput:   tc.ActualOutput,
				ExpectedOutput: tc.ExpectedOutput,
				TimeTakenMs:    tc.TimeTakenMs,
			})
		}
	}
	if finalJob.Error != "" {
		errStr = finalJob.Error
	}

	subRecord := models.Submission{
		ID:            submissionID,
		UserID:        claims.UserID,
		APIKeyID:      nil,
		Language:      finalJob.Language,
		Code:          finalJob.Code,
		Status:        subStatus,
		TimeLimitMs:   finalJob.TimeLimitMs,
		MemoryLimitMB: finalJob.MemoryLimitMB,
		ExitCode:      exitCode,
		Stdout:        stdout,
		Stderr:        stderr,
		TimeTakenMs:   timeTakenMs,
		MemoryUsedKB:  memoryUsedKB,
		TestResults:   testRecords,
		Error:         errStr,
		CreatedAt:     finalJob.CreatedAt,
		CompletedAt:   finalJob.CompletedAt,
	}

	if h.subRepo != nil {
		_ = h.subRepo.Create(c.Context(), &subRecord)
	}

	// Update monthly quota in Redis and invalidate cached dashboard queries
	if h.cache != nil {
		now := time.Now().UTC()
		ym := fmt.Sprintf("%04d-%02d", now.Year(), now.Month())
		_ = h.cache.Client().Incr(c.Context(), cache.Key("user", "quota", claims.UserID.String(), ym, "runs")).Err()
		_ = h.cache.Revoke(c.Context(), cache.Key("dashboard", "stats", claims.UserID.String()))
		_ = h.cache.RevokePattern(c.Context(), cache.Key("dashboard", "submissions", claims.UserID.String(), "list")+"*")
		_ = h.cache.RevokePattern(c.Context(), cache.Key("dashboard", "stats", "daily", claims.UserID.String())+"*")
	}

	return response.Success(c, finalJob)
}
