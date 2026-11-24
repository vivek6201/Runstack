package worker

import (
	"context"
	"sync"

	"code-runner/execution-engine/internal/ports"
	"code-runner/shared/logger"
)

// Pool orchestrates multiple concurrent Worker goroutines.
type Pool struct {
	concurrency int
	queue       ports.JobQueue
	store       ports.StateStore
	executor    ports.Executor
	wg          sync.WaitGroup
}

func NewPool(concurrency int, queue ports.JobQueue, store ports.StateStore, executor ports.Executor) *Pool {
	if concurrency <= 0 {
		concurrency = 4
	}
	return &Pool{
		concurrency: concurrency,
		queue:       queue,
		store:       store,
		executor:    executor,
	}
}

func (p *Pool) Start(ctx context.Context) {
	logger.Info("Starting worker pool...")
	for i := 1; i <= p.concurrency; i++ {
		w := NewWorker(i, p.queue, p.store, p.executor)
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			w.Start(ctx)
		}()
	}
}

func (p *Pool) Wait() {
	p.wg.Wait()
	logger.Info("Worker pool gracefully stopped.")
}

func (p *Pool) Stop() {
	p.Wait()
}
