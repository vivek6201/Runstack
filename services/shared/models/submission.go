package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type SubmissionStatus string

const (
	SubmissionQueued    SubmissionStatus = "QUEUED"
	SubmissionRunning   SubmissionStatus = "RUNNING"
	SubmissionCompleted SubmissionStatus = "COMPLETED"
	SubmissionFailed    SubmissionStatus = "FAILED"
	SubmissionTimedOut  SubmissionStatus = "TIMED_OUT"
)

// TestCaseResultRecord represents a single test case outcome stored inside JSONB.
type TestCaseResultRecord struct {
	TestCaseID     string `json:"test_case_id"`
	Passed         bool   `json:"passed"`
	ActualOutput   string `json:"actual_output"`
	ExpectedOutput string `json:"expected_output"`
	TimeTakenMs    int64  `json:"time_taken_ms"`
}

// JSONBTestResults implements sql.Scanner and driver.Valuer for PostgreSQL JSONB.
type JSONBTestResults []TestCaseResultRecord

func (j JSONBTestResults) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "[]", nil
	}
	return json.Marshal(j)
}

func (j *JSONBTestResults) Scan(value interface{}) error {
	if value == nil {
		*j = []TestCaseResultRecord{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for JSONBTestResults")
	}
	return json.Unmarshal(bytes, j)
}

// Submission stores permanent execution records in PostgreSQL.
type Submission struct {
	ID        uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID    uuid.UUID        `gorm:"type:uuid;index;not null" json:"user_id"`
	APIKeyID  *uuid.UUID       `gorm:"type:uuid;index" json:"api_key_id,omitempty"`
	Language  string           `gorm:"type:varchar(32);not null;index" json:"language"`
	Code      string           `gorm:"type:text;not null" json:"code"`
	Status    SubmissionStatus `gorm:"type:varchar(32);not null;index" json:"status"`

	// Resource bounds configured for this submission
	TimeLimitMs   int64 `gorm:"not null" json:"time_limit_ms"`
	MemoryLimitMB int64 `gorm:"not null" json:"memory_limit_mb"`

	// Execution metrics
	ExitCode     int              `json:"exit_code"`
	Stdout       string           `gorm:"type:text" json:"stdout"`
	Stderr       string           `gorm:"type:text" json:"stderr"`
	Error        string           `gorm:"type:text" json:"error,omitempty"`
	TimeTakenMs  int64            `gorm:"index" json:"time_taken_ms"`
	MemoryUsedKB int64            `json:"memory_used_kb"`
	TestResults  JSONBTestResults `gorm:"type:jsonb" json:"test_results,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `gorm:"autoCreateTime;index" json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
