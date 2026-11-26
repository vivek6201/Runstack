package test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"code-runner/dashboard/internal/profile"
	"code-runner/shared/models"
)

func TestProfileService_UsernameValidationAndConflict(t *testing.T) {
	repo := newInMemoryAuthRepo()
	svc := profile.NewProfileService(repo, nil)

	user1 := uuid.New()
	user2 := uuid.New()
	_ = repo.CreateWithProfile(context.Background(),
		&models.User{ID: user1, Email: "u1@example.com", PasswordHash: "x"},
		&models.Profile{ID: uuid.New(), UserID: user1, Username: "alice", DefaultLanguage: "python", EditorTheme: "dark"},
	)
	_ = repo.CreateWithProfile(context.Background(),
		&models.User{ID: user2, Email: "u2@example.com", PasswordHash: "x"},
		&models.Profile{ID: uuid.New(), UserID: user2, Username: "bob", DefaultLanguage: "python", EditorTheme: "dark"},
	)

	// Taken username rejected
	taken := "ALICE"
	if _, err := svc.UpdateByUserID(context.Background(), user2, profile.UpdateProfileRequest{Username: &taken}); !errors.Is(err, profile.ErrUsernameTaken) {
		t.Errorf("expected ErrUsernameTaken, got %v", err)
	}

	// Invalid username characters rejected
	bad := "bad name!!"
	if _, err := svc.UpdateByUserID(context.Background(), user2, profile.UpdateProfileRequest{Username: &bad}); err == nil {
		t.Errorf("expected invalid username format error")
	}

	// Valid update succeeds
	good := "bob_dev"
	lang := "go"
	res, err := svc.UpdateByUserID(context.Background(), user2, profile.UpdateProfileRequest{
		Username:        &good,
		DefaultLanguage: &lang,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Username != "bob_dev" || res.DefaultLanguage != "go" {
		t.Errorf("unexpected updated profile: %+v", res)
	}
}
