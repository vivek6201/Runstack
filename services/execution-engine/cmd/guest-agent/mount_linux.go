//go:build linux

package main

import (
	"log"
	"os"

	"golang.org/x/sys/unix"
)

// mountFilesystems mounts /proc, /sys, /dev, /tmp, /run, /workspace when running as PID 1
func mountFilesystems() {
	// Set standard system environment paths
	_ = os.Setenv("PATH", "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin")
	_ = os.Setenv("TMPDIR", "/tmp")
	_ = os.Setenv("HOME", "/workspace")

	mounts := []struct {
		source string
		target string
		fstype string
		flags  uintptr
		perm   os.FileMode
	}{
		{"proc", "/proc", "proc", 0, 0755},
		{"sysfs", "/sys", "sysfs", 0, 0755},
		{"devtmpfs", "/dev", "devtmpfs", 0, 0755},
		{"devpts", "/dev/pts", "devpts", 0, 0755},
		{"tmpfs", "/dev/shm", "tmpfs", 0, 0777},
		{"tmpfs", "/tmp", "tmpfs", 0, 0777},
		{"tmpfs", "/run", "tmpfs", 0, 0755},
		{"tmpfs", workspaceDir, "tmpfs", 0, 0777},
	}

	for _, m := range mounts {
		_ = os.MkdirAll(m.target, m.perm)
		if err := unix.Mount(m.source, m.target, m.fstype, m.flags, ""); err != nil {
			log.Printf("[Guest-Agent] Mount %s on %s (%s): %v", m.source, m.target, m.fstype, err)
		}
	}
}
