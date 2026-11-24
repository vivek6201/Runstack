package firecracker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code-runner/shared/logger"
)

// Pool maintains a warm pool of pre-booted, idle Firecracker microVMs ready for instant execution.
type Pool struct {
	cfg          *Config
	poolChan     chan *MicroVM
	mu           sync.Mutex
	isClosed     bool
	lastSpawnErr error
}

// NewPool initializes the pool and starts background spawning of warm microVMs.
func NewPool(ctx context.Context, cfg *Config) (*Pool, error) {
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 2
	}

	p := &Pool{
		cfg:      cfg,
		poolChan: make(chan *MicroVM, cfg.PoolSize),
	}

	logger.Info(fmt.Sprintf("Pre-warming %d Firecracker microVMs in background...", cfg.PoolSize))

	// Pre-boot initial warm VMs
	for i := 0; i < cfg.PoolSize; i++ {
		go p.spawnAndEnqeue(ctx)
	}

	return p, nil
}

// Acquire pops an idle, pre-warmed microVM from the pool.
func (p *Pool) Acquire(ctx context.Context) (*MicroVM, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case vm := <-p.poolChan:
		return vm, nil
	case <-time.After(10 * time.Second):
		p.mu.Lock()
		lastErr := p.lastSpawnErr
		p.mu.Unlock()
		if lastErr != nil {
			return nil, fmt.Errorf("timeout waiting for an idle microVM from warm pool (last spawn error: %v)", lastErr)
		}
		return nil, fmt.Errorf("timeout waiting for an idle microVM from warm pool")
	}
}

// ReleaseAndReplace destroys the used (dirty) VM and asynchronously boots a clean replacement.
func (p *Pool) ReleaseAndReplace(vm *MicroVM) {
	go func() {
		// 1. Terminate dirty VM immediately
		if vm != nil {
			vm.Stop()
		}

		// 2. Boot a fresh replacement VM into the pool
		p.mu.Lock()
		closed := p.isClosed
		p.mu.Unlock()

		if !closed {
			p.spawnAndEnqeue(context.Background())
		}
	}()
}

// Shutdown terminates all idle microVMs in the pool.
func (p *Pool) Shutdown() {
	p.mu.Lock()
	p.isClosed = true
	p.mu.Unlock()

	close(p.poolChan)
	for vm := range p.poolChan {
		if vm != nil {
			vm.Stop()
		}
	}
	logger.Info("Firecracker warm pool shutdown complete.")
}

func (p *Pool) spawnAndEnqeue(ctx context.Context) {
	vm, err := Spawn(ctx, p.cfg)
	if err != nil {
		p.mu.Lock()
		p.lastSpawnErr = err
		p.mu.Unlock()

		logger.Info(fmt.Sprintf("Warning: failed to spawn warm microVM: %v", err))
		// Retry after a backoff
		time.Sleep(2 * time.Second)
		p.spawnAndEnqeue(ctx)
		return
	}

	p.poolChan <- vm
	logger.Info(fmt.Sprintf("Warm microVM [%s] ready in pool", vm.ID))
}
