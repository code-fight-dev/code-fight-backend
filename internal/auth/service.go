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

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,24}$`)

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
		PasswordHash: passwordHash,
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
			ID:           uuid.New(),
			Email:        email,
			Username:     username,
			PasswordHash: randomPasswordHash,
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
	if err != nil && !errors.Is(err, ErrOAuthAccountLinked) {
		return AuthResult{}, err
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

	if err := s.repository.DeleteExpiredSessions(ctx, time.Now()); err != nil {
		return PublicUser{}, err
	}

	session, err := s.repository.GetSessionByTokenHash(ctx, hashSessionToken(sessionToken))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return PublicUser{}, ErrUnauthorized
		}

		return PublicUser{}, fmt.Errorf("get session: %w", err)
	}

	if session.ExpiresAt.Before(time.Now()) {
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
		return "", "", "", fmt.Errorf("%w: username must be 3-24 chars and use letters, numbers, or underscores", ErrInvalidInput)
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return "", "", "", fmt.Errorf("%w: valid email is required", ErrInvalidInput)
	}

	password := input.Password
	if len(password) < 8 || len(password) > 72 {
		return "", "", "", fmt.Errorf("%w: password must be between 8 and 72 characters", ErrInvalidInput)
	}

	return username, email, password, nil
}

func normalizeSignInInput(input SignInInput) (string, string, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return "", "", fmt.Errorf("%w: valid email is required", ErrInvalidInput)
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
		base = "player"
	}

	if len(base) > 24 {
		base = base[:24]
	}
	if len(base) < 3 {
		base = "player"
	}

	candidate := base
	for attempt := 0; attempt < 10; attempt++ {
		if _, err := s.repository.GetUserByUsername(ctx, candidate); errors.Is(err, ErrNotFound) {
			return candidate, nil
		} else if err != nil {
			return "", fmt.Errorf("check oauth username uniqueness: %w", err)
		}

		suffix := "_" + strconv.Itoa(attempt+1)
		trimmedBase := base
		if len(trimmedBase)+len(suffix) > 24 {
			trimmedBase = trimmedBase[:24-len(suffix)]
		}
		candidate = trimmedBase + suffix
	}

	return "player_" + strings.ToLower(uuid.NewString()[:8]), nil
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
	if len(result) > 24 {
		result = result[:24]
	}

	return result
}
