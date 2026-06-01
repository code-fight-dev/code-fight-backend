package auth

import (
	"errors"
	"testing"
	"time"

	"code-fight-backend/internal/testsupport"

	"github.com/google/uuid"
)

func TestRepositoryIntegrationUserSessionOAuthAndPasswordReset(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	repo := NewRepository(db)

	suffix := uuid.NewString()
	user, err := repo.CreateUser(ctx, User{
		ID:           uuid.New(),
		Email:        "player-" + suffix + "@example.com",
		Username:     "player_" + suffix[:8],
		DisplayName:  "Player " + suffix[:8],
		PasswordHash: "hash",
		AvatarSource: AvatarSourceNone,
	})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "delete from users where id = $1", user.ID)
	})

	byEmail, err := repo.GetUserByEmail(ctx, user.Email)
	if err != nil || byEmail.ID != user.ID {
		t.Fatalf("GetUserByEmail = %#v, %v", byEmail, err)
	}
	byUsername, err := repo.GetUserByUsername(ctx, user.Username)
	if err != nil || byUsername.ID != user.ID {
		t.Fatalf("GetUserByUsername = %#v, %v", byUsername, err)
	}
	byID, err := repo.GetUserByID(ctx, user.ID)
	if err != nil || byID.Email != user.Email {
		t.Fatalf("GetUserByID = %#v, %v", byID, err)
	}

	_, err = repo.CreateUser(ctx, User{
		ID:           uuid.New(),
		Email:        user.Email,
		Username:     "other_" + suffix[:8],
		DisplayName:  "Other",
		PasswordHash: "hash",
		AvatarSource: AvatarSourceNone,
	})
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("duplicate email error = %v, want ErrEmailTaken", err)
	}

	account, err := repo.CreateOAuthAccount(ctx, OAuthAccount{
		ID:             uuid.New(),
		UserID:         user.ID,
		Provider:       OAuthProviderGitHub,
		ProviderUserID: "github-" + suffix,
		ProviderEmail:  user.Email,
	})
	if err != nil {
		t.Fatalf("CreateOAuthAccount returned error: %v", err)
	}
	loadedAccount, err := repo.GetOAuthAccount(ctx, OAuthProviderGitHub, account.ProviderUserID)
	if err != nil || loadedAccount.UserID != user.ID {
		t.Fatalf("GetOAuthAccount = %#v, %v", loadedAccount, err)
	}
	loadedByUser, err := repo.GetOAuthAccountByUserProvider(ctx, user.ID, OAuthProviderGitHub)
	if err != nil || loadedByUser.ProviderUserID != account.ProviderUserID {
		t.Fatalf("GetOAuthAccountByUserProvider = %#v, %v", loadedByUser, err)
	}

	if err := repo.UpdateProviderAvatar(ctx, user.ID, "https://avatar.example.com/u.png"); err != nil {
		t.Fatalf("UpdateProviderAvatar returned error: %v", err)
	}

	session, err := repo.CreateSession(ctx, Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: "token-" + suffix,
		UserAgent: "go-test",
		IPAddress: "127.0.0.1",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	loadedSession, err := repo.GetSessionByTokenHash(ctx, session.TokenHash)
	if err != nil || loadedSession.UserID != user.ID {
		t.Fatalf("GetSessionByTokenHash = %#v, %v", loadedSession, err)
	}
	if err := repo.DeleteSessionByTokenHash(ctx, session.TokenHash); err != nil {
		t.Fatalf("DeleteSessionByTokenHash returned error: %v", err)
	}
	if _, err := repo.GetSessionByTokenHash(ctx, session.TokenHash); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted session error = %v, want ErrNotFound", err)
	}

	resetCode, err := repo.CreatePasswordResetCode(ctx, PasswordResetCode{
		ID:        uuid.New(),
		UserID:    user.ID,
		CodeHash:  "code-hash",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreatePasswordResetCode returned error: %v", err)
	}
	active, err := repo.GetActivePasswordResetCode(ctx, user.ID, time.Now())
	if err != nil || active.ID != resetCode.ID {
		t.Fatalf("GetActivePasswordResetCode = %#v, %v", active, err)
	}
	if err := repo.IncrementPasswordResetAttempts(ctx, resetCode.ID); err != nil {
		t.Fatalf("IncrementPasswordResetAttempts returned error: %v", err)
	}
	if err := repo.MarkPasswordResetCodeUsed(ctx, resetCode.ID); err != nil {
		t.Fatalf("MarkPasswordResetCodeUsed returned error: %v", err)
	}
	if err := repo.UpdateUserPasswordHash(ctx, user.ID, "new-hash"); err != nil {
		t.Fatalf("UpdateUserPasswordHash returned error: %v", err)
	}
	if err := repo.DeleteSessionsByUserID(ctx, user.ID); err != nil {
		t.Fatalf("DeleteSessionsByUserID returned error: %v", err)
	}
	if err := repo.DeleteExpiredSessions(ctx, time.Now()); err != nil {
		t.Fatalf("DeleteExpiredSessions returned error: %v", err)
	}
}
