package keys

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	apiKeyRepo "code-runner/dashboard/internal/repository/apikey"
	"code-runner/shared/cache"
	"code-runner/shared/crypto"
	"code-runner/shared/dto"
	"code-runner/shared/models"
)

var (
	ErrKeyNotFound = errors.New("api key not found or already revoked")
)

type Service struct {
	keyRepo apiKeyRepo.Repository
	cache   *cache.Cache
}

func NewService(keyRepo apiKeyRepo.Repository, cache *cache.Cache) *Service {
	return &Service{
		keyRepo: keyRepo,
		cache:   cache,
	}
}

func (s *Service) CreateKey(ctx context.Context, userID uuid.UUID, plan models.PlanTier, req dto.CreateAPIKeyRequest) (*dto.CreateAPIKeyResponse, error) {
	name := req.Name
	if name == "" {
		name = "Default API Key"
	}

	// 1. Determine default & maximum allowances by plan tier to prevent privilege escalation
	var defaultRPM, maxRPM int
	var defaultConcurrency, maxConcurrency int

	switch plan {
	case models.PlanEnterprise:
		defaultRPM, maxRPM = 1200, 2400
		defaultConcurrency, maxConcurrency = 50, 100
	case models.PlanPro:
		defaultRPM, maxRPM = 300, 600
		defaultConcurrency, maxConcurrency = 10, 20
	default: // Free
		defaultRPM, maxRPM = 60, 60
		defaultConcurrency, maxConcurrency = 2, 2
	}

	rpm := req.RateLimitRPM
	if rpm <= 0 || rpm > maxRPM {
		rpm = defaultRPM
	}

	concurrency := req.MaxConcurrency
	if concurrency <= 0 || concurrency > maxConcurrency {
		concurrency = defaultConcurrency
	}

	rawKey, prefix, hash, err := crypto.GenerateAPIKey("live")
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	apiKey := models.APIKey{
		ID:             uuid.New(),
		UserID:         userID,
		KeyHash:        hash,
		KeyPrefix:      prefix,
		Name:           name,
		IsActive:       true,
		RateLimitRPM:   rpm,
		MaxConcurrency: concurrency,
	}

	if err := s.keyRepo.Create(ctx, &apiKey); err != nil {
		return nil, fmt.Errorf("failed to save api key: %w", err)
	}

	// Cache in Redis for sub-millisecond execution engine lookup.
	// We store without TTL (0) so keys do not expire unexpectedly after 24h.
	// Key invalidation is explicitly handled via s.cache.Revoke on key revocation.
	if s.cache != nil {
		metadata := models.APIKeyCachedMetadata{
			APIKeyID:       apiKey.ID,
			UserID:         userID,
			Plan:           plan,
			RateLimitRPM:   apiKey.RateLimitRPM,
			MaxConcurrency: apiKey.MaxConcurrency,
			IsActive:       true,
		}
		_ = s.cache.Set(ctx, cache.Key("apikey", "cache", hash), metadata, 0)
		_ = s.cache.Revoke(ctx, cache.Key("dashboard", "keys", userID.String()))
	}

	return &dto.CreateAPIKeyResponse{
		ID:             apiKey.ID,
		Name:           apiKey.Name,
		KeyPrefix:      apiKey.KeyPrefix,
		RawKey:         rawKey,
		RateLimitRPM:   apiKey.RateLimitRPM,
		MaxConcurrency: apiKey.MaxConcurrency,
		CreatedAt:      apiKey.CreatedAt,
	}, nil
}

// ListKeys wraps the DB lookup with cache.Fetch: checks Redis first, queries DB on miss.
func (s *Service) ListKeys(ctx context.Context, userID uuid.UUID) ([]dto.APIKeyItemResponse, error) {
	cacheKey := cache.Key("dashboard", "keys", userID.String())

	return cache.Fetch(ctx, s.cache, cacheKey, 5*time.Minute, func() ([]dto.APIKeyItemResponse, error) {
		keys, err := s.keyRepo.FindByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}

		res := make([]dto.APIKeyItemResponse, len(keys))
		for i, k := range keys {
			res[i] = dto.APIKeyItemResponse{
				ID:             k.ID,
				Name:           k.Name,
				KeyPrefix:      k.KeyPrefix,
				IsActive:       k.IsActive,
				RateLimitRPM:   k.RateLimitRPM,
				MaxConcurrency: k.MaxConcurrency,
				CreatedAt:      k.CreatedAt,
				RevokedAt:      k.RevokedAt,
			}
		}
		return res, nil
	})
}

func (s *Service) RevokeKey(ctx context.Context, userID, keyID uuid.UUID) error {
	key, err := s.keyRepo.FindByID(ctx, userID, keyID)
	if err != nil {
		if errors.Is(err, apiKeyRepo.ErrKeyNotFound) {
			return ErrKeyNotFound
		}
		return err
	}

	if err := s.keyRepo.Revoke(ctx, userID, keyID); err != nil {
		if errors.Is(err, apiKeyRepo.ErrKeyNotFound) {
			return ErrKeyNotFound
		}
		return err
	}

	// Invalidate API key metadata and keys list cache
	if s.cache != nil {
		_ = s.cache.Revoke(ctx,
			cache.Key("apikey", "cache", key.KeyHash),
			cache.Key("dashboard", "keys", userID.String()),
		)
	}

	return nil
}
