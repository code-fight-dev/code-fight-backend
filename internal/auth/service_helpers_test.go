package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeSignUpInput(t *testing.T) {
	username, email, password, err := normalizeSignUpInput(SignUpInput{
		Username: " Player_1 ",
		Email:    " USER@Example.COM ",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("normalizeSignUpInput returned error: %v", err)
	}
	if username != "Player_1" {
		t.Fatalf("username = %q", username)
	}
	if email != "user@example.com" {
		t.Fatalf("email = %q", email)
	}
	if password != "password123" {
		t.Fatalf("password = %q", password)
	}
}

func TestNormalizeSignUpInputRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		input SignUpInput
	}{
		{name: "short username", input: SignUpInput{Username: "ab", Email: "u@example.com", Password: "password123"}},
		{name: "bad username chars", input: SignUpInput{Username: "bad-name", Email: "u@example.com", Password: "password123"}},
		{name: "bad email", input: SignUpInput{Username: "player", Email: "bad", Password: "password123"}},
		{name: "short password", input: SignUpInput{Username: "player", Email: "u@example.com", Password: "short"}},
		{name: "long password", input: SignUpInput{Username: "player", Email: "u@example.com", Password: strings.Repeat("x", passwordMaxLength+1)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := normalizeSignUpInput(tt.input)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("normalizeSignUpInput error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestNormalizeSignInInput(t *testing.T) {
	email, password, err := normalizeSignInInput(SignInInput{
		Email:    " USER@Example.COM ",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("normalizeSignInInput returned error: %v", err)
	}
	if email != "user@example.com" || password != "password123" {
		t.Fatalf("email/password = %q/%q", email, password)
	}

	if _, _, err := normalizeSignInInput(SignInInput{Email: "user@example.com"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("empty password error = %v, want ErrInvalidInput", err)
	}
}

func TestNormalizePasswordResetConfirmInput(t *testing.T) {
	email, code, password, err := normalizePasswordResetConfirmInput(PasswordResetConfirmInput{
		Email:    " USER@Example.COM ",
		Code:     " 123456 ",
		Password: "new-password",
	})
	if err != nil {
		t.Fatalf("normalizePasswordResetConfirmInput returned error: %v", err)
	}
	if email != "user@example.com" || code != "123456" || password != "new-password" {
		t.Fatalf("normalized = %q/%q/%q", email, code, password)
	}

	tests := []PasswordResetConfirmInput{
		{Email: "bad", Code: "123456", Password: "new-password"},
		{Email: "user@example.com", Code: "12345", Password: "new-password"},
		{Email: "user@example.com", Code: "12345x", Password: "new-password"},
		{Email: "user@example.com", Code: "123456", Password: "short"},
	}
	for _, input := range tests {
		if _, _, _, err := normalizePasswordResetConfirmInput(input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("input %#v error = %v, want ErrInvalidInput", input, err)
		}
	}
}

func TestSanitizeUsernameCandidate(t *testing.T) {
	if got := sanitizeUsernameCandidate("  A.B-C_1!  "); got != "a_b_c_1" {
		t.Fatalf("sanitizeUsernameCandidate = %q", got)
	}
	if got := sanitizeUsernameCandidate("__bad__"); got != "bad" {
		t.Fatalf("sanitizeUsernameCandidate trimmed = %q", got)
	}
	if got := sanitizeUsernameCandidate(strings.Repeat("a", usernameMaxLength+10)); len(got) != usernameMaxLength {
		t.Fatalf("sanitized length = %d", len(got))
	}
}

func TestNormalizeEmailAndOAuthPresentationHelpers(t *testing.T) {
	email, err := normalizeEmail(" USER@Example.COM ")
	if err != nil {
		t.Fatalf("normalizeEmail returned error: %v", err)
	}
	if email != "user@example.com" {
		t.Fatalf("email = %q", email)
	}
	if _, err := normalizeEmail("not-an-email"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("normalizeEmail error = %v, want ErrInvalidInput", err)
	}

	if got := resolveAvatarSource(" https://avatar.example.com/u.png "); got != AvatarSourceProvider {
		t.Fatalf("avatar source = %q", got)
	}
	if got := resolveAvatarSource(" "); got != AvatarSourceNone {
		t.Fatalf("empty avatar source = %q", got)
	}

	identity := OAuthIdentity{DisplayName: " Ada "}
	if got := resolveOAuthDisplayName(identity, "fallback"); got != "Ada" {
		t.Fatalf("display name = %q", got)
	}
	if got := resolveOAuthDisplayName(OAuthIdentity{}, "fallback"); got != "fallback" {
		t.Fatalf("fallback display name = %q", got)
	}
}

func TestUserPublicAndParseOAuthProvider(t *testing.T) {
	userID := uuid.New()
	user := User{
		ID:       userID,
		Email:    "u@example.com",
		Username: "player",
	}
	public := user.Public()
	if public.ID != userID || public.Email != "u@example.com" || public.Username != "player" {
		t.Fatalf("public user = %#v", public)
	}

	provider, err := ParseOAuthProvider("github")
	if err != nil || provider != OAuthProviderGitHub {
		t.Fatalf("github provider = %q, %v", provider, err)
	}
	provider, err = ParseOAuthProvider("google")
	if err != nil || provider != OAuthProviderGoogle {
		t.Fatalf("google provider = %q, %v", provider, err)
	}
	if _, err := ParseOAuthProvider("unknown"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("unknown provider error = %v, want ErrInvalidInput", err)
	}
}
