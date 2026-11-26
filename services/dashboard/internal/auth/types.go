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

type SessionMetadata struct {
	Provider   string
	IPAddress  string
	UserAgent  string
	DeviceName string
}

type SessionInfoResponse struct {
	Provider      string    `json:"provider"`
	IPAddress     string    `json:"ip_address"`
	UserAgent     string    `json:"user_agent"`
	DeviceName    string    `json:"device_name"`
	ExpiresAt     time.Time `json:"expires_at"`
	LastRotatedAt time.Time `json:"last_rotated_at"`
	LastActiveAt  time.Time `json:"last_active_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type ProfileSummary struct {
	ID              uuid.UUID `json:"id"`
	Username        string    `json:"username"`
	DisplayName     string    `json:"display_name"`
	Bio             string    `json:"bio"`
	AvatarURL       string    `json:"avatar_url"`
	DefaultLanguage string    `json:"default_language"`
	EditorTheme     string    `json:"editor_theme"`
	Company         string    `json:"company"`
	GitHubURL       string    `json:"github_url"`
}

type UserResponse struct {
	ID        uuid.UUID       `json:"id"`
	Email     string          `json:"email"`
	Role      models.UserRole `json:"role"`
	Plan      models.PlanTier `json:"plan"`
	Profile   *ProfileSummary `json:"profile,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type AuthResponse struct {
	User      UserResponse         `json:"user"`
	Session   *SessionInfoResponse `json:"session,omitempty"`
	Token     string               `json:"token,omitempty"`
	ExpiresAt time.Time            `json:"expires_at"`
}

// Claims represents the authenticated caller identity attached to request context locals.
type Claims struct {
	SessionID string          `json:"-"`
	UserID    uuid.UUID       `json:"user_id"`
	Email     string          `json:"email"`
	Role      models.UserRole `json:"role"`
	Plan      models.PlanTier `json:"plan"`
}

// SessionContext is stored in Redis cache and attached to request locals after session lookup.
type SessionContext struct {
	SessionID     string          `json:"session_id"`
	UserID        uuid.UUID       `json:"user_id"`
	Email         string          `json:"email"`
	Role          models.UserRole `json:"role"`
	Plan          models.PlanTier `json:"plan"`
	Provider      string          `json:"provider"`
	IPAddress     string          `json:"ip_address"`
	UserAgent     string          `json:"user_agent"`
	DeviceName    string          `json:"device_name"`
	ExpiresAt     time.Time       `json:"expires_at"`
	LastRotatedAt time.Time       `json:"last_rotated_at"`
	LastActiveAt  time.Time       `json:"last_active_at"`
	CreatedAt     time.Time       `json:"created_at"`
}

func (s *SessionContext) ToClaims() *Claims {
	return &Claims{
		SessionID: s.SessionID,
		UserID:    s.UserID,
		Email:     s.Email,
		Role:      s.Role,
		Plan:      s.Plan,
	}
}

func (s *SessionContext) ToSessionInfo() *SessionInfoResponse {
	return &SessionInfoResponse{
		Provider:      s.Provider,
		IPAddress:     s.IPAddress,
		UserAgent:     s.UserAgent,
		DeviceName:    s.DeviceName,
		ExpiresAt:     s.ExpiresAt,
		LastRotatedAt: s.LastRotatedAt,
		LastActiveAt:  s.LastActiveAt,
		CreatedAt:     s.CreatedAt,
	}
}
