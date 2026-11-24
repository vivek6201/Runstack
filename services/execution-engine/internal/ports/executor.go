package ports

import (
	"context"

	"code-runner/shared/dto"
)

// Executor defines the contract for executing untrusted code.
// Concrete adapters include LocalSubprocessAdapter (for macOS dev)
// and FirecrackerAdapter (for Linux KVM production).
type Executor interface {
	Name() string
	Execute(ctx context.Context, job *dto.Job) (*dto.ExecutionResult, error)
}
