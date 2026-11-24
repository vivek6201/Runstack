package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateAPIKeyRequest holds parameters for creating a new machine API key.
type CreateAPIKeyRequest struct {
	Name           string `json:"name"`
	RateLimitRPM   int    `json:"rate_limit_rpm,omitempty"`
	MaxConcurrency int    `json:"max_concurrency,omitempty"`
}

// CreateAPIKeyResponse includes the raw plaintext key shown ONCE to the user.
type CreateAPIKeyResponse struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	KeyPrefix      string    `json:"key_prefix"`
	RawKey         string    `json:"raw_key"` // Shown only once upon creation
	RateLimitRPM   int       `json:"rate_limit_rpm"`
	MaxConcurrency int       `json:"max_concurrency"`
	CreatedAt      time.Time `json:"created_at"`
}

// APIKeyItemResponse represents safe key details listed in the dashboard.
type APIKeyItemResponse struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	KeyPrefix      string     `json:"key_prefix"` // e.g. "cr_live_8f3a..."
	IsActive       bool       `json:"is_active"`
	RateLimitRPM   int        `json:"rate_limit_rpm"`
	MaxConcurrency int        `json:"max_concurrency"`
	CreatedAt      time.Time  `json:"created_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}
