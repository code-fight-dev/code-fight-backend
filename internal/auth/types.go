package auth

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	UserAgent string
	IPAddress string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type OAuthProvider string

const (
	OAuthProviderGitHub OAuthProvider = "github"
	OAuthProviderGoogle OAuthProvider = "google"
)

type OAuthAccount struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Provider       OAuthProvider
	ProviderUserID string
	ProviderEmail  string
	CreatedAt      time.Time
}

type OAuthIdentity struct {
	Provider       OAuthProvider
	ProviderUserID string
	Email          string
	EmailVerified  bool
	Username       string
	DisplayName    string
}

type PublicUser struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
}

type SignUpInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignInInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SessionMeta struct {
	UserAgent string
	IPAddress string
}

type AuthResult struct {
	User         PublicUser
	SessionToken string
	ExpiresAt    time.Time
}

func (u User) Public() PublicUser {
	return PublicUser{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username,
		CreatedAt: u.CreatedAt,
	}
}

func ParseOAuthProvider(raw string) (OAuthProvider, error) {
	switch OAuthProvider(raw) {
	case OAuthProviderGitHub:
		return OAuthProviderGitHub, nil
	case OAuthProviderGoogle:
		return OAuthProviderGoogle, nil
	default:
		return "", ErrInvalidInput
	}
}
