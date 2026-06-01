package auth

import (
	"testing"
)

func TestHashPasswordAndComparePassword(t *testing.T) {
	hash, err := hashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("hashPassword returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("password hash is empty")
	}
	if hash == "correct horse battery staple" {
		t.Fatal("password hash should not equal raw password")
	}
	if err := comparePassword(hash, "correct horse battery staple"); err != nil {
		t.Fatalf("comparePassword returned error for valid password: %v", err)
	}
	if err := comparePassword(hash, "wrong password"); err == nil {
		t.Fatal("comparePassword returned nil error for wrong password")
	}
}

func TestHashPasswordUsesSalt(t *testing.T) {
	first, err := hashPassword("same-password")
	if err != nil {
		t.Fatalf("hashPassword returned error: %v", err)
	}
	second, err := hashPassword("same-password")
	if err != nil {
		t.Fatalf("hashPassword returned error: %v", err)
	}
	if first == second {
		t.Fatal("same password produced identical hashes")
	}
}
