package models

import (
	"time"

	"github.com/google/uuid"
)

// Profile stores developer profile and editor preferences for a User (1:1).
type Profile struct {
	ID              uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	Username        string    `gorm:"type:varchar(32);uniqueIndex;not null" json:"username"`
	DisplayName     string    `gorm:"type:varchar(100)" json:"display_name"`
	Bio             string    `gorm:"type:text" json:"bio"`
	AvatarURL       string    `gorm:"type:text" json:"avatar_url"`
	DefaultLanguage string    `gorm:"type:varchar(32);default:'python';not null" json:"default_language"`
	EditorTheme     string    `gorm:"type:varchar(32);default:'dark';not null" json:"editor_theme"`
	Company         string    `gorm:"type:varchar(100)" json:"company"`
	GitHubURL       string    `gorm:"column:github_url;type:varchar(255)" json:"github_url"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
