package auth

import (
	"encoding/base64"
	"testing"
)

func TestGenerateSessionTokenReturnsPlainAndHash(t *testing.T) {
	plain, hash, err := generateSessionToken()
	if err != nil {
		t.Fatalf("generateSessionToken returned error: %v", err)
	}
	if plain == "" {
		t.Fatal("plain token is empty")
	}
	if hash == "" {
		t.Fatal("hash is empty")
	}
	if plain == hash {
		t.Fatal("plain token and hash should differ")
	}
	if hashSessionToken(plain) != hash {
		t.Fatal("hash does not match plain token")
	}
	if _, err := base64.RawURLEncoding.DecodeString(plain); err != nil {
		t.Fatalf("plain token is not raw URL base64: %v", err)
	}
	if _, err := base64.RawURLEncoding.DecodeString(hash); err != nil {
		t.Fatalf("hash is not raw URL base64: %v", err)
	}
}

func TestGenerateSessionTokenProducesUniqueTokens(t *testing.T) {
	firstPlain, firstHash, err := generateSessionToken()
	if err != nil {
		t.Fatalf("generateSessionToken returned error: %v", err)
	}
	secondPlain, secondHash, err := generateSessionToken()
	if err != nil {
		t.Fatalf("generateSessionToken returned error: %v", err)
	}

	if firstPlain == secondPlain {
		t.Fatal("generated duplicate plain tokens")
	}
	if firstHash == secondHash {
		t.Fatal("generated duplicate token hashes")
	}
}

func TestHashSessionTokenIsDeterministic(t *testing.T) {
	first := hashSessionToken("session-token")
	second := hashSessionToken("session-token")

	if first != second {
		t.Fatalf("hashSessionToken is not deterministic: %q != %q", first, second)
	}
	if first == hashSessionToken("other-session-token") {
		t.Fatal("different tokens produced the same hash")
	}
}

func TestGeneratePasswordResetCodeReturnsSixDigits(t *testing.T) {
	code, err := generatePasswordResetCode()
	if err != nil {
		t.Fatalf("generatePasswordResetCode returned error: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("code length = %d, want 6", len(code))
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			t.Fatalf("code contains non-digit rune %q", r)
		}
	}
}
