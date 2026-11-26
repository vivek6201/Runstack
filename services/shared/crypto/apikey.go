package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateAPIKey generates a Stripe-style API key:
// Format: cr_{env}_{random32hex}
// Returns:
//   - rawKey: "cr_live_8f3a..." (shown ONCE to the user)
//   - prefix: "cr_live_8f3a"   (stored in DB for display)
//   - hash:   SHA-256 string    (stored in DB and Redis)
func GenerateAPIKey(env string) (rawKey string, prefix string, hash string, error error) {
	if env == "" {
		env = "live"
	}

	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	randomPart := hex.EncodeToString(bytes)
	rawKey = fmt.Sprintf("cr_%s_%s", env, randomPart)
	prefix = rawKey[:min(len(rawKey), 12)]
	hash = HashAPIKey(rawKey)
	return rawKey, prefix, hash, nil
}

// HashAPIKey returns the lowercase hex-encoded SHA-256 hash of the given raw API key.
func HashAPIKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

// GenerateSessionToken creates a cryptographically random opaque session token
// and its SHA-256 hash (stored in PostgreSQL and Redis).
func GenerateSessionToken() (rawToken string, tokenHash string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("failed to generate session bytes: %w", err)
	}
	rawToken = "cr_sess_" + hex.EncodeToString(bytes)
	tokenHash = HashAPIKey(rawToken)
	return rawToken, tokenHash, nil
}
