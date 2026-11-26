package worker

import (
	"context"
	"fmt"
	"time"

	"code-runner/execution-engine/internal/ports"
	"code-runner/shared/dto"
	"code-runner/shared/logger"
)

// Worker continuously consumes jobs from the queue and executes them using the injected Executor adapter.
type Worker struct {
	id       int
	queue    ports.JobQueue
	store    ports.StateStore
	executor ports.Executor
}

func NewWorker(id int, queue ports.JobQueue, store ports.StateStore, executor ports.Executor) *Worker {
	return &Worker{
		id:       id,
		queue:    queue,
		store:    store,
		executor: executor,
	}
}

func (w *Worker) Start(ctx context.Context) {
	logger.Info(fmt.Sprintf("Worker #%d started with executor [%s]", w.id, w.executor.Name()))

	for {
		if ctx.Err() != nil {
			logger.Info(fmt.Sprintf("Worker #%d stopping...", w.id))
			return
		}
		w.pollAndProcessOnce(ctx)
	}
}

func (w *Worker) pollAndProcessOnce(ctx context.Context) {
	job, err := w.queue.Dequeue(ctx)
	if err != nil {
		time.Sleep(500 * time.Millisecond)
		return
	}
	if job == nil {
		return
	}
	w.processJob(ctx, job)
}

func (w *Worker) processJob(ctx context.Context, job *dto.Job) {
	logger.Info(fmt.Sprintf("[Worker #%d] Processing job %s (lang: %s)", w.id, job.ID, job.Language))

	if err := w.store.SetStatus(ctx, job.ID, dto.StatusRunning); err != nil {
		logger.Info(fmt.Sprintf("[Worker #%d] Failed to set status RUNNING: %v", w.id, err))
	}

	result, err := w.executor.Execute(ctx, job)
	if err != nil {
		logger.Info(fmt.Sprintf("[Worker #%d] Execution failed for job %s: %v", w.id, job.ID, err))
		_ = w.store.SetError(ctx, job.ID, err.Error())
		w.ackJob(job.ID)
		return
	}

	if err := w.store.SetResult(ctx, job.ID, result); err != nil {
		logger.Info(fmt.Sprintf("[Worker #%d] Failed to set result: %v", w.id, err))
		return
	}

	w.ackJob(job.ID)
	logger.Info(fmt.Sprintf("[Worker #%d] Job %s completed in %d ms", w.id, job.ID, result.TimeTakenMs))
}

func (w *Worker) ackJob(jobID string) {
	ackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := w.queue.Ack(ackCtx, jobID); err != nil {
		logger.Info(fmt.Sprintf("[Worker #%d] Failed to ack job %s: %v", w.id, jobID, err))
	}
}
