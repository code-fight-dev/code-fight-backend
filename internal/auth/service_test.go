package auth

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestServiceSignUpCreatesUserAndSession(t *testing.T) {
	repo := newFakeAuthRepo()
	service := NewService(repo, time.Hour, nil)

	result, err := service.SignUp(context.Background(), SignUpInput{
		Username: "player",
		Email:    "player@example.com",
		Password: "password123",
	}, SessionMeta{UserAgent: "browser", IPAddress: "127.0.0.1"})
	if err != nil {
		t.Fatalf("SignUp returned error: %v", err)
	}

	if result.User.Username != "player" || result.SessionToken == "" {
		t.Fatalf("result = %#v", result)
	}
	if len(repo.usersByID) != 1 {
		t.Fatalf("users = %d, want 1", len(repo.usersByID))
	}
	if len(repo.sessionsByHash) != 1 {
		t.Fatalf("sessions = %d, want 1", len(repo.sessionsByHash))
	}
}

func TestServiceSignUpRejectsTakenEmailAndUsername(t *testing.T) {
	repo := newFakeAuthRepo()
	existing := User{ID: uuid.New(), Email: "player@example.com", Username: "player"}
	repo.storeUser(existing)
	service := NewService(repo, time.Hour, nil)

	_, err := service.SignUp(context.Background(), SignUpInput{
		Username: "other",
		Email:    "player@example.com",
		Password: "password123",
	}, SessionMeta{})
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("SignUp error = %v, want ErrEmailTaken", err)
	}

	_, err = service.SignUp(context.Background(), SignUpInput{
		Username: "player",
		Email:    "other@example.com",
		Password: "password123",
	}, SessionMeta{})
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("SignUp error = %v, want ErrUsernameTaken", err)
	}
}

func TestServiceSignIn(t *testing.T) {
	repo := newFakeAuthRepo()
	hash, err := hashPassword("password123")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	repo.storeUser(User{ID: uuid.New(), Email: "player@example.com", Username: "player", PasswordHash: hash})
	service := NewService(repo, time.Hour, nil)

	result, err := service.SignIn(context.Background(), SignInInput{
		Email:    "player@example.com",
		Password: "password123",
	}, SessionMeta{})
	if err != nil {
		t.Fatalf("SignIn returned error: %v", err)
	}
	if result.SessionToken == "" {
		t.Fatal("session token is empty")
	}

	_, err = service.SignIn(context.Background(), SignInInput{
		Email:    "player@example.com",
		Password: "wrong-password",
	}, SessionMeta{})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password error = %v, want ErrInvalidCredentials", err)
	}
}

func TestServiceSignInRejectsInvalidInputsAndMissingUser(t *testing.T) {
	service := NewService(newFakeAuthRepo(), time.Hour, nil)

	tests := []SignInInput{
		{Email: "not-email", Password: "password123"},
		{Email: "missing@example.com", Password: ""},
		{Email: "missing@example.com", Password: "password123"},
	}
	for _, input := range tests {
		if _, err := service.SignIn(context.Background(), input, SessionMeta{}); err == nil {
			t.Fatalf("SignIn(%#v) returned nil error", input)
		}
	}
}

func TestServiceCurrentUser(t *testing.T) {
	repo := newFakeAuthRepo()
	user := User{ID: uuid.New(), Email: "player@example.com", Username: "player"}
	repo.storeUser(user)
	token := "plain-token"
	repo.sessionsByHash[hashSessionToken(token)] = Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashSessionToken(token),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	service := NewService(repo, time.Hour, nil)

	current, err := service.CurrentUser(context.Background(), token)
	if err != nil {
		t.Fatalf("CurrentUser returned error: %v", err)
	}
	if current.ID != user.ID {
		t.Fatalf("current user = %#v", current)
	}

	_, err = service.CurrentUser(context.Background(), "")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("empty token error = %v, want ErrUnauthorized", err)
	}

	expiredToken := "expired"
	repo.sessionsByHash[hashSessionToken(expiredToken)] = Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashSessionToken(expiredToken),
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	_, err = service.CurrentUser(context.Background(), expiredToken)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expired token error = %v, want ErrUnauthorized", err)
	}
	if _, ok := repo.sessionsByHash[hashSessionToken(expiredToken)]; ok {
		t.Fatal("expired session was not deleted")
	}
}

func TestServiceSignOutAndCleanup(t *testing.T) {
	repo := newFakeAuthRepo()
	token := "plain-token"
	repo.sessionsByHash[hashSessionToken(token)] = Session{TokenHash: hashSessionToken(token)}
	service := NewService(repo, time.Hour, nil)

	if err := service.SignOut(context.Background(), token); err != nil {
		t.Fatalf("SignOut returned error: %v", err)
	}
	if _, ok := repo.sessionsByHash[hashSessionToken(token)]; ok {
		t.Fatal("session was not deleted")
	}
	if err := service.SignOut(context.Background(), ""); err != nil {
		t.Fatalf("empty SignOut returned error: %v", err)
	}
	if err := service.CleanupExpiredSessions(context.Background()); err != nil {
		t.Fatalf("CleanupExpiredSessions returned error: %v", err)
	}
	if repo.deleteExpiredCalls != 1 {
		t.Fatalf("deleteExpiredCalls = %d", repo.deleteExpiredCalls)
	}
}

func TestServicePasswordResetFlow(t *testing.T) {
	repo := newFakeAuthRepo()
	user := User{ID: uuid.New(), Email: "player@example.com", Username: "player"}
	repo.storeUser(user)
	mailer := &fakeMailer{}
	service := NewService(repo, time.Hour, mailer)

	if err := service.RequestPasswordReset(context.Background(), PasswordResetRequestInput{Email: "player@example.com"}); err != nil {
		t.Fatalf("RequestPasswordReset returned error: %v", err)
	}
	if mailer.to != "player@example.com" || mailer.code == "" {
		t.Fatalf("mailer = %#v", mailer)
	}
	if len(repo.resetCodesByUser) != 1 {
		t.Fatalf("reset codes = %d", len(repo.resetCodesByUser))
	}

	reset := repo.resetCodesByUser[user.ID]
	if err := service.ConfirmPasswordReset(context.Background(), PasswordResetConfirmInput{
		Email:    "player@example.com",
		Code:     mailer.code,
		Password: "new-password",
	}); err != nil {
		t.Fatalf("ConfirmPasswordReset returned error: %v", err)
	}
	if repo.resetCodesByID[reset.ID].UsedAt == nil {
		t.Fatal("reset code was not marked used")
	}
	if repo.deleteSessionsByUserCalls != 1 {
		t.Fatalf("deleteSessionsByUserCalls = %d", repo.deleteSessionsByUserCalls)
	}
}

func TestServicePasswordResetErrors(t *testing.T) {
	repo := newFakeAuthRepo()
	service := NewService(repo, time.Hour, nil)

	if err := service.RequestPasswordReset(context.Background(), PasswordResetRequestInput{Email: "missing@example.com"}); err != nil {
		t.Fatalf("missing user reset returned error: %v", err)
	}

	user := User{ID: uuid.New(), Email: "player@example.com", Username: "player"}
	repo.storeUser(user)
	if err := service.RequestPasswordReset(context.Background(), PasswordResetRequestInput{Email: "player@example.com"}); err == nil {
		t.Fatal("RequestPasswordReset returned nil error without mailer")
	}

	hash, err := hashPassword("123456")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	resetID := uuid.New()
	repo.resetCodesByUser[user.ID] = PasswordResetCode{
		ID:       resetID,
		UserID:   user.ID,
		CodeHash: hash,
		Attempts: passwordResetMaxAttempts,
	}
	repo.resetCodesByID[resetID] = repo.resetCodesByUser[user.ID]
	if err := service.ConfirmPasswordReset(context.Background(), PasswordResetConfirmInput{
		Email:    "player@example.com",
		Code:     "123456",
		Password: "new-password",
	}); !errors.Is(err, ErrResetCodeInvalid) {
		t.Fatalf("max attempts error = %v, want ErrResetCodeInvalid", err)
	}
}

func TestServiceConfirmPasswordResetInvalidAndExpiredBranches(t *testing.T) {
	repo := newFakeAuthRepo()
	service := NewService(repo, time.Hour, nil)

	if err := service.ConfirmPasswordReset(context.Background(), PasswordResetConfirmInput{
		Email:    "missing@example.com",
		Code:     "123456",
		Password: "new-password",
	}); !errors.Is(err, ErrResetCodeInvalid) {
		t.Fatalf("missing user error = %v, want ErrResetCodeInvalid", err)
	}

	user := repo.storeUser(User{ID: uuid.New(), Email: "player@example.com", Username: "player"})
	if err := service.ConfirmPasswordReset(context.Background(), PasswordResetConfirmInput{
		Email:    user.Email,
		Code:     "123456",
		Password: "new-password",
	}); !errors.Is(err, ErrResetCodeExpired) {
		t.Fatalf("missing reset code error = %v, want ErrResetCodeExpired", err)
	}

	hash, err := hashPassword("123456")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	resetID := uuid.New()
	repo.resetCodesByUser[user.ID] = PasswordResetCode{
		ID:       resetID,
		UserID:   user.ID,
		CodeHash: hash,
	}
	repo.resetCodesByID[resetID] = repo.resetCodesByUser[user.ID]
	if err := service.ConfirmPasswordReset(context.Background(), PasswordResetConfirmInput{
		Email:    user.Email,
		Code:     "654321",
		Password: "new-password",
	}); !errors.Is(err, ErrResetCodeInvalid) {
		t.Fatalf("wrong code error = %v, want ErrResetCodeInvalid", err)
	}
	if repo.resetCodesByID[resetID].Attempts != 1 {
		t.Fatalf("attempts = %d, want 1", repo.resetCodesByID[resetID].Attempts)
	}
}

func TestServiceSignInWithOAuthCreatesAndLinksUser(t *testing.T) {
	repo := newFakeAuthRepo()
	service := NewService(repo, time.Hour, nil)

	result, err := service.SignInWithOAuth(context.Background(), OAuthIdentity{
		Provider:       OAuthProviderGitHub,
		ProviderUserID: "github-1",
		Email:          "PLAYER@example.com",
		EmailVerified:  true,
		Username:       "Player.Name",
		DisplayName:    "Player Name",
		AvatarURL:      "https://avatar.example.com/u.png",
	}, SessionMeta{})
	if err != nil {
		t.Fatalf("SignInWithOAuth returned error: %v", err)
	}
	if result.User.Username != "player_name" {
		t.Fatalf("username = %q", result.User.Username)
	}
	if len(repo.oauthByProviderID) != 1 {
		t.Fatalf("oauth accounts = %d", len(repo.oauthByProviderID))
	}

	again, err := service.SignInWithOAuth(context.Background(), OAuthIdentity{
		Provider:       OAuthProviderGitHub,
		ProviderUserID: "github-1",
		Email:          "player@example.com",
		EmailVerified:  true,
	}, SessionMeta{})
	if err != nil {
		t.Fatalf("second SignInWithOAuth returned error: %v", err)
	}
	if again.User.ID != result.User.ID {
		t.Fatalf("second user id = %s, want %s", again.User.ID, result.User.ID)
	}

	_, err = service.SignInWithOAuth(context.Background(), OAuthIdentity{Email: "player@example.com"}, SessionMeta{})
	if !errors.Is(err, ErrOAuthEmailRequired) {
		t.Fatalf("unverified email error = %v, want ErrOAuthEmailRequired", err)
	}
}

func TestServiceSignInWithOAuthExistingAccountAndEmail(t *testing.T) {
	repo := newFakeAuthRepo()
	user := repo.storeUser(User{ID: uuid.New(), Email: "player@example.com", Username: "player"})
	account := OAuthAccount{ID: uuid.New(), UserID: user.ID, Provider: OAuthProviderGitHub, ProviderUserID: "github-1"}
	repo.oauthByProviderID[string(OAuthProviderGitHub)+":github-1"] = account
	repo.oauthByUserProvider[user.ID.String()+":"+string(OAuthProviderGitHub)] = account
	service := NewService(repo, time.Hour, nil)

	existing, err := service.SignInWithOAuth(context.Background(), OAuthIdentity{
		Provider:       OAuthProviderGitHub,
		ProviderUserID: "github-1",
		Email:          "player@example.com",
		EmailVerified:  true,
	}, SessionMeta{})
	if err != nil {
		t.Fatalf("existing account SignInWithOAuth returned error: %v", err)
	}
	if existing.User.ID != user.ID {
		t.Fatalf("existing user = %#v", existing.User)
	}

	repo2 := newFakeAuthRepo()
	existingUser := repo2.storeUser(User{ID: uuid.New(), Email: "linked@example.com", Username: "linked"})
	service = NewService(repo2, time.Hour, nil)
	linked, err := service.SignInWithOAuth(context.Background(), OAuthIdentity{
		Provider:       OAuthProviderGoogle,
		ProviderUserID: "google-1",
		Email:          "linked@example.com",
		EmailVerified:  true,
		AvatarURL:      "https://avatar.example.com/linked.png",
	}, SessionMeta{})
	if err != nil {
		t.Fatalf("existing email SignInWithOAuth returned error: %v", err)
	}
	if linked.User.ID != existingUser.ID || repo2.usersByID[existingUser.ID].ProviderAvatarURL == "" {
		t.Fatalf("linked=%#v stored=%#v", linked.User, repo2.usersByID[existingUser.ID])
	}
}

func TestServiceSignInWithOAuthResolvesLinkedRace(t *testing.T) {
	repo := newFakeAuthRepo()
	user := repo.storeUser(User{ID: uuid.New(), Email: "player@example.com", Username: "player"})
	account := OAuthAccount{ID: uuid.New(), UserID: user.ID, Provider: OAuthProviderGitHub, ProviderUserID: "github-1"}
	repo.oauthByProviderID[string(OAuthProviderGitHub)+":github-1"] = account
	repo.oauthByUserProvider[user.ID.String()+":"+string(OAuthProviderGitHub)] = account
	service := NewService(repo, time.Hour, nil)

	resolved, err := service.resolveOAuthLinkedUser(context.Background(), OAuthProviderGitHub, "github-1", user.ID)
	if err != nil || resolved.ID != user.ID {
		t.Fatalf("resolveOAuthLinkedUser = %#v, %v", resolved, err)
	}

	_, err = service.resolveOAuthLinkedUser(context.Background(), OAuthProviderGitHub, "github-1", uuid.New())
	if !errors.Is(err, ErrOAuthAccountLinked) {
		t.Fatalf("wrong expected user error = %v", err)
	}

	repo = newFakeAuthRepo()
	user = repo.storeUser(User{ID: uuid.New(), Email: "player@example.com", Username: "player"})
	account = OAuthAccount{ID: uuid.New(), UserID: user.ID, Provider: OAuthProviderGitHub, ProviderUserID: "github-2"}
	repo.oauthByUserProvider[user.ID.String()+":"+string(OAuthProviderGitHub)] = account
	service = NewService(repo, time.Hour, nil)
	resolved, err = service.resolveOAuthLinkedUser(context.Background(), OAuthProviderGitHub, "github-2", user.ID)
	if err != nil || resolved.ID != user.ID {
		t.Fatalf("resolve by user/provider = %#v, %v", resolved, err)
	}

	_, err = service.resolveOAuthLinkedUser(context.Background(), OAuthProviderGitHub, "different", user.ID)
	if !errors.Is(err, ErrOAuthAccountLinked) {
		t.Fatalf("different provider user error = %v", err)
	}
}

func TestGenerateOAuthUsernameFallsBackAfterCollisions(t *testing.T) {
	repo := newFakeAuthRepo()
	for i := 0; i < oauthUsernameMaxAttempts; i++ {
		username := "player"
		if i > 0 {
			username = "player_" + strconv.Itoa(i)
		}
		repo.storeUser(User{ID: uuid.New(), Email: username + "@example.com", Username: username})
	}
	service := NewService(repo, time.Hour, nil)

	username, err := service.generateOAuthUsername(context.Background(), OAuthIdentity{Email: "player@example.com"})
	if err != nil {
		t.Fatalf("generateOAuthUsername returned error: %v", err)
	}
	if !strings.HasPrefix(username, "player_") || len(username) <= len("player_") {
		t.Fatalf("fallback username = %q", username)
	}
}

func TestServiceSignUpRepositoryErrors(t *testing.T) {
	checkErr := errors.New("check failed")
	service := NewService(&errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), getEmailErr: checkErr}, time.Hour, nil)
	_, err := service.SignUp(context.Background(), SignUpInput{
		Username: "player",
		Email:    "player@example.com",
		Password: "password123",
	}, SessionMeta{})
	if err == nil || !strings.Contains(err.Error(), "check email uniqueness") {
		t.Fatalf("email check error = %v", err)
	}

	service = NewService(&errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), getUsernameErr: checkErr}, time.Hour, nil)
	_, err = service.SignUp(context.Background(), SignUpInput{
		Username: "player",
		Email:    "player@example.com",
		Password: "password123",
	}, SessionMeta{})
	if err == nil || !strings.Contains(err.Error(), "check username uniqueness") {
		t.Fatalf("username check error = %v", err)
	}

	createErr := errors.New("create failed")
	service = NewService(&errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), createUserErr: createErr}, time.Hour, nil)
	_, err = service.SignUp(context.Background(), SignUpInput{
		Username: "player",
		Email:    "player@example.com",
		Password: "password123",
	}, SessionMeta{})
	if !errors.Is(err, createErr) {
		t.Fatalf("create error = %v, want %v", err, createErr)
	}
}

func TestServiceSessionAndCurrentUserErrors(t *testing.T) {
	repo := newFakeAuthRepo()
	user := repo.storeUser(User{ID: uuid.New(), Email: "player@example.com", Username: "player"})
	token := "plain-token"
	repo.sessionsByHash[hashSessionToken(token)] = Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashSessionToken(token),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	service := NewService(&errorAuthRepo{fakeAuthRepo: repo, createSessionErr: errors.New("session failed")}, time.Hour, nil)
	_, err := service.SignInWithOAuth(context.Background(), OAuthIdentity{
		Provider:       OAuthProviderGitHub,
		ProviderUserID: "github-1",
		Email:          "oauth@example.com",
		EmailVerified:  true,
	}, SessionMeta{})
	if err == nil {
		t.Fatal("expected create session error")
	}

	sessionErr := errors.New("session read failed")
	service = NewService(&errorAuthRepo{fakeAuthRepo: repo, getSessionErr: sessionErr}, time.Hour, nil)
	_, err = service.CurrentUser(context.Background(), token)
	if err == nil || !strings.Contains(err.Error(), "get session") {
		t.Fatalf("session read error = %v", err)
	}

	userErr := errors.New("user read failed")
	service = NewService(&errorAuthRepo{fakeAuthRepo: repo, getUserByIDErr: userErr}, time.Hour, nil)
	_, err = service.CurrentUser(context.Background(), token)
	if err == nil || !strings.Contains(err.Error(), "get user by id") {
		t.Fatalf("user read error = %v", err)
	}

	missingUserRepo := newFakeAuthRepo()
	missingUserRepo.sessionsByHash[hashSessionToken(token)] = Session{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: hashSessionToken(token),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	service = NewService(missingUserRepo, time.Hour, nil)
	_, err = service.CurrentUser(context.Background(), token)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("missing session user error = %v, want ErrUnauthorized", err)
	}

	expiredRepo := newFakeAuthRepo()
	expiredUser := expiredRepo.storeUser(User{ID: uuid.New(), Email: "expired@example.com", Username: "expired"})
	expiredRepo.sessionsByHash[hashSessionToken(token)] = Session{
		ID:        uuid.New(),
		UserID:    expiredUser.ID,
		TokenHash: hashSessionToken(token),
		ExpiresAt: time.Now().Add(-time.Minute),
	}
	service = NewService(expiredRepo, time.Hour, nil)
	_, err = service.CurrentUser(context.Background(), token)
	if !errors.Is(err, ErrUnauthorized) || len(expiredRepo.sessionsByHash) != 0 {
		t.Fatalf("expired session error=%v sessions=%d", err, len(expiredRepo.sessionsByHash))
	}

	deleteErr := errors.New("delete session failed")
	service = NewService(&errorAuthRepo{fakeAuthRepo: repo, deleteSessionErr: deleteErr}, time.Hour, nil)
	if err := service.SignOut(context.Background(), token); !errors.Is(err, deleteErr) {
		t.Fatalf("sign out delete error = %v, want %v", err, deleteErr)
	}

	cleanupErr := errors.New("cleanup failed")
	service = NewService(&errorAuthRepo{fakeAuthRepo: repo, deleteExpiredErr: cleanupErr}, time.Hour, nil)
	if err := service.CleanupExpiredSessions(context.Background()); err == nil || !strings.Contains(err.Error(), "cleanup expired sessions") {
		t.Fatalf("cleanup error = %v", err)
	}
}

func TestServicePasswordResetRepositoryErrors(t *testing.T) {
	user := User{ID: uuid.New(), Email: "player@example.com", Username: "player"}
	getErr := errors.New("lookup failed")
	service := NewService(&errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), getEmailErr: getErr}, time.Hour, nil)
	if err := service.RequestPasswordReset(context.Background(), PasswordResetRequestInput{Email: user.Email}); err == nil || !strings.Contains(err.Error(), "get password reset user") {
		t.Fatalf("password reset lookup error = %v", err)
	}

	createErr := errors.New("create reset failed")
	repo := &errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), createResetErr: createErr}
	repo.storeUser(user)
	service = NewService(repo, time.Hour, &fakeMailer{})
	if err := service.RequestPasswordReset(context.Background(), PasswordResetRequestInput{Email: user.Email}); !errors.Is(err, createErr) {
		t.Fatalf("create reset error = %v, want %v", err, createErr)
	}

	mailerErr := errors.New("mail failed")
	repo = &errorAuthRepo{fakeAuthRepo: newFakeAuthRepo()}
	repo.storeUser(user)
	service = NewService(repo, time.Hour, &fakeMailer{err: mailerErr})
	if err := service.RequestPasswordReset(context.Background(), PasswordResetRequestInput{Email: user.Email}); err == nil || !strings.Contains(err.Error(), "send password reset code") {
		t.Fatalf("mailer error = %v", err)
	}

	activeErr := errors.New("active failed")
	repo = &errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), activeResetErr: activeErr}
	repo.storeUser(user)
	service = NewService(repo, time.Hour, nil)
	if err := service.ConfirmPasswordReset(context.Background(), PasswordResetConfirmInput{Email: user.Email, Code: "123456", Password: "new-password"}); err == nil || !strings.Contains(err.Error(), "get active password reset code") {
		t.Fatalf("active reset error = %v", err)
	}

	hash, err := hashPassword("123456")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	resetID := uuid.New()
	repo = &errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), updatePasswordErr: errors.New("update failed")}
	repo.storeUser(user)
	repo.resetCodesByUser[user.ID] = PasswordResetCode{ID: resetID, UserID: user.ID, CodeHash: hash}
	repo.resetCodesByID[resetID] = repo.resetCodesByUser[user.ID]
	service = NewService(repo, time.Hour, nil)
	if err := service.ConfirmPasswordReset(context.Background(), PasswordResetConfirmInput{Email: user.Email, Code: "123456", Password: "new-password"}); !errors.Is(err, repo.updatePasswordErr) {
		t.Fatalf("update password error = %v, want %v", err, repo.updatePasswordErr)
	}

	repo = &errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), markResetErr: errors.New("mark failed")}
	repo.storeUser(user)
	repo.resetCodesByUser[user.ID] = PasswordResetCode{ID: resetID, UserID: user.ID, CodeHash: hash}
	repo.resetCodesByID[resetID] = repo.resetCodesByUser[user.ID]
	service = NewService(repo, time.Hour, nil)
	if err := service.ConfirmPasswordReset(context.Background(), PasswordResetConfirmInput{Email: user.Email, Code: "123456", Password: "new-password"}); !errors.Is(err, repo.markResetErr) {
		t.Fatalf("mark reset error = %v, want %v", err, repo.markResetErr)
	}

	repo = &errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), deleteSessionsErr: errors.New("delete sessions failed")}
	repo.storeUser(user)
	repo.resetCodesByUser[user.ID] = PasswordResetCode{ID: resetID, UserID: user.ID, CodeHash: hash}
	repo.resetCodesByID[resetID] = repo.resetCodesByUser[user.ID]
	service = NewService(repo, time.Hour, nil)
	if err := service.ConfirmPasswordReset(context.Background(), PasswordResetConfirmInput{Email: user.Email, Code: "123456", Password: "new-password"}); !errors.Is(err, repo.deleteSessionsErr) {
		t.Fatalf("delete sessions error = %v, want %v", err, repo.deleteSessionsErr)
	}

	repo = &errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), incrementResetErr: errors.New("increment failed")}
	repo.storeUser(user)
	repo.resetCodesByUser[user.ID] = PasswordResetCode{ID: resetID, UserID: user.ID, CodeHash: hash}
	repo.resetCodesByID[resetID] = repo.resetCodesByUser[user.ID]
	service = NewService(repo, time.Hour, nil)
	if err := service.ConfirmPasswordReset(context.Background(), PasswordResetConfirmInput{Email: user.Email, Code: "000000", Password: "new-password"}); !errors.Is(err, ErrResetCodeInvalid) {
		t.Fatalf("invalid code error = %v, want %v", err, ErrResetCodeInvalid)
	}
}

func TestServiceOAuthErrorBranches(t *testing.T) {
	getOAuthErr := errors.New("oauth lookup failed")
	service := NewService(&errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), getOAuthErr: getOAuthErr}, time.Hour, nil)
	_, err := service.SignInWithOAuth(context.Background(), OAuthIdentity{
		Provider:       OAuthProviderGitHub,
		ProviderUserID: "github-1",
		Email:          "player@example.com",
		EmailVerified:  true,
	}, SessionMeta{})
	if err == nil || !strings.Contains(err.Error(), "get oauth account") {
		t.Fatalf("oauth lookup error = %v", err)
	}

	userReadErr := errors.New("user read failed")
	repo := &errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), getUserByIDErr: userReadErr}
	account := OAuthAccount{ID: uuid.New(), UserID: uuid.New(), Provider: OAuthProviderGitHub, ProviderUserID: "github-1"}
	repo.oauthByProviderID[string(OAuthProviderGitHub)+":github-1"] = account
	service = NewService(repo, time.Hour, nil)
	_, err = service.SignInWithOAuth(context.Background(), OAuthIdentity{
		Provider:       OAuthProviderGitHub,
		ProviderUserID: "github-1",
		Email:          "player@example.com",
		EmailVerified:  true,
	}, SessionMeta{})
	if !errors.Is(err, userReadErr) {
		t.Fatalf("existing oauth user error = %v, want %v", err, userReadErr)
	}

	repo = &errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), getEmailErr: errors.New("email failed")}
	service = NewService(repo, time.Hour, nil)
	_, err = service.SignInWithOAuth(context.Background(), OAuthIdentity{
		Provider:       OAuthProviderGitHub,
		ProviderUserID: "github-1",
		Email:          "player@example.com",
		EmailVerified:  true,
	}, SessionMeta{})
	if err == nil || !strings.Contains(err.Error(), "get user by email for oauth") {
		t.Fatalf("oauth email lookup error = %v", err)
	}

	repo = &errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), updateAvatarErr: errors.New("avatar failed")}
	service = NewService(repo, time.Hour, nil)
	_, err = service.SignInWithOAuth(context.Background(), OAuthIdentity{
		Provider:       OAuthProviderGitHub,
		ProviderUserID: "github-1",
		Email:          "player@example.com",
		EmailVerified:  true,
		Username:       "player",
		AvatarURL:      "https://avatar.example.com/u.png",
	}, SessionMeta{})
	if err == nil || !strings.Contains(err.Error(), "update oauth provider avatar") {
		t.Fatalf("avatar update error = %v", err)
	}

	createOAuthErr := errors.New("create oauth failed")
	repo = &errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), createOAuthErr: createOAuthErr}
	service = NewService(repo, time.Hour, nil)
	_, err = service.SignInWithOAuth(context.Background(), OAuthIdentity{
		Provider:       OAuthProviderGitHub,
		ProviderUserID: "github-1",
		Email:          "player@example.com",
		EmailVerified:  true,
		Username:       "player",
	}, SessionMeta{})
	if !errors.Is(err, createOAuthErr) {
		t.Fatalf("create oauth error = %v, want %v", err, createOAuthErr)
	}

	repo = &errorAuthRepo{fakeAuthRepo: newFakeAuthRepo(), createOAuthErr: ErrOAuthAccountLinked, getOAuthByUserProviderErr: errors.New("user provider failed")}
	repo.storeUser(User{ID: uuid.New(), Email: "player@example.com", Username: "player"})
	service = NewService(repo, time.Hour, nil)
	_, err = service.SignInWithOAuth(context.Background(), OAuthIdentity{
		Provider:       OAuthProviderGitHub,
		ProviderUserID: "github-1",
		Email:          "player@example.com",
		EmailVerified:  true,
	}, SessionMeta{})
	if err == nil || !strings.Contains(err.Error(), "get oauth account by user/provider") {
		t.Fatalf("resolve linked error = %v", err)
	}
}

type fakeMailer struct {
	to   string
	code string
	err  error
}

func (m *fakeMailer) SendPasswordResetCode(_ context.Context, to string, code string) error {
	m.to = to
	m.code = code
	return m.err
}

type fakeAuthRepo struct {
	usersByID                 map[uuid.UUID]User
	usersByEmail              map[string]User
	usersByUsername           map[string]User
	sessionsByHash            map[string]Session
	oauthByProviderID         map[string]OAuthAccount
	oauthByUserProvider       map[string]OAuthAccount
	resetCodesByID            map[uuid.UUID]PasswordResetCode
	resetCodesByUser          map[uuid.UUID]PasswordResetCode
	deleteExpiredCalls        int
	deleteSessionsByUserCalls int
}

func newFakeAuthRepo() *fakeAuthRepo {
	return &fakeAuthRepo{
		usersByID:           make(map[uuid.UUID]User),
		usersByEmail:        make(map[string]User),
		usersByUsername:     make(map[string]User),
		sessionsByHash:      make(map[string]Session),
		oauthByProviderID:   make(map[string]OAuthAccount),
		oauthByUserProvider: make(map[string]OAuthAccount),
		resetCodesByID:      make(map[uuid.UUID]PasswordResetCode),
		resetCodesByUser:    make(map[uuid.UUID]PasswordResetCode),
	}
}

func (r *fakeAuthRepo) storeUser(user User) User {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	r.usersByID[user.ID] = user
	r.usersByEmail[user.Email] = user
	r.usersByUsername[user.Username] = user
	return user
}

func (r *fakeAuthRepo) CreateUser(_ context.Context, user User) (User, error) {
	return r.storeUser(user), nil
}

func (r *fakeAuthRepo) GetUserByEmail(_ context.Context, email string) (User, error) {
	user, ok := r.usersByEmail[email]
	if !ok {
		return User{}, ErrNotFound
	}
	return user, nil
}

func (r *fakeAuthRepo) GetUserByUsername(_ context.Context, username string) (User, error) {
	user, ok := r.usersByUsername[username]
	if !ok {
		return User{}, ErrNotFound
	}
	return user, nil
}

func (r *fakeAuthRepo) GetUserByID(_ context.Context, userID uuid.UUID) (User, error) {
	user, ok := r.usersByID[userID]
	if !ok {
		return User{}, ErrNotFound
	}
	return user, nil
}

func (r *fakeAuthRepo) GetOAuthAccount(_ context.Context, provider OAuthProvider, providerUserID string) (OAuthAccount, error) {
	account, ok := r.oauthByProviderID[string(provider)+":"+providerUserID]
	if !ok {
		return OAuthAccount{}, ErrNotFound
	}
	return account, nil
}

func (r *fakeAuthRepo) GetOAuthAccountByUserProvider(_ context.Context, userID uuid.UUID, provider OAuthProvider) (OAuthAccount, error) {
	account, ok := r.oauthByUserProvider[userID.String()+":"+string(provider)]
	if !ok {
		return OAuthAccount{}, ErrNotFound
	}
	return account, nil
}

func (r *fakeAuthRepo) CreateOAuthAccount(_ context.Context, account OAuthAccount) (OAuthAccount, error) {
	if account.ID == uuid.Nil {
		account.ID = uuid.New()
	}
	providerKey := string(account.Provider) + ":" + account.ProviderUserID
	userKey := account.UserID.String() + ":" + string(account.Provider)
	if _, ok := r.oauthByProviderID[providerKey]; ok {
		return OAuthAccount{}, ErrOAuthAccountLinked
	}
	if _, ok := r.oauthByUserProvider[userKey]; ok {
		return OAuthAccount{}, ErrOAuthAccountLinked
	}
	r.oauthByProviderID[providerKey] = account
	r.oauthByUserProvider[userKey] = account
	return account, nil
}

func (r *fakeAuthRepo) UpdateProviderAvatar(_ context.Context, userID uuid.UUID, avatarURL string) error {
	user := r.usersByID[userID]
	user.ProviderAvatarURL = avatarURL
	user.AvatarSource = resolveAvatarSource(avatarURL)
	r.storeUser(user)
	return nil
}

func (r *fakeAuthRepo) CreateSession(_ context.Context, session Session) (Session, error) {
	r.sessionsByHash[session.TokenHash] = session
	return session, nil
}

func (r *fakeAuthRepo) GetSessionByTokenHash(_ context.Context, tokenHash string) (Session, error) {
	session, ok := r.sessionsByHash[tokenHash]
	if !ok {
		return Session{}, ErrNotFound
	}
	return session, nil
}

func (r *fakeAuthRepo) DeleteSessionByTokenHash(_ context.Context, tokenHash string) error {
	delete(r.sessionsByHash, tokenHash)
	return nil
}

func (r *fakeAuthRepo) DeleteExpiredSessions(context.Context, time.Time) error {
	r.deleteExpiredCalls++
	return nil
}

func (r *fakeAuthRepo) DeleteSessionsByUserID(_ context.Context, userID uuid.UUID) error {
	r.deleteSessionsByUserCalls++
	for hash, session := range r.sessionsByHash {
		if session.UserID == userID {
			delete(r.sessionsByHash, hash)
		}
	}
	return nil
}

func (r *fakeAuthRepo) CreatePasswordResetCode(_ context.Context, resetCode PasswordResetCode) (PasswordResetCode, error) {
	if resetCode.ID == uuid.Nil {
		resetCode.ID = uuid.New()
	}
	r.resetCodesByID[resetCode.ID] = resetCode
	r.resetCodesByUser[resetCode.UserID] = resetCode
	return resetCode, nil
}

func (r *fakeAuthRepo) GetActivePasswordResetCode(_ context.Context, userID uuid.UUID, _ time.Time) (PasswordResetCode, error) {
	resetCode, ok := r.resetCodesByUser[userID]
	if !ok || resetCode.UsedAt != nil {
		return PasswordResetCode{}, ErrNotFound
	}
	return resetCode, nil
}

func (r *fakeAuthRepo) IncrementPasswordResetAttempts(_ context.Context, codeID uuid.UUID) error {
	resetCode := r.resetCodesByID[codeID]
	resetCode.Attempts++
	r.resetCodesByID[codeID] = resetCode
	r.resetCodesByUser[resetCode.UserID] = resetCode
	return nil
}

func (r *fakeAuthRepo) MarkPasswordResetCodeUsed(_ context.Context, codeID uuid.UUID) error {
	resetCode := r.resetCodesByID[codeID]
	now := time.Now()
	resetCode.UsedAt = &now
	r.resetCodesByID[codeID] = resetCode
	r.resetCodesByUser[resetCode.UserID] = resetCode
	return nil
}

func (r *fakeAuthRepo) UpdateUserPasswordHash(_ context.Context, userID uuid.UUID, passwordHash string) error {
	user := r.usersByID[userID]
	user.PasswordHash = passwordHash
	r.storeUser(user)
	return nil
}

type errorAuthRepo struct {
	*fakeAuthRepo
	getEmailErr               error
	getUsernameErr            error
	getUserByIDErr            error
	getOAuthErr               error
	getOAuthByUserProviderErr error
	createOAuthErr            error
	createUserErr             error
	createSessionErr          error
	getSessionErr             error
	deleteSessionErr          error
	deleteExpiredErr          error
	deleteSessionsErr         error
	createResetErr            error
	activeResetErr            error
	incrementResetErr         error
	markResetErr              error
	updatePasswordErr         error
	updateAvatarErr           error
}

func (r *errorAuthRepo) CreateUser(ctx context.Context, user User) (User, error) {
	if r.createUserErr != nil {
		return User{}, r.createUserErr
	}
	return r.fakeAuthRepo.CreateUser(ctx, user)
}

func (r *errorAuthRepo) GetUserByEmail(ctx context.Context, email string) (User, error) {
	if r.getEmailErr != nil {
		return User{}, r.getEmailErr
	}
	return r.fakeAuthRepo.GetUserByEmail(ctx, email)
}

func (r *errorAuthRepo) GetUserByUsername(ctx context.Context, username string) (User, error) {
	if r.getUsernameErr != nil {
		return User{}, r.getUsernameErr
	}
	return r.fakeAuthRepo.GetUserByUsername(ctx, username)
}

func (r *errorAuthRepo) GetUserByID(ctx context.Context, userID uuid.UUID) (User, error) {
	if r.getUserByIDErr != nil {
		return User{}, r.getUserByIDErr
	}
	return r.fakeAuthRepo.GetUserByID(ctx, userID)
}

func (r *errorAuthRepo) GetOAuthAccount(ctx context.Context, provider OAuthProvider, providerUserID string) (OAuthAccount, error) {
	if r.getOAuthErr != nil {
		return OAuthAccount{}, r.getOAuthErr
	}
	return r.fakeAuthRepo.GetOAuthAccount(ctx, provider, providerUserID)
}

func (r *errorAuthRepo) GetOAuthAccountByUserProvider(ctx context.Context, userID uuid.UUID, provider OAuthProvider) (OAuthAccount, error) {
	if r.getOAuthByUserProviderErr != nil {
		return OAuthAccount{}, r.getOAuthByUserProviderErr
	}
	return r.fakeAuthRepo.GetOAuthAccountByUserProvider(ctx, userID, provider)
}

func (r *errorAuthRepo) CreateOAuthAccount(ctx context.Context, account OAuthAccount) (OAuthAccount, error) {
	if r.createOAuthErr != nil {
		return OAuthAccount{}, r.createOAuthErr
	}
	return r.fakeAuthRepo.CreateOAuthAccount(ctx, account)
}

func (r *errorAuthRepo) CreateSession(ctx context.Context, session Session) (Session, error) {
	if r.createSessionErr != nil {
		return Session{}, r.createSessionErr
	}
	return r.fakeAuthRepo.CreateSession(ctx, session)
}

func (r *errorAuthRepo) GetSessionByTokenHash(ctx context.Context, tokenHash string) (Session, error) {
	if r.getSessionErr != nil {
		return Session{}, r.getSessionErr
	}
	return r.fakeAuthRepo.GetSessionByTokenHash(ctx, tokenHash)
}

func (r *errorAuthRepo) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	if r.deleteSessionErr != nil {
		return r.deleteSessionErr
	}
	return r.fakeAuthRepo.DeleteSessionByTokenHash(ctx, tokenHash)
}

func (r *errorAuthRepo) DeleteExpiredSessions(ctx context.Context, now time.Time) error {
	if r.deleteExpiredErr != nil {
		return r.deleteExpiredErr
	}
	return r.fakeAuthRepo.DeleteExpiredSessions(ctx, now)
}

func (r *errorAuthRepo) DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error {
	if r.deleteSessionsErr != nil {
		return r.deleteSessionsErr
	}
	return r.fakeAuthRepo.DeleteSessionsByUserID(ctx, userID)
}

func (r *errorAuthRepo) CreatePasswordResetCode(ctx context.Context, resetCode PasswordResetCode) (PasswordResetCode, error) {
	if r.createResetErr != nil {
		return PasswordResetCode{}, r.createResetErr
	}
	return r.fakeAuthRepo.CreatePasswordResetCode(ctx, resetCode)
}

func (r *errorAuthRepo) GetActivePasswordResetCode(ctx context.Context, userID uuid.UUID, now time.Time) (PasswordResetCode, error) {
	if r.activeResetErr != nil {
		return PasswordResetCode{}, r.activeResetErr
	}
	return r.fakeAuthRepo.GetActivePasswordResetCode(ctx, userID, now)
}

func (r *errorAuthRepo) IncrementPasswordResetAttempts(ctx context.Context, codeID uuid.UUID) error {
	if r.incrementResetErr != nil {
		return r.incrementResetErr
	}
	return r.fakeAuthRepo.IncrementPasswordResetAttempts(ctx, codeID)
}

func (r *errorAuthRepo) MarkPasswordResetCodeUsed(ctx context.Context, codeID uuid.UUID) error {
	if r.markResetErr != nil {
		return r.markResetErr
	}
	return r.fakeAuthRepo.MarkPasswordResetCodeUsed(ctx, codeID)
}

func (r *errorAuthRepo) UpdateUserPasswordHash(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	if r.updatePasswordErr != nil {
		return r.updatePasswordErr
	}
	return r.fakeAuthRepo.UpdateUserPasswordHash(ctx, userID, passwordHash)
}

func (r *errorAuthRepo) UpdateProviderAvatar(ctx context.Context, userID uuid.UUID, avatarURL string) error {
	if r.updateAvatarErr != nil {
		return r.updateAvatarErr
	}
	return r.fakeAuthRepo.UpdateProviderAvatar(ctx, userID, avatarURL)
}
