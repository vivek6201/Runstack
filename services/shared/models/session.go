package models

import (
	"time"

	"github.com/google/uuid"
)

// Session tracks the single active server-side login session and device metadata for a User.
// ID stores the SHA-256 hash of the opaque session token presented by the client.
type Session struct {
	ID            string    `gorm:"type:varchar(64);primaryKey" json:"-"`
	UserID        uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	Provider      string    `gorm:"type:varchar(20);default:'password';not null" json:"provider"`
	IPAddress     string    `gorm:"type:varchar(45)" json:"ip_address"`
	UserAgent     string    `gorm:"type:text" json:"user_agent"`
	DeviceName    string    `gorm:"type:varchar(120)" json:"device_name"`
	ExpiresAt     time.Time `gorm:"index;not null" json:"expires_at"`
	LastRotatedAt time.Time `gorm:"not null" json:"last_rotated_at"`
	LastActiveAt  time.Time `gorm:"not null" json:"last_active_at"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}
