package firecracker

import (
	"context"
	"fmt"
	"os"

	"code-runner/shared/dto"
)

// FirecrackerAdapter implements ports.Executor using a pool of AWS Firecracker microVMs.
// Each execution runs inside a hardware-isolated microVM connected over virtio-vsock.
type FirecrackerAdapter struct {
	cfg  *Config
	pool *Pool
}

// NewFirecrackerAdapter initializes the microVM pool and returns an implementation of ports.Executor.
func NewFirecrackerAdapter(ctx context.Context, cfg *Config) (*FirecrackerAdapter, error) {
	if cfg == nil {
		cfg = NewConfigFromEnv()
	}

	// Validate host environment (KVM, binary, kernel, rootfs)
	if err := CheckPrerequisites(cfg); err != nil {
		return nil, err
	}

	pool, err := NewPool(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firecracker pool: %w", err)
	}

	return &FirecrackerAdapter{
		cfg:  cfg,
		pool: pool,
	}, nil
}

func (a *FirecrackerAdapter) Name() string {
	return "firecracker-microvm"
}

// Execute claims a pre-booted clean microVM from the pool, pipes code over vsock,
// and schedules the dirty VM for immediate destruction and background replacement.
func (a *FirecrackerAdapter) Execute(ctx context.Context, job *dto.Job) (*dto.ExecutionResult, error) {
	// 1. Acquire warm microVM from pool (< 1ms latency)
	vm, err := a.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire warm microvm: %w", err)
	}

	// 2. Ensure the VM is destroyed and replaced after execution (Zero-Trust isolation)
	defer a.pool.ReleaseAndReplace(vm)

	// 3. Stream payload over virtio-vsock to the Guest Agent
	result, err := ExecuteOverVsock(ctx, vm.VsockPath, job)
	if err != nil {
		return nil, fmt.Errorf("vsock execution error: %w", err)
	}

	return result, nil
}

// Close gracefully stops all VMs in the pool.
func (a *FirecrackerAdapter) Close() {
	if a.pool != nil {
		a.pool.Shutdown()
	}
}

// CheckPrerequisites verifies that Linux KVM, the Firecracker binary, kernel, and rootfs exist.
func CheckPrerequisites(cfg *Config) error {
	// 1. Check /dev/kvm (Required for hardware virtualization)
	if _, err := os.Stat("/dev/kvm"); err != nil {
		return fmt.Errorf("Linux KVM is not available (/dev/kvm not found): Firecracker requires a Linux host with KVM enabled")
	}

	// 2. Check Firecracker executable
	if _, err := os.Stat(cfg.BinaryPath); err != nil {
		return fmt.Errorf("firecracker executable not found at %s: install firecracker or update FIRECRACKER_BIN", cfg.BinaryPath)
	}

	// 3. Check Kernel image
	if _, err := os.Stat(cfg.KernelPath); err != nil {
		return fmt.Errorf("vmlinux kernel not found at %s: run scripts/firecracker/download-kernel.sh or check volume mount", cfg.KernelPath)
	}

	// 4. Check Rootfs ext4 image
	if _, err := os.Stat(cfg.RootfsPath); err != nil {
		return fmt.Errorf("rootfs.ext4 disk not found at %s: run scripts/firecracker/build-rootfs.sh or check volume mount", cfg.RootfsPath)
	}

	return nil
}
