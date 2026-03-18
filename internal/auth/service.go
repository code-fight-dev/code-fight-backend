package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	oauthUsernameFallbackPrefix = "player"
	oauthUsernameMaxAttempts    = 10
	passwordMaxLength           = 72
	passwordMinLength           = 8
	usernameMinLength           = 3
	usernameMaxLength           = 24
)

var usernamePattern = regexp.MustCompile(
	fmt.Sprintf("^[a-zA-Z0-9_]{%d,%d}$", usernameMinLength, usernameMaxLength),
)

type Service struct {
	repository *Repository
	sessionTTL time.Duration
}

func NewService(repository *Repository, sessionTTL time.Duration) *Service {
	return &Service{
		repository: repository,
		sessionTTL: sessionTTL,
	}
}

func (s *Service) SignUp(ctx context.Context, input SignUpInput, meta SessionMeta) (AuthResult, error) {
	username, email, password, err := normalizeSignUpInput(input)
	if err != nil {
		return AuthResult{}, err
	}

	if _, err := s.repository.GetUserByEmail(ctx, email); err == nil {
		return AuthResult{}, ErrEmailTaken
	} else if !errors.Is(err, ErrNotFound) {
		return AuthResult{}, fmt.Errorf("check email uniqueness: %w", err)
	}

	if _, err := s.repository.GetUserByUsername(ctx, username); err == nil {
		return AuthResult{}, ErrUsernameTaken
	} else if !errors.Is(err, ErrNotFound) {
		return AuthResult{}, fmt.Errorf("check username uniqueness: %w", err)
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repository.CreateUser(ctx, User{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		DisplayName:  username,
		PasswordHash: passwordHash,
		AvatarSource: AvatarSourceNone,
	})
	if err != nil {
		return AuthResult{}, err
	}

	return s.createSessionForUser(ctx, user, meta)
}

func (s *Service) SignIn(ctx context.Context, input SignInInput, meta SessionMeta) (AuthResult, error) {
	email, password, err := normalizeSignInInput(input)
	if err != nil {
		return AuthResult{}, err
	}

	user, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return AuthResult{}, ErrInvalidCredentials
		}

		return AuthResult{}, fmt.Errorf("get user by email: %w", err)
	}

	if err := comparePassword(user.PasswordHash, password); err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	return s.createSessionForUser(ctx, user, meta)
}

func (s *Service) SignInWithOAuth(ctx context.Context, identity OAuthIdentity, meta SessionMeta) (AuthResult, error) {
	email := strings.ToLower(strings.TrimSpace(identity.Email))
	avatarURL := strings.TrimSpace(identity.AvatarURL)
	if email == "" || !identity.EmailVerified {
		return AuthResult{}, ErrOAuthEmailRequired
	}

	if existingAccount, err := s.repository.GetOAuthAccount(ctx, identity.Provider, identity.ProviderUserID); err == nil {
		user, err := s.repository.GetUserByID(ctx, existingAccount.UserID)
		if err != nil {
			return AuthResult{}, err
		}

		return s.createSessionForUser(ctx, user, meta)
	} else if !errors.Is(err, ErrNotFound) {
		return AuthResult{}, fmt.Errorf("get oauth account: %w", err)
	}

	user, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return AuthResult{}, fmt.Errorf("get user by email for oauth: %w", err)
		}

		username, usernameErr := s.generateOAuthUsername(ctx, identity)
		if usernameErr != nil {
			return AuthResult{}, usernameErr
		}

		randomPasswordHash, hashErr := hashPassword(uuid.NewString())
		if hashErr != nil {
			return AuthResult{}, fmt.Errorf("hash oauth fallback password: %w", hashErr)
		}

		user, err = s.repository.CreateUser(ctx, User{
			ID:                uuid.New(),
			Email:             email,
			Username:          username,
			DisplayName:       resolveOAuthDisplayName(identity, username),
			PasswordHash:      randomPasswordHash,
			ProviderAvatarURL: avatarURL,
			AvatarSource:      resolveAvatarSource(avatarURL),
		})
		if err != nil {
			return AuthResult{}, err
		}
	}

	_, err = s.repository.CreateOAuthAccount(ctx, OAuthAccount{
		ID:             uuid.New(),
		UserID:         user.ID,
		Provider:       identity.Provider,
		ProviderUserID: identity.ProviderUserID,
		ProviderEmail:  email,
	})
	if err != nil {
		if !errors.Is(err, ErrOAuthAccountLinked) {
			return AuthResult{}, err
		}

		resolvedUser, resolveErr := s.resolveOAuthLinkedUser(ctx, identity.Provider, identity.ProviderUserID, user.ID)
		if resolveErr != nil {
			return AuthResult{}, resolveErr
		}

		user = resolvedUser
	}

	if err := s.repository.UpdateProviderAvatar(ctx, user.ID, avatarURL); err != nil {
		return AuthResult{}, fmt.Errorf("update oauth provider avatar: %w", err)
	}

	return s.createSessionForUser(ctx, user, meta)
}

func (s *Service) SignOut(ctx context.Context, sessionToken string) error {
	if sessionToken == "" {
		return nil
	}

	if err := s.repository.DeleteSessionByTokenHash(ctx, hashSessionToken(sessionToken)); err != nil {
		return err
	}

	return nil
}

func (s *Service) CurrentUser(ctx context.Context, sessionToken string) (PublicUser, error) {
	if sessionToken == "" {
		return PublicUser{}, ErrUnauthorized
	}

	session, err := s.repository.GetSessionByTokenHash(ctx, hashSessionToken(sessionToken))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return PublicUser{}, ErrUnauthorized
		}

		return PublicUser{}, fmt.Errorf("get session: %w", err)
	}

	now := time.Now()
	if session.ExpiresAt.Before(now) {
		_ = s.repository.DeleteSessionByTokenHash(ctx, session.TokenHash)
		return PublicUser{}, ErrUnauthorized
	}

	user, err := s.repository.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return PublicUser{}, ErrUnauthorized
		}

		return PublicUser{}, fmt.Errorf("get user by id: %w", err)
	}

	return user.Public(), nil
}

func (s *Service) CleanupExpiredSessions(ctx context.Context) error {
	if err := s.repository.DeleteExpiredSessions(ctx, time.Now()); err != nil {
		return fmt.Errorf("cleanup expired sessions: %w", err)
	}

	return nil
}

func (s *Service) resolveOAuthLinkedUser(ctx context.Context, provider OAuthProvider, providerUserID string, expectedUserID uuid.UUID) (User, error) {
	account, err := s.repository.GetOAuthAccount(ctx, provider, providerUserID)
	if err == nil {
		if account.UserID != expectedUserID {
			return User{}, ErrOAuthAccountLinked
		}

		user, err := s.repository.GetUserByID(ctx, account.UserID)
		if err != nil {
			return User{}, fmt.Errorf("get oauth-linked user by id: %w", err)
		}

		return user, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return User{}, fmt.Errorf("re-read oauth account: %w", err)
	}

	account, err = s.repository.GetOAuthAccountByUserProvider(ctx, expectedUserID, provider)
	if err == nil {
		if account.ProviderUserID != providerUserID {
			return User{}, ErrOAuthAccountLinked
		}

		user, err := s.repository.GetUserByID(ctx, account.UserID)
		if err != nil {
			return User{}, fmt.Errorf("get provider-linked user by id: %w", err)
		}

		return user, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return User{}, fmt.Errorf("get oauth account by user/provider: %w", err)
	}

	return User{}, ErrOAuthAccountLinked
}

func (s *Service) createSessionForUser(ctx context.Context, user User, meta SessionMeta) (AuthResult, error) {
	token, tokenHash, err := generateSessionToken()
	if err != nil {
		return AuthResult{}, fmt.Errorf("generate session token: %w", err)
	}

	expiresAt := time.Now().Add(s.sessionTTL)
	_, err = s.repository.CreateSession(ctx, Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		UserAgent: strings.TrimSpace(meta.UserAgent),
		IPAddress: strings.TrimSpace(meta.IPAddress),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		User:         user.Public(),
		SessionToken: token,
		ExpiresAt:    expiresAt,
	}, nil
}

func normalizeSignUpInput(input SignUpInput) (string, string, string, error) {
	username := strings.TrimSpace(input.Username)
	if !usernamePattern.MatchString(username) {
		return "", "", "", fmt.Errorf(
			"%w: username must be %d-%d chars and use letters, numbers, or underscores",
			ErrInvalidInput,
			usernameMinLength,
			usernameMaxLength,
		)
	}

	email, err := normalizeEmail(input.Email)
	if err != nil {
		return "", "", "", err
	}

	password := input.Password
	if len(password) < passwordMinLength || len(password) > passwordMaxLength {
		return "", "", "", fmt.Errorf(
			"%w: password must be between %d and %d characters",
			ErrInvalidInput,
			passwordMinLength,
			passwordMaxLength,
		)
	}

	return username, email, password, nil
}

func normalizeSignInInput(input SignInInput) (string, string, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return "", "", err
	}

	password := input.Password
	if password == "" {
		return "", "", fmt.Errorf("%w: password is required", ErrInvalidInput)
	}

	return email, password, nil
}

func (s *Service) generateOAuthUsername(ctx context.Context, identity OAuthIdentity) (string, error) {
	base := sanitizeUsernameCandidate(identity.Username)
	if base == "" {
		base = sanitizeUsernameCandidate(strings.Split(strings.ToLower(strings.TrimSpace(identity.Email)), "@")[0])
	}
	if base == "" {
		base = oauthUsernameFallbackPrefix
	}

	if len(base) > usernameMaxLength {
		base = base[:usernameMaxLength]
	}
	if len(base) < usernameMinLength {
		base = oauthUsernameFallbackPrefix
	}

	candidate := base
	for attempt := 0; attempt < oauthUsernameMaxAttempts; attempt++ {
		if _, err := s.repository.GetUserByUsername(ctx, candidate); errors.Is(err, ErrNotFound) {
			return candidate, nil
		} else if err != nil {
			return "", fmt.Errorf("check oauth username uniqueness: %w", err)
		}

		suffix := "_" + strconv.Itoa(attempt+1)
		trimmedBase := base
		if len(trimmedBase)+len(suffix) > usernameMaxLength {
			trimmedBase = trimmedBase[:usernameMaxLength-len(suffix)]
		}
		candidate = trimmedBase + suffix
	}

	return oauthUsernameFallbackPrefix + "_" + strings.ToLower(uuid.NewString()[:8]), nil
}

func sanitizeUsernameCandidate(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	var builder strings.Builder

	for _, r := range normalized {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '_' || r == '-' || r == '.':
			builder.WriteRune('_')
		}
	}

	result := builder.String()
	result = strings.Trim(result, "_")
	if len(result) > usernameMaxLength {
		result = result[:usernameMaxLength]
	}

	return result
}

func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if _, err := mail.ParseAddress(email); err != nil {
		return "", fmt.Errorf("%w: valid email is required", ErrInvalidInput)
	}

	return email, nil
}

func resolveAvatarSource(avatarURL string) string {
	if strings.TrimSpace(avatarURL) != "" {
		return AvatarSourceProvider
	}

	return AvatarSourceNone
}

func resolveOAuthDisplayName(identity OAuthIdentity, fallbackUsername string) string {
	displayName := strings.TrimSpace(identity.DisplayName)
	if displayName != "" {
		return displayName
	}

	return fallbackUsername
}
