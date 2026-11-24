package models

import (
	"time"

	"github.com/google/uuid"
)

// APIKey represents a machine-to-machine authentication credential.
type APIKey struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID        uuid.UUID  `gorm:"type:uuid;index;not null" json:"user_id"`
	KeyHash       string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
	KeyPrefix     string     `gorm:"type:varchar(16);not null" json:"key_prefix"` // e.g. "cr_live_8f3a"
	Name          string     `gorm:"type:varchar(100);not null" json:"name"`       // e.g. "Production Key"
	IsActive      bool       `gorm:"default:true;not null;index" json:"is_active"`
	RateLimitRPM  int        `gorm:"default:60;not null" json:"rate_limit_rpm"`
	MaxConcurrency int       `gorm:"default:2;not null" json:"max_concurrency"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
}

// APIKeyCachedMetadata is the payload stored in Redis for sub-millisecond lookup.
type APIKeyCachedMetadata struct {
	APIKeyID       uuid.UUID `json:"api_key_id"`
	UserID         uuid.UUID `json:"user_id"`
	Plan           PlanTier  `json:"plan"`
	RateLimitRPM   int       `json:"rate_limit_rpm"`
	MaxConcurrency int       `json:"max_concurrency"`
	IsActive       bool      `json:"is_active"`
}
