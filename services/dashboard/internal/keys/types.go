package keys

import (
	"time"

	"github.com/google/uuid"
)

type CreateAPIKeyRequest struct {
	Name           string `json:"name" validate:"omitempty,max=100"`
	RateLimitRPM   int    `json:"rate_limit_rpm,omitempty"`
	MaxConcurrency int    `json:"max_concurrency,omitempty"`
}

type CreateAPIKeyResponse struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	KeyPrefix      string    `json:"key_prefix"`
	RawKey         string    `json:"raw_key"`
	RateLimitRPM   int       `json:"rate_limit_rpm"`
	MaxConcurrency int       `json:"max_concurrency"`
	CreatedAt      time.Time `json:"created_at"`
}

type APIKeyItemResponse struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	KeyPrefix      string     `json:"key_prefix"`
	IsActive       bool       `json:"is_active"`
	RateLimitRPM   int        `json:"rate_limit_rpm"`
	MaxConcurrency int        `json:"max_concurrency"`
	CreatedAt      time.Time  `json:"created_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}
