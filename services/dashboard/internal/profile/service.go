package profile

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"code-runner/shared/cache"
	"code-runner/shared/models"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,32}$`)

type IProfileService interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*ProfileResponse, error)
	UpdateByUserID(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*ProfileResponse, error)
}

type ProfileService struct {
	repo   IProfileRepository
	caches *Caches
}

func NewProfileService(repo IProfileRepository, caches *Caches) IProfileService {
	return &ProfileService{
		repo:   repo,
		caches: caches,
	}
}

func (s *ProfileService) GetByUserID(ctx context.Context, userID uuid.UUID) (*ProfileResponse, error) {
	load := func() (*ProfileResponse, error) {
		prof, err := s.repo.FindByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		return toProfileResponse(prof), nil
	}

	if s.caches == nil || s.caches.ByUserID == nil {
		return load()
	}

	key := cache.Key("dashboard", "profile", userID.String())
	return s.caches.ByUserID.Fetch(ctx, key, 10*time.Minute, load)
}

func (s *ProfileService) UpdateByUserID(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*ProfileResponse, error) {
	prof, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.Username != nil {
		username := strings.ToLower(strings.TrimSpace(*req.Username))
		if !usernamePattern.MatchString(username) {
			return nil, errors.New("username must be 3-32 characters using letters, numbers, '.', '_', or '-'")
		}
		if username != prof.Username {
			if existing, err := s.repo.FindByUsername(ctx, username); err == nil && existing.UserID != userID {
				return nil, ErrUsernameTaken
			}
			prof.Username = username
		}
	}

	applyProfilePatch(prof, req)

	if err := s.repo.Update(ctx, prof); err != nil {
		return nil, err
	}

	if s.caches != nil && s.caches.ByUserID != nil {
		s.caches.ByUserID.InvalidateAsync(cache.Key("dashboard", "profile", userID.String()))
	}

	return toProfileResponse(prof), nil
}

func applyProfilePatch(prof *models.Profile, req UpdateProfileRequest) {
	if req.DisplayName != nil {
		prof.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.Bio != nil {
		prof.Bio = strings.TrimSpace(*req.Bio)
	}
	if req.AvatarURL != nil {
		prof.AvatarURL = strings.TrimSpace(*req.AvatarURL)
	}
	if req.DefaultLanguage != nil {
		prof.DefaultLanguage = strings.TrimSpace(*req.DefaultLanguage)
	}
	if req.EditorTheme != nil {
		prof.EditorTheme = strings.TrimSpace(*req.EditorTheme)
	}
	if req.Company != nil {
		prof.Company = strings.TrimSpace(*req.Company)
	}
	if req.GitHubURL != nil {
		prof.GitHubURL = strings.TrimSpace(*req.GitHubURL)
	}
}

func toProfileResponse(prof *models.Profile) *ProfileResponse {
	return &ProfileResponse{
		ID:              prof.ID,
		UserID:          prof.UserID,
		Username:        prof.Username,
		DisplayName:     prof.DisplayName,
		Bio:             prof.Bio,
		AvatarURL:       prof.AvatarURL,
		DefaultLanguage: prof.DefaultLanguage,
		EditorTheme:     prof.EditorTheme,
		Company:         prof.Company,
		GitHubURL:       prof.GitHubURL,
		CreatedAt:       prof.CreatedAt,
		UpdatedAt:       prof.UpdatedAt,
	}
}
