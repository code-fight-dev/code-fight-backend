package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func generateSessionToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}

	plain := base64.RawURLEncoding.EncodeToString(raw)
	return plain, hashSessionToken(plain), nil
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
