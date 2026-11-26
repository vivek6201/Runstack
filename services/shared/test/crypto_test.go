package test

import (
	"strings"
	"testing"

	"code-runner/shared/crypto"
)

func TestGenerateAPIKey(t *testing.T) {
	rawKey, prefix, hash, err := crypto.GenerateAPIKey("live")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(rawKey, "cr_live_") {
		t.Errorf("expected key to start with 'cr_live_', got: %s", rawKey)
	}

	if !strings.HasPrefix(prefix, "cr_live_") {
		t.Errorf("expected prefix to start with 'cr_live_', got: %s", prefix)
	}

	if len(hash) != 64 {
		t.Errorf("expected 64-char SHA256 hex hash, got length %d", len(hash))
	}

	// Verify that hashing rawKey matches the returned hash
	computedHash := crypto.HashAPIKey(rawKey)
	if computedHash != hash {
		t.Errorf("computed hash %s != returned hash %s", computedHash, hash)
	}
}

func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	key1, _, hash1, _ := crypto.GenerateAPIKey("live")
	key2, _, hash2, _ := crypto.GenerateAPIKey("live")

	if key1 == key2 {
		t.Errorf("expected keys to be unique, got identical keys: %s", key1)
	}
	if hash1 == hash2 {
		t.Errorf("expected hashes to be unique, got identical hashes: %s", hash1)
	}
}

func TestGenerateSessionToken(t *testing.T) {
	raw1, hash1, err := crypto.GenerateSessionToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(raw1, "cr_sess_") || len(raw1) != len("cr_sess_")+64 {
		t.Errorf("unexpected raw session token format: %q", raw1)
	}
	if len(hash1) != 64 || crypto.HashAPIKey(raw1) != hash1 {
		t.Errorf("session token hash mismatch")
	}
	raw2, hash2, _ := crypto.GenerateSessionToken()
	if raw1 == raw2 || hash1 == hash2 {
		t.Errorf("expected unique session tokens")
	}
}
