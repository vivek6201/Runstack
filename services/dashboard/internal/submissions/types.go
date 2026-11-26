package submissions

import (
	"time"

	"github.com/google/uuid"

	"code-runner/shared/models"
)

type SubmissionListQuery struct {
	Page     int    `json:"page" query:"page"`
	Limit    int    `json:"limit" query:"limit"`
	Status   string `json:"status,omitempty" query:"status"`
	Language string `json:"language,omitempty" query:"language"`
}

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

type PaginatedSubmissions struct {
	Items []SubmissionSummaryResponse `json:"items"`
	Total int64                       `json:"total"`
}

type SubmissionDetailResponse struct {
	ID            uuid.UUID               `json:"id"`
	UserID        uuid.UUID               `json:"user_id"`
	APIKeyID      *uuid.UUID              `json:"api_key_id,omitempty"`
	Language      string                  `json:"language"`
	Code          string                  `json:"code"`
	Status        models.SubmissionStatus `json:"status"`
	TimeLimitMs   int64                   `json:"time_limit_ms"`
	MemoryLimitMB int64                   `json:"memory_limit_mb"`
	ExitCode      int                     `json:"exit_code"`
	Stdout        string                  `json:"stdout"`
	Stderr        string                  `json:"stderr"`
	Error         string                  `json:"error,omitempty"`
	TimeTakenMs   int64                   `json:"time_taken_ms"`
	MemoryUsedKB  int64                   `json:"memory_used_kb"`
	TestResults   models.JSONBTestResults `json:"test_results,omitempty"`
	CreatedAt     time.Time               `json:"created_at"`
	StartedAt     *time.Time              `json:"started_at,omitempty"`
	CompletedAt   *time.Time              `json:"completed_at,omitempty"`
}

type DailyStatRow struct {
	DateStr  string  `gorm:"column:d"`
	Total    int64   `gorm:"column:total"`
	Failed   int64   `gorm:"column:failed"`
	AvgTime  float64 `gorm:"column:avg_time"`
}

type StatsAggregate struct {
	TotalSubmissions int64
	SuccessfulRuns   int64
	FailedRuns       int64
	TimeoutRuns      int64
	AvgTimeMs        float64
	TotalTimeMs      int64
}
