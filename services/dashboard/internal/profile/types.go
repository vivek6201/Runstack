package profile

import (
	"time"

	"github.com/google/uuid"
)

type ProfileResponse struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"user_id"`
	Username        string    `json:"username"`
	DisplayName     string    `json:"display_name"`
	Bio             string    `json:"bio"`
	AvatarURL       string    `json:"avatar_url"`
	DefaultLanguage string    `json:"default_language"`
	EditorTheme     string    `json:"editor_theme"`
	Company         string    `json:"company"`
	GitHubURL       string    `json:"github_url"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UpdateProfileRequest struct {
	Username        *string `json:"username" validate:"omitempty,min=3,max=32"`
	DisplayName     *string `json:"display_name" validate:"omitempty,max=100"`
	Bio             *string `json:"bio" validate:"omitempty,max=500"`
	AvatarURL       *string `json:"avatar_url" validate:"omitempty,url,max=500"`
	DefaultLanguage *string `json:"default_language" validate:"omitempty,oneof=python javascript typescript c cpp go java"`
	EditorTheme     *string `json:"editor_theme" validate:"omitempty,oneof=dark light"`
	Company         *string `json:"company" validate:"omitempty,max=100"`
	GitHubURL       *string `json:"github_url" validate:"omitempty,url,max=255"`
}
