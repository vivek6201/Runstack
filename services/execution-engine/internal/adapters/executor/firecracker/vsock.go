package firecracker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"code-runner/shared/dto"
)

const guestPort = 5000

// ExecuteOverVsock connects to the microVM's vsock Unix socket, performs the Firecracker
// vsock handshake ("CONNECT 5000\n"), sends the job, and parses the execution result.
func ExecuteOverVsock(ctx context.Context, vsockPath string, job *dto.Job) (*dto.ExecutionResult, error) {
	// 1. Connect to host vsock Unix domain socket and handshake (retrying while guest agent boots)
	conn, reader, err := dialAndHandshake(ctx, vsockPath, guestPort, 8*time.Second)
	if err != nil {
		return nil, fmt.Errorf("vsock handshake error: %w", err)
	}
	defer conn.Close()

	// 2. Send Job JSON to the Guest Agent
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(job); err != nil {
		return nil, fmt.Errorf("failed to stream job payload to guest: %w", err)
	}

	// 3. Read Execution Result JSON back from the Guest Agent
	var result dto.ExecutionResult
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode execution result from guest: %w", err)
	}

	return &result, nil
}

// dialAndHandshake connects to Firecracker's host UDS and sends "CONNECT <port>\n".
// If the guest kernel is still booting or /init is binding the vsock port, Firecracker may
// close the connection (EOF) or reject it. This function retries until port 5000 responds "OK".
func dialAndHandshake(ctx context.Context, path string, port int, timeout time.Duration) (net.Conn, *bufio.Reader, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}

		conn, err := net.DialTimeout("unix", path, 500*time.Millisecond)
		if err != nil {
			lastErr = err
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Short deadline for the handshake exchange
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

		handshake := fmt.Sprintf("CONNECT %d\n", port)
		if _, err := conn.Write([]byte(handshake)); err != nil {
			_ = conn.Close()
			lastErr = fmt.Errorf("failed to send handshake: %w", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		reader := bufio.NewReader(conn)
		responseLine, err := reader.ReadString('\n')
		if err != nil {
			_ = conn.Close()
			lastErr = fmt.Errorf("read handshake response: %w", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if !strings.HasPrefix(responseLine, "OK") {
			_ = conn.Close()
			lastErr = fmt.Errorf("port %d rejected: %s", port, strings.TrimSpace(responseLine))
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Success! Clear deadline for actual code execution
		_ = conn.SetDeadline(time.Time{})
		return conn, reader, nil
	}

	return nil, nil, fmt.Errorf("vsock connection to port %d on %s failed within %v (last error: %v)", port, path, timeout, lastErr)
}
