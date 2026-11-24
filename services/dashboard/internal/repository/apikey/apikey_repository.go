package apikey

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"code-runner/shared/models"
)

var (
	ErrKeyNotFound = errors.New("api key not found or already revoked")
)

type Repository interface {
	Create(ctx context.Context, key *models.APIKey) error
	FindByID(ctx context.Context, userID, keyID uuid.UUID) (*models.APIKey, error)
	FindByHash(ctx context.Context, hash string) (*models.APIKey, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]models.APIKey, error)
	Revoke(ctx context.Context, userID, keyID uuid.UUID) error
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, key *models.APIKey) error {
	return r.db.WithContext(ctx).Create(key).Error
}

func (r *GormRepository) FindByID(ctx context.Context, userID, keyID uuid.UUID) (*models.APIKey, error) {
	var k models.APIKey
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", keyID, userID).First(&k).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}
	return &k, nil
}

func (r *GormRepository) FindByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	var k models.APIKey
	if err := r.db.WithContext(ctx).Where("key_hash = ? AND is_active = ?", hash, true).First(&k).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}
	return &k, nil
}

func (r *GormRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]models.APIKey, error) {
	var keys []models.APIKey
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&keys).Error
	return keys, err
}

func (r *GormRepository) Revoke(ctx context.Context, userID, keyID uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&models.APIKey{}).
		Where("id = ? AND user_id = ? AND is_active = ?", keyID, userID, true).
		Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrKeyNotFound
	}
	return nil
}
