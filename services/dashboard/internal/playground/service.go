package playground

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"code-runner/dashboard/internal/config"
	"code-runner/dashboard/internal/submissions"
	"code-runner/shared/dto"
	"code-runner/shared/models"
	"code-runner/shared/response"
)

var (
	ErrEngineUnreachable = errors.New("execution engine unreachable")
	ErrExecutionTimeout  = errors.New("execution timed out while waiting for worker response")
)

type EngineHTTPError struct {
	StatusCode int
	Body       string
}

func (e *EngineHTTPError) Error() string {
	return e.Body
}

type IPlaygroundService interface {
	Execute(ctx context.Context, userID uuid.UUID, req *ExecuteRequest) (*dto.Job, error)
	SetHTTPClient(client *http.Client)
}

type PlaygroundService struct {
	subRepo    submissions.ISubmissionsRepository
	caches     *Caches
	cfg        *config.ConfigVar
	httpClient *http.Client
}

func NewPlaygroundService(
	subRepo submissions.ISubmissionsRepository,
	caches *Caches,
	cfg *config.ConfigVar,
) IPlaygroundService {
	var cfgCopy config.ConfigVar
	if cfg != nil {
		cfgCopy = *cfg
	}
	if cfgCopy.EngineURL == "" {
		cfgCopy.EngineURL = "http://localhost:8081"
	}

	return &PlaygroundService{
		subRepo: subRepo,
		caches:  caches,
		cfg:     &cfgCopy,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (s *PlaygroundService) SetHTTPClient(client *http.Client) {
	if client != nil {
		s.httpClient = client
	}
}

func (s *PlaygroundService) Execute(ctx context.Context, userID uuid.UUID, req *ExecuteRequest) (*dto.Job, error) {
	submitPayload := buildSubmitPayload(req)

	jobID, err := s.submitToEngine(ctx, userID, &submitPayload)
	if err != nil {
		return nil, err
	}

	finalJob, err := s.pollEngineStatus(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}

	s.persistSubmission(ctx, userID, jobID, finalJob)
	s.caches.RecordRunAndInvalidateAsync(userID)

	return finalJob, nil
}

func buildSubmitPayload(req *ExecuteRequest) dto.SubmitRequest {
	timeLimit := req.TimeLimitMs
	if timeLimit <= 0 {
		timeLimit = 3000
	}
	memLimit := req.MemoryLimitMB
	if memLimit <= 0 {
		memLimit = 128
	}
	return dto.SubmitRequest{
		Language:      req.Language,
		Code:          req.Code,
		TestCases:     req.TestCases,
		TimeLimitMs:   timeLimit,
		MemoryLimitMB: memLimit,
	}
}

func (s *PlaygroundService) newEngineRequest(ctx context.Context, method, path string, userID uuid.UUID, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, s.cfg.EngineURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", s.cfg.DevBypassKey)
	req.Header.Set("X-User-ID", userID.String())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (s *PlaygroundService) submitToEngine(ctx context.Context, userID uuid.UUID, payload *dto.SubmitRequest) (string, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to encode job payload: %w", err)
	}

	submitReq, err := s.newEngineRequest(ctx, http.MethodPost, "/v1/submit", userID, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create engine request: %w", err)
	}

	resp, err := s.httpClient.Do(submitReq)
	if err != nil {
		return "", fmt.Errorf("%w at %s: %v", ErrEngineUnreachable, s.cfg.EngineURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", &EngineHTTPError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	var env response.Response[dto.SubmitResponse]
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil || !env.Success {
		return "", errors.New("failed to parse engine response")
	}

	return env.Data.JobID, nil
}

func (s *PlaygroundService) pollEngineStatus(ctx context.Context, userID uuid.UUID, jobID string) (*dto.Job, error) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}

		job, done := s.fetchCompletedJob(ctx, userID, jobID)
		if done {
			return job, nil
		}
	}

	return nil, ErrExecutionTimeout
}

func (s *PlaygroundService) fetchCompletedJob(ctx context.Context, userID uuid.UUID, jobID string) (*dto.Job, bool) {
	statusReq, err := s.newEngineRequest(ctx, http.MethodGet, "/v1/status/"+jobID, userID, nil)
	if err != nil {
		return nil, false
	}

	statusResp, err := s.httpClient.Do(statusReq)
	if err != nil {
		return nil, false
	}
	defer statusResp.Body.Close()

	if statusResp.StatusCode != http.StatusOK {
		return nil, false
	}

	var statusEnv response.Response[dto.Job]
	if err := json.NewDecoder(statusResp.Body).Decode(&statusEnv); err != nil || !statusEnv.Success {
		return nil, false
	}

	if !isTerminalStatus(statusEnv.Data.Status) {
		return nil, false
	}
	return &statusEnv.Data, true
}

func isTerminalStatus(status dto.JobStatus) bool {
	return status == dto.StatusCompleted || status == dto.StatusFailed || status == dto.StatusCompileError
}

func (s *PlaygroundService) persistSubmission(ctx context.Context, userID uuid.UUID, jobID string, finalJob *dto.Job) {
	if s.subRepo == nil || finalJob == nil {
		return
	}
	record := mapSubmissionRecord(userID, jobID, finalJob)
	_ = s.subRepo.Create(ctx, &record)
}

func mapSubmissionRecord(userID uuid.UUID, jobID string, finalJob *dto.Job) models.Submission {
	submissionID, err := uuid.Parse(jobID)
	if err != nil {
		submissionID = uuid.New()
	}

	subStatus := models.SubmissionCompleted
	if finalJob.Status == dto.StatusFailed || finalJob.Status == dto.StatusCompileError {
		subStatus = models.SubmissionFailed
	}

	subRecord := models.Submission{
		ID:            submissionID,
		UserID:        userID,
		Language:      finalJob.Language,
		Code:          finalJob.Code,
		Status:        subStatus,
		TimeLimitMs:   finalJob.TimeLimitMs,
		MemoryLimitMB: finalJob.MemoryLimitMB,
		Error:         finalJob.Error,
		CreatedAt:     finalJob.CreatedAt,
		CompletedAt:   finalJob.CompletedAt,
	}

	if finalJob.Result == nil {
		return subRecord
	}

	subRecord.ExitCode = finalJob.Result.ExitCode
	subRecord.Stdout = finalJob.Result.Stdout
	subRecord.Stderr = finalJob.Result.Stderr
	subRecord.TimeTakenMs = finalJob.Result.TimeTakenMs
	subRecord.MemoryUsedKB = finalJob.Result.MemoryUsedKB
	for _, tc := range finalJob.Result.TestResults {
		subRecord.TestResults = append(subRecord.TestResults, models.TestCaseResultRecord{
			TestCaseID:     tc.TestCaseID,
			Passed:         tc.Passed,
			ActualOutput:   tc.ActualOutput,
			ExpectedOutput: tc.ExpectedOutput,
			TimeTakenMs:    tc.TimeTakenMs,
		})
	}
	return subRecord
}
