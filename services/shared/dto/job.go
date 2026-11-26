package dto

import "time"

type JobStatus string

const (
	StatusQueued       JobStatus = "QUEUED"
	StatusRunning      JobStatus = "RUNNING"
	StatusCompleted    JobStatus = "COMPLETED"
	StatusFailed       JobStatus = "FAILED"
	StatusCompileError JobStatus = "COMPILE_ERROR"
)

// Standard supported language identifiers
const (
	LangPython     = "python"
	LangJavaScript = "javascript"
	LangTypeScript = "typescript"
	LangC          = "c"
	LangCPP        = "cpp"
	LangGo         = "go"
	LangJava       = "java"
)

type TestCase struct {
	ID             string `json:"id"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
}

type TestResult struct {
	TestCaseID     string `json:"test_case_id"`
	Passed         bool   `json:"passed"`
	ActualOutput   string `json:"actual_output"`
	ExpectedOutput string `json:"expected_output"`
	TimeTakenMs    int64  `json:"time_taken_ms"`
}

type SubmitRequest struct {
	Language      string     `json:"language" validate:"required"` // "python", "javascript", "cpp"
	Code          string     `json:"code" validate:"required"`     // Raw code
	TestCases     []TestCase `json:"test_cases,omitempty"`         // Test cases
	TimeLimitMs   int64      `json:"time_limit_ms"`                // Timeout (e.g. 2000)
	MemoryLimitMB int64      `json:"memory_limit_mb"`              // RAM limit (e.g. 128)
}

type SubmitResponse struct {
	JobID  string    `json:"job_id"`
	Status JobStatus `json:"status"`
}

type ExecutionResult struct {
	Stdout            string       `json:"stdout"`
	Stderr            string       `json:"stderr"`
	ExitCode          int          `json:"exit_code"`
	TimeTakenMs       int64        `json:"time_taken_ms"`
	MemoryUsedKB      int64        `json:"memory_used_kb"`
	CompilationTimeMs int64        `json:"compilation_time_ms,omitempty"`
	TestResults       []TestResult `json:"test_results,omitempty"`
}

// In-flight Job representation used inside Queue and Engine
type Job struct {
	ID            string           `json:"id"`
	UserID        string           `json:"user_id,omitempty"`
	APIKeyID      string           `json:"api_key_id,omitempty"`
	Language      string           `json:"language"`
	Code          string           `json:"code"`
	TestCases     []TestCase       `json:"test_cases,omitempty"`
	TimeLimitMs   int64            `json:"time_limit_ms"`
	MemoryLimitMB int64            `json:"memory_limit_mb"`
	Status        JobStatus        `json:"status"`
	Error         string           `json:"error,omitempty"`
	Result        *ExecutionResult `json:"result,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	StartedAt     *time.Time       `json:"started_at,omitempty"`
	CompletedAt   *time.Time       `json:"completed_at,omitempty"`
}
