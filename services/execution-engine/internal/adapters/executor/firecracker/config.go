package firecracker

import (
	"os"
	"path/filepath"

	"code-runner/shared/env"
)

// Config defines the machine specs, binaries, and disk paths for Firecracker microVMs.
type Config struct {
	BinaryPath string // Path to `firecracker` executable
	KernelPath string // Path to `vmlinux` uncompressed kernel
	RootfsPath string // Path to `rootfs.ext4` filesystem image
	SocketDir  string // Temporary directory for Unix sockets
	VCPUCount  int    // Number of vCPUs per microVM (default: 1)
	MemSizeMiB int    // Memory size in MiB per microVM (default: 128)
	PoolSize   int    // Number of warm, pre-booted microVMs to keep idle (default: 2)
}

// NewConfigFromEnv loads Firecracker settings from environment variables with sensible defaults.
func NewConfigFromEnv() *Config {
	pwd, _ := os.Getwd()
	defaultBuildDir := filepath.Join(pwd, "build", "firecracker")

	kernelPath := env.Get("FIRECRACKER_KERNEL_PATH", "")
	if kernelPath == "" {
		kernelPath = env.Get("FIRECRACKER_KERNEL", filepath.Join(defaultBuildDir, "vmlinux"))
	}

	rootfsPath := env.Get("FIRECRACKER_ROOTFS_PATH", "")
	if rootfsPath == "" {
		rootfsPath = env.Get("FIRECRACKER_ROOTFS", filepath.Join(defaultBuildDir, "rootfs.ext4"))
	}

	return &Config{
		BinaryPath: env.Get("FIRECRACKER_BIN", "/usr/local/bin/firecracker"),
		KernelPath: kernelPath,
		RootfsPath: rootfsPath,
		SocketDir:  env.Get("FIRECRACKER_SOCKET_DIR", os.TempDir()),
		VCPUCount:  env.GetInt("FIRECRACKER_VCPU", 1),
		MemSizeMiB: env.GetInt("FIRECRACKER_MEM_MIB", 128),
		PoolSize:   env.GetInt("FIRECRACKER_POOL_SIZE", 2),
	}
}
