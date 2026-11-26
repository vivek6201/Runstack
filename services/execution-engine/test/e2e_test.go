package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"code-runner/execution-engine/internal/adapters/executor"
	"code-runner/execution-engine/internal/api"
	"code-runner/execution-engine/internal/worker"
	"code-runner/shared/dto"
	"code-runner/shared/response"
	sharedValidator "code-runner/shared/validator"
)

// In-memory queue with channels for a true concurrent E2E test
type channelQueueStore struct {
	mu       sync.RWMutex
	jobs     map[string]*dto.Job
	jobChan  chan *dto.Job
}

func newChannelQueueStore() *channelQueueStore {
	return &channelQueueStore{
		jobs:    make(map[string]*dto.Job),
		jobChan: make(chan *dto.Job, 100),
	}
}

func (c *channelQueueStore) Enqueue(ctx context.Context, job *dto.Job) error {
	c.mu.Lock()
	c.jobs[job.ID] = job
	c.mu.Unlock()
	c.jobChan <- job
	return nil
}

func (c *channelQueueStore) Dequeue(ctx context.Context) (*dto.Job, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case job := <-c.jobChan:
		return job, nil
	}
}

func (c *channelQueueStore) Ack(ctx context.Context, jobID string) error {
	return nil
}

func (c *channelQueueStore) SetStatus(ctx context.Context, jobID string, status dto.JobStatus) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if j, ok := c.jobs[jobID]; ok {
		j.Status = status
	}
	return nil
}

func (c *channelQueueStore) SetResult(ctx context.Context, jobID string, result *dto.ExecutionResult) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if j, ok := c.jobs[jobID]; ok {
		j.Status = dto.StatusCompleted
		j.Result = result
	}
	return nil
}

func (c *channelQueueStore) SetError(ctx context.Context, jobID string, errMessage string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if j, ok := c.jobs[jobID]; ok {
		j.Status = dto.StatusFailed
		j.Error = errMessage
	}
	return nil
}

func (c *channelQueueStore) GetJob(ctx context.Context, jobID string) (*dto.Job, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if j, ok := c.jobs[jobID]; ok {
		copyJob := *j
		return &copyJob, nil
	}
	return nil, nil
}

func TestE2E_FullFlow(t *testing.T) {
	// 1. Initialize channel-backed queue and store
	qs := newChannelQueueStore()

	// 2. Start worker pool with local executor
	localExec := executor.NewLocalSubprocessAdapter()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workerPool := worker.NewPool(2, qs, qs, localExec)
	workerPool.Start(ctx)

	// 3. Setup Fiber API
	handler := api.NewHandler(qs, qs)
	app := fiber.New(fiber.Config{
		StructValidator: sharedValidator.NewStructValidator(),
	})
	app.Post("/v1/submit", handler.SubmitHandler)
	app.Get("/v1/status/:id", handler.StatusHandler)

	// 4. Submit Python code via API
	payload := dto.SubmitRequest{
		Language: "python",
		Code: `import sys
val = int(sys.stdin.read().strip())
print(val + 100)
`,
		TimeLimitMs: 3000,
		TestCases: []dto.TestCase{
			{ID: "tc-1", Input: "25\n", ExpectedOutput: "125\n"},
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/submit", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("submit request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
	}

	var env response.Response[dto.SubmitResponse]
	json.NewDecoder(resp.Body).Decode(&env)
	jobID := env.Data.JobID

	// 5. Poll until completed (timeout after 5 seconds)
	var finalJob *dto.Job
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		statusReq := httptest.NewRequest("GET", "/v1/status/"+jobID, nil)
		statusResp, err := app.Test(statusReq)
		if err != nil {
			t.Fatalf("status request failed: %v", err)
		}

		var statusEnv response.Response[dto.Job]
		json.NewDecoder(statusResp.Body).Decode(&statusEnv)

		if statusEnv.Success && statusEnv.Data.Status == dto.StatusCompleted {
			copyJob := statusEnv.Data
			finalJob = &copyJob
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if finalJob == nil {
		t.Fatalf("job did not complete within 5 seconds")
	}

	if finalJob.Result == nil {
		t.Fatalf("job completed but result is nil")
	}

	if len(finalJob.Result.TestResults) != 1 || !finalJob.Result.TestResults[0].Passed {
		t.Errorf("expected test case to pass, got: %+v", finalJob.Result.TestResults)
	}

	if finalJob.Result.Stdout != "125\n" {
		t.Errorf("expected stdout '125\\n', got %q", finalJob.Result.Stdout)
	}
}
