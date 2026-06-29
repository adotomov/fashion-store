package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// Generate returns a new random opaque bearer token and its SHA-256 hash.
// Only the hash should ever be persisted.
func Generate() (token string, hash string) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token = hex.EncodeToString(b)
	return token, Hash(token)
}

// Hash returns the SHA-256 hash of a token, hex-encoded.
func Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
