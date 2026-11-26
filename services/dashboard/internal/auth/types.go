package auth

import (
	"time"

	"github.com/google/uuid"

	"code-runner/shared/models"
)

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email,max=254"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UserResponse struct {
	ID        uuid.UUID       `json:"id"`
	Email     string          `json:"email"`
	Role      models.UserRole `json:"role"`
	Plan      models.PlanTier `json:"plan"`
	CreatedAt time.Time       `json:"created_at"`
}

type AuthResponse struct {
	User      UserResponse `json:"user"`
	Token     string       `json:"token,omitempty"`
	ExpiresAt time.Time    `json:"expires_at"`
}
