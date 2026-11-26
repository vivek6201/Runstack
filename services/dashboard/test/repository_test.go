package test

import (
	"testing"

	"code-runner/dashboard/internal/auth"
	"code-runner/dashboard/internal/keys"
	"code-runner/dashboard/internal/submissions"
)

func TestRepositories_Instantiation(t *testing.T) {
	aRepo := auth.NewAuthRepository(nil)
	if aRepo == nil {
		t.Fatal("expected auth repository instance")
	}

	kRepo := keys.NewKeysRepository(nil)
	if kRepo == nil {
		t.Fatal("expected keys repository instance")
	}

	sRepo := submissions.NewSubmissionsRepository(nil)
	if sRepo == nil {
		t.Fatal("expected submissions repository instance")
	}

	// Verify they satisfy their respective interfaces
	var _ auth.IAuthRepository = aRepo
	var _ keys.IKeysRepository = kRepo
	var _ submissions.ISubmissionsRepository = sRepo
}
