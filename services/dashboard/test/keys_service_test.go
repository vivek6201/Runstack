package test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"code-runner/dashboard/internal/keys"
	"code-runner/shared/models"
)

type mockAPIKeyRepo struct {
	createdKey *models.APIKey
}

func (m *mockAPIKeyRepo) Create(ctx context.Context, key *models.APIKey) error {
	m.createdKey = key
	return nil
}

func (m *mockAPIKeyRepo) FindByID(ctx context.Context, userID, keyID uuid.UUID) (*models.APIKey, error) {
	return nil, nil
}

func (m *mockAPIKeyRepo) FindByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	return nil, nil
}

func (m *mockAPIKeyRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]models.APIKey, error) {
	return nil, nil
}

func (m *mockAPIKeyRepo) Revoke(ctx context.Context, userID, keyID uuid.UUID) error {
	return nil
}

func TestKeysService_PlanTierLimitsEnforced(t *testing.T) {
	mockRepo := &mockAPIKeyRepo{}
	svc := keys.NewKeysService(mockRepo, nil)

	userID := uuid.New()

	// 1. Free tier user attempts to request 1,000,000 RPM and 500 concurrency
	resp, err := svc.CreateKey(context.Background(), userID, models.PlanFree, keys.CreateAPIKeyRequest{
		Name:           "Free User Exploiting Limits",
		RateLimitRPM:   1000000,
		MaxConcurrency: 500,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must be capped to Free tier limit (60 RPM, 2 concurrency)
	if resp.RateLimitRPM > 60 {
		t.Errorf("SECURITY VULNERABILITY: RateLimitRPM was not capped for Free plan: got %d", resp.RateLimitRPM)
	}
	if resp.MaxConcurrency > 2 {
		t.Errorf("SECURITY VULNERABILITY: MaxConcurrency was not capped for Free plan: got %d", resp.MaxConcurrency)
	}

	// 2. Pro tier user requesting within limits (e.g. 500 RPM, 15 concurrency)
	respPro, err := svc.CreateKey(context.Background(), userID, models.PlanPro, keys.CreateAPIKeyRequest{
		Name:           "Pro User Custom Limits",
		RateLimitRPM:   500,
		MaxConcurrency: 15,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respPro.RateLimitRPM != 500 {
		t.Errorf("expected 500 RPM for Pro tier, got %d", respPro.RateLimitRPM)
	}
	if respPro.MaxConcurrency != 15 {
		t.Errorf("expected 15 concurrency for Pro tier, got %d", respPro.MaxConcurrency)
	}
}
