package keys

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"code-runner/shared/cache"
	"code-runner/shared/crypto"
	"code-runner/shared/models"
)

type IKeysService interface {
	CreateKey(ctx context.Context, userID uuid.UUID, plan models.PlanTier, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error)
	ListKeys(ctx context.Context, userID uuid.UUID) ([]APIKeyItemResponse, error)
	RevokeKey(ctx context.Context, userID, keyID uuid.UUID) error
}

type KeysService struct {
	repo   IKeysRepository
	caches *Caches
}

func NewKeysService(repo IKeysRepository, caches *Caches) IKeysService {
	return &KeysService{
		repo:   repo,
		caches: caches,
	}
}

func (s *KeysService) CreateKey(ctx context.Context, userID uuid.UUID, plan models.PlanTier, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	name := req.Name
	if name == "" {
		name = "Default API Key"
	}

	rpm, concurrency := resolveKeyQuotas(plan, req.RateLimitRPM, req.MaxConcurrency)

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

	if err := s.repo.Create(ctx, &apiKey); err != nil {
		return nil, fmt.Errorf("failed to save api key: %w", err)
	}

	s.cacheNewKey(ctx, &apiKey, userID, plan)

	return &CreateAPIKeyResponse{
		ID:             apiKey.ID,
		Name:           apiKey.Name,
		KeyPrefix:      apiKey.KeyPrefix,
		RawKey:         rawKey,
		RateLimitRPM:   apiKey.RateLimitRPM,
		MaxConcurrency: apiKey.MaxConcurrency,
		CreatedAt:      apiKey.CreatedAt,
	}, nil
}

func (s *KeysService) ListKeys(ctx context.Context, userID uuid.UUID) ([]APIKeyItemResponse, error) {
	load := func() (*[]APIKeyItemResponse, error) {
		list, err := s.repo.FindByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		items := mapKeyItems(list)
		return &items, nil
	}

	if s.caches == nil || s.caches.Keys == nil {
		res, err := load()
		if err != nil {
			return nil, err
		}
		return *res, nil
	}

	cacheKey := cache.Key("dashboard", "keys", userID.String())
	res, err := s.caches.Keys.Fetch(ctx, cacheKey, 5*time.Minute, load)
	if err != nil {
		return nil, err
	}
	return *res, nil
}

func (s *KeysService) RevokeKey(ctx context.Context, userID, keyID uuid.UUID) error {
	key, err := s.repo.FindByID(ctx, userID, keyID)
	if err != nil {
		return err
	}

	if err := s.repo.Revoke(ctx, userID, keyID); err != nil {
		return err
	}

	s.invalidateRevokedKey(ctx, key.KeyHash, userID)
	return nil
}

func (s *KeysService) cacheNewKey(ctx context.Context, apiKey *models.APIKey, userID uuid.UUID, plan models.PlanTier) {
	if s.caches == nil {
		return
	}
	if s.caches.KeyMetadata != nil {
		metadata := models.APIKeyCachedMetadata{
			APIKeyID:       apiKey.ID,
			UserID:         userID,
			Plan:           plan,
			RateLimitRPM:   apiKey.RateLimitRPM,
			MaxConcurrency: apiKey.MaxConcurrency,
			IsActive:       true,
		}
		_ = s.caches.KeyMetadata.Set(ctx, cache.Key("apikey", "cache", apiKey.KeyHash), &metadata, 0)
	}
	if s.caches.Keys != nil {
		s.caches.Keys.InvalidateAsync(cache.Key("dashboard", "keys", userID.String()))
	}
}

func (s *KeysService) invalidateRevokedKey(ctx context.Context, keyHash string, userID uuid.UUID) {
	if s.caches == nil {
		return
	}
	if s.caches.KeyMetadata != nil {
		_ = s.caches.KeyMetadata.Delete(ctx, cache.Key("apikey", "cache", keyHash))
	}
	if s.caches.Keys != nil {
		s.caches.Keys.InvalidateAsync(cache.Key("dashboard", "keys", userID.String()))
	}
}

func resolveKeyQuotas(plan models.PlanTier, reqRPM, reqConcurrency int) (int, int) {
	defaultRPM, maxRPM, defaultConc, maxConc := planLimits(plan)
	rpm := reqRPM
	if rpm <= 0 || rpm > maxRPM {
		rpm = defaultRPM
	}
	concurrency := reqConcurrency
	if concurrency <= 0 || concurrency > maxConc {
		concurrency = defaultConc
	}
	return rpm, concurrency
}

func mapKeyItems(list []models.APIKey) []APIKeyItemResponse {
	res := make([]APIKeyItemResponse, len(list))
	for i, k := range list {
		res[i] = APIKeyItemResponse{
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
	return res
}

func planLimits(plan models.PlanTier) (defaultRPM, maxRPM, defaultConc, maxConc int) {
	switch plan {
	case models.PlanEnterprise:
		return 1200, 2400, 50, 100
	case models.PlanPro:
		return 300, 600, 10, 20
	default:
		return 60, 60, 2, 2
	}
}
