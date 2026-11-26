package ports

import (
	"context"

	"code-runner/shared/dto"
)

// JobQueue defines how jobs are enqueued, dequeued, and acknowledged asynchronously.
type JobQueue interface {
	Enqueue(ctx context.Context, job *dto.Job) error
	Dequeue(ctx context.Context) (*dto.Job, error)
	Ack(ctx context.Context, jobID string) error
}

// StateStore defines how active execution states are tracked in-flight.
type StateStore interface {
	SetStatus(ctx context.Context, jobID string, status dto.JobStatus) error
	SetResult(ctx context.Context, jobID string, result *dto.ExecutionResult) error
	SetError(ctx context.Context, jobID string, errMessage string) error
	GetJob(ctx context.Context, jobID string) (*dto.Job, error)
}
