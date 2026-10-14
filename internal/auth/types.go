package auth

import (
	"time"

	"code-fight-backend/internal/access"

	"github.com/google/uuid"
)

const (
	AvatarSourceNone     = "none"
	AvatarSourceProvider = "provider"
	AvatarSourceCustom   = "custom"
)

type User struct {
	ID                uuid.UUID
	Role              access.Role
	RoleVersion       int64
	Email             string
	Username          string
	DisplayName       string
	PasswordHash      string
	AvatarURL         string
	ProviderAvatarURL string
	AvatarSource      string
	CreatedAt         time.Time
	UpdatedAt         time.Time
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

type PasswordResetCode struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CodeHash  string
	ExpiresAt time.Time
	UsedAt    *time.Time
	Attempts  int
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
	AvatarURL      string
}

type PublicUser struct {
	ID          uuid.UUID   `json:"id"`
	Role        access.Role `json:"role"`
	RoleVersion int64       `json:"roleVersion"`
	Email       string      `json:"email"`
	Username    string      `json:"username"`
	CreatedAt   time.Time   `json:"createdAt"`
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

type PasswordResetRequestInput struct {
	Email        string `json:"email"`
	CaptchaToken string `json:"captchaToken"`
}

type PasswordResetConfirmInput struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
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
		ID:          u.ID,
		Role:        u.Role,
		RoleVersion: u.RoleVersion,
		Email:       u.Email,
		Username:    u.Username,
		CreatedAt:   u.CreatedAt,
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
