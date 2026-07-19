package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateClientAPIKey generates a cryptographically secure API key
func GenerateClientAPIKey() (string, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	randomString := hex.EncodeToString(randomBytes)
	return "sk_live_" + randomString, nil
}

// HashAPIKey creates a SHA-256 hash of the API key
func HashAPIKey(apikey string) string {
	hash := sha256.Sum256([]byte(apikey))
	return hex.EncodeToString(hash[:])
}
