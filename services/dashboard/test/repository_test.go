package test

import (
	"testing"

	"code-runner/dashboard/internal/auth"
	"code-runner/dashboard/internal/keys"
	"code-runner/dashboard/internal/profile"
	"code-runner/dashboard/internal/submissions"
)

func TestRepositories_Instantiation(t *testing.T) {
	aRepo := auth.NewAuthRepository(nil)
	if aRepo == nil {
		t.Fatal("expected auth repository instance")
	}

	pRepo := profile.NewProfileRepository(nil)
	if pRepo == nil {
		t.Fatal("expected profile repository instance")
	}

	kRepo := keys.NewKeysRepository(nil)
	if kRepo == nil {
		t.Fatal("expected keys repository instance")
	}

	sRepo := submissions.NewSubmissionsRepository(nil)
	if sRepo == nil {
		t.Fatal("expected submissions repository instance")
	}

	var _ auth.IAuthRepository = aRepo
	var _ profile.IProfileRepository = pRepo
	var _ keys.IKeysRepository = kRepo
	var _ submissions.ISubmissionsRepository = sRepo
}
