package firecracker

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// MicroVM manages the lifecycle of a single Firecracker process and its communication sockets.
type MicroVM struct {
	ID          string
	SocketPath  string // API control socket
	VsockPath   string // Virtio-vsock data socket
	ConsolePath string // Serial console log file
	ConsoleFile *os.File
	Cmd         *exec.Cmd
	Client      *Client
	CreatedAt   time.Time
}

// Spawn boots a new microVM instance and configures its hardware, kernel, rootfs, and vsock.
func Spawn(ctx context.Context, cfg *Config) (*MicroVM, error) {
	vmID := uuid.New().String()[:8]
	socketPath := filepath.Join(cfg.SocketDir, fmt.Sprintf("fc-%s.sock", vmID))
	vsockPath := filepath.Join(cfg.SocketDir, fmt.Sprintf("vsock-%s.sock", vmID))
	consolePath := filepath.Join(cfg.SocketDir, fmt.Sprintf("fc-%s-console.log", vmID))

	// Clean up any stale sockets or logs with the same name
	_ = os.Remove(socketPath)
	_ = os.Remove(vsockPath)
	_ = os.Remove(consolePath)

	consoleFile, _ := os.OpenFile(consolePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

	// 1. Launch the Firecracker binary process
	cmd := exec.CommandContext(ctx, cfg.BinaryPath, "--api-sock", socketPath)
	var stderrBuf bytes.Buffer
	if consoleFile != nil {
		cmd.Stdout = consoleFile
		cmd.Stderr = consoleFile
	} else {
		cmd.Stderr = &stderrBuf
		cmd.Stdout = nil
	}

	if err := cmd.Start(); err != nil {
		if consoleFile != nil {
			_ = consoleFile.Close()
		}
		return nil, fmt.Errorf("failed to start firecracker binary (%s): %w", cfg.BinaryPath, err)
	}

	vm := &MicroVM{
		ID:          vmID,
		SocketPath:  socketPath,
		VsockPath:   vsockPath,
		ConsolePath: consolePath,
		ConsoleFile: consoleFile,
		Cmd:         cmd,
		Client:      NewClient(socketPath),
		CreatedAt:   time.Now(),
	}

	// 2. Wait for API socket to become responsive (up to 3 seconds)
	if err := vm.waitForSocket(3 * time.Second); err != nil {
		vm.Stop()
		extra := stderrBuf.String()
		if extra == "" && consolePath != "" {
			data, _ := os.ReadFile(consolePath)
			extra = string(data)
		}
		if extra != "" {
			return nil, fmt.Errorf("firecracker api socket did not become ready: %w (stderr: %s)", err, extra)
		}
		return nil, fmt.Errorf("firecracker api socket did not become ready: %w", err)
	}

	// 3. Configure Machine Resources (1 vCPU, 128MB RAM)
	if err := vm.Client.SetMachineConfig(ctx, cfg.VCPUCount, cfg.MemSizeMiB); err != nil {
		vm.Stop()
		return nil, fmt.Errorf("failed to set machine config: %w", err)
	}

	// 4. Configure Boot Source (Kernel & init)
	// Supports both ttyS0 (x86) and ttyAMA0 (ARM64)
	bootArgs := "console=ttyS0 console=ttyAMA0 reboot=k panic=1 pci=off init=/init"
	if err := vm.Client.SetBootSource(ctx, cfg.KernelPath, bootArgs); err != nil {
		vm.Stop()
		return nil, fmt.Errorf("failed to set boot source (%s): %w", cfg.KernelPath, err)
	}

	// 5. Configure Rootfs Drive (Read-Only)
	if err := vm.Client.SetRootDrive(ctx, cfg.RootfsPath); err != nil {
		vm.Stop()
		return nil, fmt.Errorf("failed to set root drive (%s): %w", cfg.RootfsPath, err)
	}

	// 6. Configure Virtio-Vsock Device (Guest CID: 3)
	if err := vm.Client.SetVsock(ctx, 3, vsockPath); err != nil {
		vm.Stop()
		return nil, fmt.Errorf("failed to set vsock: %w", err)
	}

	// 7. Boot the microVM
	if err := vm.Client.StartInstance(ctx); err != nil {
		vm.Stop()
		extra := stderrBuf.String()
		if extra == "" && consolePath != "" {
			data, _ := os.ReadFile(consolePath)
			extra = string(data)
		}
		if extra != "" {
			return nil, fmt.Errorf("failed to start microvm instance: %w (stderr: %s)", err, extra)
		}
		return nil, fmt.Errorf("failed to start microvm instance: %w", err)
	}

	return vm, nil
}

// Stop terminates the Firecracker process and cleans up Unix domain socket files.
func (vm *MicroVM) Stop() {
	if vm.Cmd != nil && vm.Cmd.Process != nil {
		_ = vm.Cmd.Process.Kill()
		_ = vm.Cmd.Wait()
	}
	if vm.ConsoleFile != nil {
		_ = vm.ConsoleFile.Close()
	}
	_ = os.Remove(vm.SocketPath)
	_ = os.Remove(vm.VsockPath)
	if vm.ConsolePath != "" {
		_ = os.Remove(vm.ConsolePath)
	}
}

func (vm *MicroVM) waitForSocket(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("unix", vm.SocketPath, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("socket %s timed out after %v", vm.SocketPath, timeout)
}
