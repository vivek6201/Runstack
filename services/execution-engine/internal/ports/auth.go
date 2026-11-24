package ports

import (
	"context"

	"code-runner/shared/models"
)

// KeyValidator validates an incoming API key hash and returns cached authorization metadata.
type KeyValidator interface {
	ValidateKey(ctx context.Context, keyHash string) (*models.APIKeyCachedMetadata, error)
}
