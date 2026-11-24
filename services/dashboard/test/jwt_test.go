package test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"code-runner/dashboard/internal/auth"
	"code-runner/shared/models"
)

func TestJWT_GenerateAndParse(t *testing.T) {
	secret := "test-secret-key-1234567890123456"
	userID := uuid.New()
	user := &models.User{
		ID:    userID,
		Email: "dev@example.com",
		Role:  models.RoleMember,
		Plan:  models.PlanPro,
	}

	token, expiresAt, err := auth.GenerateToken(user, secret, 2*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token string")
	}
	if !expiresAt.After(time.Now()) {
		t.Fatalf("expected expiration in future, got %v", expiresAt)
	}

	claims, err := auth.ParseToken(token, secret)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected UserID %v, got %v", userID, claims.UserID)
	}
	if claims.Email != "dev@example.com" {
		t.Errorf("expected Email dev@example.com, got %v", claims.Email)
	}
	if claims.Plan != models.PlanPro {
		t.Errorf("expected Plan pro, got %v", claims.Plan)
	}

	// Test invalid signature
	_, err = auth.ParseToken(token, "wrong-secret-key")
	if err == nil {
		t.Error("expected error parsing token with wrong secret, got nil")
	}
}
