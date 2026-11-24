package firecracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// Client talks to the Firecracker REST API over a Unix Domain Socket (UDS).
type Client struct {
	socketPath string
	httpClient *http.Client
}

func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", socketPath)
				},
			},
			Timeout: 5 * time.Second,
		},
	}
}

// SetBootSource configures the uncompressed Linux kernel and boot arguments.
func (c *Client) SetBootSource(ctx context.Context, kernelPath, bootArgs string) error {
	payload := map[string]interface{}{
		"kernel_image_path": kernelPath,
		"boot_args":         bootArgs,
	}
	return c.put(ctx, "/boot-source", payload)
}

// SetRootDrive configures the rootfs.ext4 disk as a read-only root device.
func (c *Client) SetRootDrive(ctx context.Context, rootfsPath string) error {
	payload := map[string]interface{}{
		"drive_id":       "rootfs",
		"path_on_host":   rootfsPath,
		"is_root_device": true,
		"is_read_only":   true,
	}
	return c.put(ctx, "/drives/rootfs", payload)
}

// SetMachineConfig configures vCPU count and memory allocation.
func (c *Client) SetMachineConfig(ctx context.Context, vcpuCount, memSizeMiB int) error {
	payload := map[string]interface{}{
		"vcpu_count":   vcpuCount,
		"mem_size_mib": memSizeMiB,
	}
	return c.put(ctx, "/machine-config", payload)
}

// SetVsock configures the virtio-vsock device connecting host to guest.
func (c *Client) SetVsock(ctx context.Context, guestCID int, hostSocketPath string) error {
	payload := map[string]interface{}{
		"guest_cid": guestCID,
		"uds_path":  hostSocketPath,
	}
	return c.put(ctx, "/vsock", payload)
}

// StartInstance triggers the boot sequence for the configured microVM.
func (c *Client) StartInstance(ctx context.Context) error {
	payload := map[string]interface{}{
		"action_type": "InstanceStart",
	}
	return c.put(ctx, "/actions", payload)
}

func (c *Client) put(ctx context.Context, path string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://localhost"+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("firecracker api %s returned status %d: %s", path, resp.StatusCode, string(respBody))
	}

	return nil
}
