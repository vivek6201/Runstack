package profile

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"code-runner/shared/models"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrUsernameTaken   = errors.New("username is already taken")
)

type IProfileRepository interface {
	FindByUserID(ctx context.Context, userID uuid.UUID) (*models.Profile, error)
	FindByUsername(ctx context.Context, username string) (*models.Profile, error)
	Update(ctx context.Context, prof *models.Profile) error
}

type ProfileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) IProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*models.Profile, error) {
	var prof models.Profile
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&prof).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}
	return &prof, nil
}

func (r *ProfileRepository) FindByUsername(ctx context.Context, username string) (*models.Profile, error) {
	var prof models.Profile
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&prof).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}
	return &prof, nil
}

func (r *ProfileRepository) Update(ctx context.Context, prof *models.Profile) error {
	return r.db.WithContext(ctx).Save(prof).Error
}
