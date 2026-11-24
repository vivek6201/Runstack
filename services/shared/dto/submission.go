package dto

import (
	"time"

	"github.com/google/uuid"

	"code-runner/shared/models"
)

// SubmissionListQuery parses query string parameters for paginated submission history.
type SubmissionListQuery struct {
	Page     int    `json:"page"`
	Limit    int    `json:"limit"`
	Status   string `json:"status,omitempty"`
	Language string `json:"language,omitempty"`
}

// SubmissionSummaryResponse provides lightweight metadata for history tables.
type SubmissionSummaryResponse struct {
	ID              uuid.UUID               `json:"id"`
	Language        string                  `json:"language"`
	Status          models.SubmissionStatus `json:"status"`
	ExitCode        int                     `json:"exit_code"`
	TimeTakenMs     int64                   `json:"time_taken_ms"`
	MemoryUsedKB    int64                   `json:"memory_used_kb"`
	TotalTestCases  int                     `json:"total_test_cases"`
	PassedTestCases int                     `json:"passed_test_cases"`
	CreatedAt       time.Time               `json:"created_at"`
	CompletedAt     *time.Time              `json:"completed_at,omitempty"`
}

// SubmissionDetailResponse provides full inspection data including source code, stdout, and stderr.
type SubmissionDetailResponse struct {
	ID            uuid.UUID                   `json:"id"`
	UserID        uuid.UUID                   `json:"user_id"`
	APIKeyID      *uuid.UUID                  `json:"api_key_id,omitempty"`
	Language      string                      `json:"language"`
	Code          string                      `json:"code"`
	Status        models.SubmissionStatus     `json:"status"`
	TimeLimitMs   int64                       `json:"time_limit_ms"`
	MemoryLimitMB int64                       `json:"memory_limit_mb"`
	ExitCode      int                         `json:"exit_code"`
	Stdout        string                      `json:"stdout"`
	Stderr        string                      `json:"stderr"`
	Error         string                      `json:"error,omitempty"`
	TimeTakenMs   int64                       `json:"time_taken_ms"`
	MemoryUsedKB  int64                       `json:"memory_used_kb"`
	TestResults   models.JSONBTestResults     `json:"test_results,omitempty"`
	CreatedAt     time.Time                   `json:"created_at"`
	StartedAt     *time.Time                  `json:"started_at,omitempty"`
	CompletedAt   *time.Time                  `json:"completed_at,omitempty"`
}
