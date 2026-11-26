package playground

import "code-runner/shared/dto"

type ExecuteRequest struct {
	Language      string         `json:"language" validate:"required"`
	Code          string         `json:"code" validate:"required"`
	TestCases     []dto.TestCase `json:"test_cases"`
	TimeLimitMs   int64          `json:"time_limit_ms"`
	MemoryLimitMB int64          `json:"memory_limit_mb"`
}
