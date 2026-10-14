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

	"code-fight-backend/internal/access"

	"github.com/google/uuid"
)

const (
	oauthUsernameFallbackPrefix = "player"
	oauthUsernameMaxAttempts    = 10
	passwordMaxLength           = 72
	passwordMinLength           = 8
	passwordResetCodeTTL        = 10 * time.Minute
	usernameMinLength           = 3
	usernameMaxLength           = 24
)

var usernamePattern = regexp.MustCompile(
	fmt.Sprintf("^[a-zA-Z0-9_]{%d,%d}$", usernameMinLength, usernameMaxLength),
)

type Service struct {
	repository AuthRepository
	sessionTTL time.Duration
	mailer     PasswordResetMailer
}

type AuthRepository interface {
	CreateUser(ctx context.Context, user User) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByUsername(ctx context.Context, username string) (User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (User, error)
	GetOAuthAccount(ctx context.Context, provider OAuthProvider, providerUserID string) (OAuthAccount, error)
	GetOAuthAccountByUserProvider(ctx context.Context, userID uuid.UUID, provider OAuthProvider) (OAuthAccount, error)
	CreateOAuthAccount(ctx context.Context, account OAuthAccount) (OAuthAccount, error)
	UpdateProviderAvatar(ctx context.Context, userID uuid.UUID, avatarURL string) error
	CreateSession(ctx context.Context, session Session) (Session, error)
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (Session, error)
	DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error
	DeleteExpiredSessions(ctx context.Context, before time.Time) error
	DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error
	CreatePasswordResetCode(ctx context.Context, resetCode PasswordResetCode) (PasswordResetCode, error)
	GetActivePasswordResetCode(ctx context.Context, userID uuid.UUID, now time.Time) (PasswordResetCode, error)
	IncrementPasswordResetAttempts(ctx context.Context, codeID uuid.UUID) error
	MarkPasswordResetCodeUsed(ctx context.Context, codeID uuid.UUID) error
	UpdateUserPasswordHash(ctx context.Context, userID uuid.UUID, passwordHash string) error
}

type PasswordResetMailer interface {
	SendPasswordResetCode(ctx context.Context, to string, code string) error
}

func NewService(repository AuthRepository, sessionTTL time.Duration, mailer PasswordResetMailer) *Service {
	return &Service{
		repository: repository,
		sessionTTL: sessionTTL,
		mailer:     mailer,
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
		Role:         access.User,
		RoleVersion:  1,
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
			Role:              access.User,
			RoleVersion:       1,
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

func (s *Service) RequestPasswordReset(ctx context.Context, input PasswordResetRequestInput) error {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return err
	}

	user, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}

		return fmt.Errorf("get password reset user by email: %w", err)
	}

	if s.mailer == nil {
		return fmt.Errorf("password reset mailer is not configured")
	}

	code, err := generatePasswordResetCode()
	if err != nil {
		return fmt.Errorf("generate password reset code: %w", err)
	}
	codeHash, err := hashPassword(code)
	if err != nil {
		return fmt.Errorf("hash password reset code: %w", err)
	}

	_, err = s.repository.CreatePasswordResetCode(ctx, PasswordResetCode{
		ID:        uuid.New(),
		UserID:    user.ID,
		CodeHash:  codeHash,
		ExpiresAt: time.Now().Add(passwordResetCodeTTL),
	})
	if err != nil {
		return err
	}

	if err := s.mailer.SendPasswordResetCode(ctx, user.Email, code); err != nil {
		return fmt.Errorf("send password reset code: %w", err)
	}

	return nil
}

func (s *Service) ConfirmPasswordReset(ctx context.Context, input PasswordResetConfirmInput) error {
	email, code, password, err := normalizePasswordResetConfirmInput(input)
	if err != nil {
		return err
	}

	user, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrResetCodeInvalid
		}

		return fmt.Errorf("get password reset user by email: %w", err)
	}

	resetCode, err := s.repository.GetActivePasswordResetCode(ctx, user.ID, time.Now())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrResetCodeExpired
		}

		return fmt.Errorf("get active password reset code: %w", err)
	}

	if resetCode.Attempts >= passwordResetMaxAttempts {
		_ = s.repository.MarkPasswordResetCodeUsed(ctx, resetCode.ID)
		return ErrResetCodeInvalid
	}

	if err := comparePassword(resetCode.CodeHash, code); err != nil {
		_ = s.repository.IncrementPasswordResetAttempts(ctx, resetCode.ID)
		return ErrResetCodeInvalid
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("hash reset password: %w", err)
	}

	if err := s.repository.UpdateUserPasswordHash(ctx, user.ID, passwordHash); err != nil {
		return err
	}
	if err := s.repository.MarkPasswordResetCodeUsed(ctx, resetCode.ID); err != nil {
		return err
	}
	if err := s.repository.DeleteSessionsByUserID(ctx, user.ID); err != nil {
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

func normalizePasswordResetConfirmInput(input PasswordResetConfirmInput) (string, string, string, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return "", "", "", err
	}

	code := strings.TrimSpace(input.Code)
	if len(code) != 6 {
		return "", "", "", fmt.Errorf("%w: reset code must be 6 digits", ErrInvalidInput)
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return "", "", "", fmt.Errorf("%w: reset code must be 6 digits", ErrInvalidInput)
		}
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

	return email, code, password, nil
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
