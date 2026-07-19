package unit

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tundesmac/rate-limiter-service/internal/auth"
)

func TestAPIKeyGeneration(t *testing.T) {
	t.Run("Generates correctly formatted key", func(t *testing.T) {
		key, err := auth.GenerateClientAPIKey()
		assert.NoError(t, err)

		// Key should start with prefix
		assert.True(t, strings.HasPrefix(key, "sk_live_"), "Key should start with 'sk_live_'")

		// Prefix (8) + 32 bytes hex encoded (64) = 72 characters total
		assert.Equal(t, 72, len(key), "Generated key should be exactly 72 characters long")
	})

	t.Run("Generates unique keys", func(t *testing.T) {
		key1, err1 := auth.GenerateClientAPIKey()
		assert.NoError(t, err1)

		key2, err2 := auth.GenerateClientAPIKey()
		assert.NoError(t, err2)

		assert.NotEqual(t, key1, key2, "Generated keys must be unique")
	})
}

func TestAPIKeyHashing(t *testing.T) {
	t.Run("Produces deterministic hashes", func(t *testing.T) {
		key := "sk_live_test1234567890"

		hash1 := auth.HashAPIKey(key)
		hash2 := auth.HashAPIKey(key)

		assert.Equal(t, hash1, hash2, "Hashing the same key must produce the exact same result")
	})

	t.Run("Produces different hashes for different keys", func(t *testing.T) {
		key1 := "sk_live_test1"
		key2 := "sk_live_test2"

		hash1 := auth.HashAPIKey(key1)
		hash2 := auth.HashAPIKey(key2)

		assert.NotEqual(t, hash1, hash2, "Different keys must produce different hashes")
	})

	t.Run("Produces correct SHA-256 hash", func(t *testing.T) {
		key := "sk_live_known_test_key"
		// Manually compute expected SHA-256 hash
		expectedHashBytes := sha256.Sum256([]byte(key))
		expectedHashStr := hex.EncodeToString(expectedHashBytes[:])

		actualHashStr := auth.HashAPIKey(key)

		assert.Equal(t, expectedHashStr, actualHashStr, "Hash must be a valid hex-encoded SHA-256 hash of the input")
		assert.Equal(t, 64, len(actualHashStr), "SHA-256 hex hash must be exactly 64 characters long")
	})
}
