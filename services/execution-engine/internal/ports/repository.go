package ports

import (
	"context"

	"code-runner/shared/dto"
	"code-runner/shared/models"
)

// SubmissionRepository defines persistence operations for completed runs.
type SubmissionRepository interface {
	CreateQueued(ctx context.Context, job *dto.Job) error
	UpdateCompleted(ctx context.Context, jobID string, result *dto.ExecutionResult) error
	UpdateFailed(ctx context.Context, jobID string, errMessage string) error
	FindByID(ctx context.Context, jobID string) (*models.Submission, error)
}
