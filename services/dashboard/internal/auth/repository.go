package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"code-runner/shared/models"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrSessionInvalid     = errors.New("invalid or expired session")
)

type IAuthRepository interface {
	CreateWithProfile(ctx context.Context, user *models.User, profile *models.Profile) error
	FindByIDWithProfile(ctx context.Context, id uuid.UUID) (*models.User, *models.Profile, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	CreateSingleSession(ctx context.Context, session *models.Session) (evictedIDs []string, err error)
	FindSessionByID(ctx context.Context, sessionHash string) (*models.Session, *models.User, error)
	DeleteSession(ctx context.Context, sessionHash string) error
	TouchSession(ctx context.Context, sessionHash string, now time.Time) error
}

type AuthRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) IAuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) CreateWithProfile(ctx context.Context, user *models.User, profile *models.Profile) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		return tx.Create(profile).Error
	})
}

func (r *AuthRepository) FindByIDWithProfile(ctx context.Context, id uuid.UUID) (*models.User, *models.Profile, error) {
	var u models.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrUserNotFound
		}
		return nil, nil, err
	}

	var p models.Profile
	err := r.db.WithContext(ctx).Where("user_id = ?", id).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &u, nil, nil
		}
		return &u, nil, err
	}
	return &u, &p, nil
}

func (r *AuthRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// CreateSingleSession atomically revokes any prior sessions for the user and stores the new session.
func (r *AuthRepository) CreateSingleSession(ctx context.Context, session *models.Session) ([]string, error) {
	var evictedIDs []string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Session{}).
			Where("user_id = ?", session.UserID).
			Pluck("id", &evictedIDs).Error; err != nil {
			return err
		}

		if len(evictedIDs) > 0 {
			if err := tx.Where("user_id = ?", session.UserID).Delete(&models.Session{}).Error; err != nil {
				return err
			}
		}

		return tx.Create(session).Error
	})
	return evictedIDs, err
}

func (r *AuthRepository) FindSessionByID(ctx context.Context, sessionHash string) (*models.Session, *models.User, error) {
	var sess models.Session
	if err := r.db.WithContext(ctx).Where("id = ?", sessionHash).First(&sess).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrSessionInvalid
		}
		return nil, nil, err
	}

	var u models.User
	if err := r.db.WithContext(ctx).Where("id = ?", sess.UserID).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrSessionInvalid
		}
		return nil, nil, err
	}

	return &sess, &u, nil
}

func (r *AuthRepository) DeleteSession(ctx context.Context, sessionHash string) error {
	return r.db.WithContext(ctx).Where("id = ?", sessionHash).Delete(&models.Session{}).Error
}

func (r *AuthRepository) TouchSession(ctx context.Context, sessionHash string, now time.Time) error {
	return r.db.WithContext(ctx).Model(&models.Session{}).
		Where("id = ?", sessionHash).
		Update("last_active_at", now).Error
}
