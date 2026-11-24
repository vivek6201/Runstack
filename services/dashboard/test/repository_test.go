package test

import (
	"testing"

	"code-runner/dashboard/internal/repository/apikey"
	"code-runner/dashboard/internal/repository/submission"
	"code-runner/dashboard/internal/repository/user"
)

func TestRepositories_Instantiation(t *testing.T) {
	uRepo := user.NewRepository(nil)
	if uRepo == nil {
		t.Fatal("expected user repository instance")
	}

	kRepo := apikey.NewRepository(nil)
	if kRepo == nil {
		t.Fatal("expected apikey repository instance")
	}

	sRepo := submission.NewRepository(nil)
	if sRepo == nil {
		t.Fatal("expected submission repository instance")
	}

	// Verify they satisfy their respective interfaces
	var _ user.Repository = uRepo
	var _ apikey.Repository = kRepo
	var _ submission.Repository = sRepo
}
