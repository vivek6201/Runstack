package dto

import (
	"time"

	"github.com/google/uuid"

	"code-runner/shared/models"
)

// RegisterRequest holds registration data.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest holds login credentials.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserResponse represents safe user details sent to clients.
type UserResponse struct {
	ID        uuid.UUID       `json:"id"`
	Email     string          `json:"email"`
	Role      models.UserRole `json:"role"`
	Plan      models.PlanTier `json:"plan"`
	CreatedAt time.Time       `json:"created_at"`
}

// AuthResponse returns session token and profile data after login/registration.
type AuthResponse struct {
	User      UserResponse `json:"user"`
	Token     string       `json:"token,omitempty"`
	ExpiresAt time.Time    `json:"expires_at"`
}
