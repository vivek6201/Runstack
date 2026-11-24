package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"code-runner/shared/cache"
	"code-runner/shared/dto"
)

const jobStateTTL = 24 * time.Hour

// RedisQueueAdapter implements both ports.JobQueue and ports.StateStore using Redis.
type RedisQueueAdapter struct {
	client *redis.Client
}

func NewRedisQueueAdapter(client *redis.Client) *RedisQueueAdapter {
	return &RedisQueueAdapter{client: client}
}

// -------------------------------------------------------------
// ports.JobQueue implementation
// -------------------------------------------------------------

func (r *RedisQueueAdapter) Enqueue(ctx context.Context, job *dto.Job) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job for queue: %w", err)
	}

	// 1. Save initial state in Redis with 24hr TTL
	if err := r.saveJobState(ctx, job); err != nil {
		return err
	}

	// 2. Push to the Redis pending queue
	if err := r.client.RPush(ctx, cache.QueueJobs, payload).Err(); err != nil {
		return fmt.Errorf("failed to push to redis queue: %w", err)
	}

	return nil
}

func (r *RedisQueueAdapter) Dequeue(ctx context.Context) (*dto.Job, error) {
	// Blocking pop with a 2-second timeout to allow graceful worker shutdown check
	res, err := r.client.BLPop(ctx, 2*time.Second, cache.QueueJobs).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // Queue was empty during window
		}
		return nil, err
	}

	if len(res) < 2 {
		return nil, nil
	}

	var job dto.Job
	if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dequeued job: %w", err)
	}

	return &job, nil
}

// -------------------------------------------------------------
// ports.StateStore implementation
// -------------------------------------------------------------

func (r *RedisQueueAdapter) SetStatus(ctx context.Context, jobID string, status dto.JobStatus) error {
	job, err := r.GetJob(ctx, jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.Status = status
	now := time.Now()
	if status == dto.StatusRunning {
		job.StartedAt = &now
	}

	return r.saveJobState(ctx, job)
}

func (r *RedisQueueAdapter) SetResult(ctx context.Context, jobID string, result *dto.ExecutionResult) error {
	job, err := r.GetJob(ctx, jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found: %s", jobID)
	}

	now := time.Now()
	job.Status = dto.StatusCompleted
	job.Result = result
	job.CompletedAt = &now

	return r.saveJobState(ctx, job)
}

func (r *RedisQueueAdapter) SetError(ctx context.Context, jobID string, errMessage string) error {
	job, err := r.GetJob(ctx, jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found: %s", jobID)
	}

	now := time.Now()
	job.Status = dto.StatusFailed
	job.Error = errMessage
	job.CompletedAt = &now

	return r.saveJobState(ctx, job)
}

func (r *RedisQueueAdapter) GetJob(ctx context.Context, jobID string) (*dto.Job, error) {
	key := cache.Key("job", "state", jobID)
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // Job not found
		}
		return nil, err
	}

	var job dto.Job
	if err := json.Unmarshal([]byte(val), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job state: %w", err)
	}

	return &job, nil
}

func (r *RedisQueueAdapter) saveJobState(ctx context.Context, job *dto.Job) error {
	key := cache.Key("job", "state", job.ID)
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to serialize job: %w", err)
	}

	return r.client.Set(ctx, key, data, jobStateTTL).Err()
}
