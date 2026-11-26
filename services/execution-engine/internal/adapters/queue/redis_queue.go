package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"code-runner/shared/cache"
	"code-runner/shared/dto"
)

const jobStateTTL = 24 * time.Hour

// RedisQueueAdapter implements both ports.JobQueue and ports.StateStore using Redis.
type RedisQueueAdapter struct {
	client   *redis.Client
	mu       sync.Mutex
	inFlight map[string]string
}

func NewRedisQueueAdapter(client *redis.Client) *RedisQueueAdapter {
	return &RedisQueueAdapter{
		client:   client,
		inFlight: make(map[string]string),
	}
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
	// Atomically move from pending queue to processing list so crashes don't drop jobs.
	rawPayload, err := r.client.BLMove(ctx, cache.QueueJobs, cache.QueueJobsProcessing, "LEFT", "RIGHT", 2*time.Second).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // Queue was empty during window
		}
		return nil, err
	}

	var job dto.Job
	if err := json.Unmarshal([]byte(rawPayload), &job); err != nil {
		// Remove poisoned payload from processing list so it doesn't stall recovery.
		_ = r.client.LRem(ctx, cache.QueueJobsProcessing, 1, rawPayload).Err()
		return nil, fmt.Errorf("failed to unmarshal dequeued job: %w", err)
	}

	r.mu.Lock()
	r.inFlight[job.ID] = rawPayload
	r.mu.Unlock()

	return &job, nil
}

// Ack removes a completed or failed job payload from the processing list.
func (r *RedisQueueAdapter) Ack(ctx context.Context, jobID string) error {
	r.mu.Lock()
	rawPayload, ok := r.inFlight[jobID]
	if ok {
		delete(r.inFlight, jobID)
	}
	r.mu.Unlock()

	if !ok {
		return nil
	}
	return r.client.LRem(ctx, cache.QueueJobsProcessing, 1, rawPayload).Err()
}

// RequeueStale scans the processing list and re-enqueues jobs orphaned by crashed workers.
func (r *RedisQueueAdapter) RequeueStale(ctx context.Context, staleAfter time.Duration) (int, error) {
	items, err := r.client.LRange(ctx, cache.QueueJobsProcessing, 0, -1).Result()
	if err != nil {
		return 0, err
	}

	requeued := 0
	cutoff := time.Now().Add(-staleAfter)
	for _, rawPayload := range items {
		if r.requeueItemIfStale(ctx, rawPayload, cutoff) {
			requeued++
		}
	}

	return requeued, nil
}

func (r *RedisQueueAdapter) requeueItemIfStale(ctx context.Context, rawPayload string, cutoff time.Time) bool {
	var queuedJob dto.Job
	if err := json.Unmarshal([]byte(rawPayload), &queuedJob); err != nil {
		_ = r.client.LRem(ctx, cache.QueueJobsProcessing, 1, rawPayload).Err()
		return false
	}

	state, err := r.GetJob(ctx, queuedJob.ID)
	if err != nil {
		return false
	}

	if state == nil || isTerminalState(state.Status) {
		_ = r.client.LRem(ctx, cache.QueueJobsProcessing, 1, rawPayload).Err()
		return false
	}

	refTime := state.CreatedAt
	if state.StartedAt != nil {
		refTime = *state.StartedAt
	}
	if !refTime.Before(cutoff) {
		return false
	}

	if err := r.client.RPush(ctx, cache.QueueJobs, rawPayload).Err(); err != nil {
		return false
	}
	_ = r.client.LRem(ctx, cache.QueueJobsProcessing, 1, rawPayload).Err()
	return true
}

func isTerminalState(status dto.JobStatus) bool {
	return status == dto.StatusCompleted || status == dto.StatusFailed || status == dto.StatusCompileError
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
