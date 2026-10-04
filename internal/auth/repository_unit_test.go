package auth

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestNewRepository(t *testing.T) {
	if repo := NewRepository(nil); repo == nil {
		t.Fatal("NewRepository returned nil")
	}
}

func TestRepositoryMethodsWithFakeDB(t *testing.T) {
	now := time.Now().UTC()
	user := User{
		ID:           uuid.New(),
		Email:        "player@example.com",
		Username:     "player",
		DisplayName:  "Player",
		PasswordHash: "hash",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	account := OAuthAccount{
		ID:             uuid.New(),
		UserID:         user.ID,
		Provider:       OAuthProviderGitHub,
		ProviderUserID: "gh-1",
		ProviderEmail:  user.Email,
		CreatedAt:      now,
	}
	session := Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: "hash-token",
		UserAgent: "browser",
		IPAddress: "127.0.0.1",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}
	resetCode := PasswordResetCode{
		ID:        uuid.New(),
		UserID:    user.ID,
		CodeHash:  "code-hash",
		ExpiresAt: now.Add(time.Hour),
		Attempts:  1,
		CreatedAt: now,
	}

	db := &fakeAuthRepositoryDB{rows: []pgx.Row{
		fakePGXRow{values: userRowValues(user)},
		fakePGXRow{values: userRowValues(user)},
		fakePGXRow{values: userRowValues(user)},
		fakePGXRow{values: userRowValues(user)},
		fakePGXRow{values: oauthAccountRowValues(account)},
		fakePGXRow{values: oauthAccountRowValues(account)},
		fakePGXRow{values: oauthAccountRowValues(account)},
		fakePGXRow{values: sessionRowValues(session)},
		fakePGXRow{values: sessionRowValues(session)},
		fakePGXRow{values: resetCodeRowValues(resetCode)},
		fakePGXRow{values: resetCodeRowValues(resetCode)},
	}}
	repo := &Repository{db: db}

	if got, err := repo.CreateUser(context.Background(), user); err != nil || got.ID != user.ID {
		t.Fatalf("CreateUser = %#v, %v", got, err)
	}
	if got, err := repo.GetUserByEmail(context.Background(), user.Email); err != nil || got.Email != user.Email {
		t.Fatalf("GetUserByEmail = %#v, %v", got, err)
	}
	if got, err := repo.GetUserByUsername(context.Background(), user.Username); err != nil || got.Username != user.Username {
		t.Fatalf("GetUserByUsername = %#v, %v", got, err)
	}
	if got, err := repo.GetUserByID(context.Background(), user.ID); err != nil || got.ID != user.ID {
		t.Fatalf("GetUserByID = %#v, %v", got, err)
	}
	if got, err := repo.GetOAuthAccount(context.Background(), OAuthProviderGitHub, account.ProviderUserID); err != nil || got.ID != account.ID {
		t.Fatalf("GetOAuthAccount = %#v, %v", got, err)
	}
	if got, err := repo.GetOAuthAccountByUserProvider(context.Background(), user.ID, OAuthProviderGitHub); err != nil || got.UserID != user.ID {
		t.Fatalf("GetOAuthAccountByUserProvider = %#v, %v", got, err)
	}
	if got, err := repo.CreateOAuthAccount(context.Background(), account); err != nil || got.ProviderUserID != account.ProviderUserID {
		t.Fatalf("CreateOAuthAccount = %#v, %v", got, err)
	}
	if err := repo.UpdateProviderAvatar(context.Background(), user.ID, "https://avatar"); err != nil {
		t.Fatalf("UpdateProviderAvatar returned error: %v", err)
	}
	if got, err := repo.CreateSession(context.Background(), session); err != nil || got.ID != session.ID {
		t.Fatalf("CreateSession = %#v, %v", got, err)
	}
	if got, err := repo.GetSessionByTokenHash(context.Background(), session.TokenHash); err != nil || got.UserID != user.ID {
		t.Fatalf("GetSessionByTokenHash = %#v, %v", got, err)
	}
	if err := repo.DeleteSessionByTokenHash(context.Background(), session.TokenHash); err != nil {
		t.Fatalf("DeleteSessionByTokenHash returned error: %v", err)
	}
	if err := repo.DeleteExpiredSessions(context.Background(), now); err != nil {
		t.Fatalf("DeleteExpiredSessions returned error: %v", err)
	}
	if err := repo.DeleteSessionsByUserID(context.Background(), user.ID); err != nil {
		t.Fatalf("DeleteSessionsByUserID returned error: %v", err)
	}
	if got, err := repo.CreatePasswordResetCode(context.Background(), resetCode); err != nil || got.ID != resetCode.ID {
		t.Fatalf("CreatePasswordResetCode = %#v, %v", got, err)
	}
	if got, err := repo.GetActivePasswordResetCode(context.Background(), user.ID, now); err != nil || got.ID != resetCode.ID {
		t.Fatalf("GetActivePasswordResetCode = %#v, %v", got, err)
	}
	if err := repo.IncrementPasswordResetAttempts(context.Background(), resetCode.ID); err != nil {
		t.Fatalf("IncrementPasswordResetAttempts returned error: %v", err)
	}
	if err := repo.MarkPasswordResetCodeUsed(context.Background(), resetCode.ID); err != nil {
		t.Fatalf("MarkPasswordResetCodeUsed returned error: %v", err)
	}
	if err := repo.UpdateUserPasswordHash(context.Background(), user.ID, "new-hash"); err != nil {
		t.Fatalf("UpdateUserPasswordHash returned error: %v", err)
	}
}

func TestRepositoryMapsNotFoundAndUniqueViolations(t *testing.T) {
	if err := normalizeNotFound(pgx.ErrNoRows); !errors.Is(err, ErrNotFound) {
		t.Fatalf("normalizeNotFound = %v, want ErrNotFound", err)
	}
	unique := &pgconn.PgError{Code: "23505", ConstraintName: "users_email_lower_key"}
	if !isUniqueViolation(unique, "users_email_lower_key") {
		t.Fatal("isUniqueViolation rejected matching unique violation")
	}
	if isUniqueViolation(unique, "users_username_lower_key") {
		t.Fatal("isUniqueViolation accepted different constraint")
	}

	repo := &Repository{db: &fakeAuthRepositoryDB{rows: []pgx.Row{
		fakePGXRow{err: unique},
		fakePGXRow{err: &pgconn.PgError{Code: "23505", ConstraintName: "users_username_lower_key"}},
	}}}
	if _, err := repo.CreateUser(context.Background(), User{}); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("email unique error = %v, want ErrEmailTaken", err)
	}
	if _, err := repo.CreateUser(context.Background(), User{}); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("username unique error = %v, want ErrUsernameTaken", err)
	}

	repo = &Repository{db: &fakeAuthRepositoryDB{rows: []pgx.Row{
		fakePGXRow{err: &pgconn.PgError{Code: "23505", ConstraintName: "oauth_accounts_provider_external_key"}},
		fakePGXRow{err: &pgconn.PgError{Code: "23505", ConstraintName: "oauth_accounts_provider_user_key"}},
	}}}
	if _, err := repo.CreateOAuthAccount(context.Background(), OAuthAccount{}); !errors.Is(err, ErrOAuthAccountLinked) {
		t.Fatalf("oauth provider unique error = %v, want ErrOAuthAccountLinked", err)
	}
	if _, err := repo.CreateOAuthAccount(context.Background(), OAuthAccount{}); !errors.Is(err, ErrOAuthAccountLinked) {
		t.Fatalf("oauth user unique error = %v, want ErrOAuthAccountLinked", err)
	}
}

func TestRepositoryExecAndTransactionErrors(t *testing.T) {
	dbErr := errors.New("db failed")
	userID := uuid.New()
	codeID := uuid.New()

	tests := []struct {
		name string
		call func(*Repository) error
		repo *Repository
		want string
	}{
		{
			name: "update avatar",
			repo: &Repository{db: &fakeAuthRepositoryDB{execErr: dbErr}},
			call: func(repo *Repository) error {
				return repo.UpdateProviderAvatar(context.Background(), userID, "avatar")
			},
			want: "update provider avatar",
		},
		{
			name: "delete session",
			repo: &Repository{db: &fakeAuthRepositoryDB{execErr: dbErr}},
			call: func(repo *Repository) error {
				return repo.DeleteSessionByTokenHash(context.Background(), "hash")
			},
			want: "delete session",
		},
		{
			name: "delete expired",
			repo: &Repository{db: &fakeAuthRepositoryDB{execErr: dbErr}},
			call: func(repo *Repository) error {
				return repo.DeleteExpiredSessions(context.Background(), time.Now())
			},
			want: "delete expired sessions",
		},
		{
			name: "delete user sessions",
			repo: &Repository{db: &fakeAuthRepositoryDB{execErr: dbErr}},
			call: func(repo *Repository) error {
				return repo.DeleteSessionsByUserID(context.Background(), userID)
			},
			want: "delete user sessions",
		},
		{
			name: "increment reset",
			repo: &Repository{db: &fakeAuthRepositoryDB{execErr: dbErr}},
			call: func(repo *Repository) error {
				return repo.IncrementPasswordResetAttempts(context.Background(), codeID)
			},
			want: "increment password reset attempts",
		},
		{
			name: "mark reset used",
			repo: &Repository{db: &fakeAuthRepositoryDB{execErr: dbErr}},
			call: func(repo *Repository) error {
				return repo.MarkPasswordResetCodeUsed(context.Background(), codeID)
			},
			want: "mark password reset code used",
		},
		{
			name: "update password",
			repo: &Repository{db: &fakeAuthRepositoryDB{execErr: dbErr}},
			call: func(repo *Repository) error {
				return repo.UpdateUserPasswordHash(context.Background(), userID, "hash")
			},
			want: "update user password hash",
		},
		{
			name: "reset begin",
			repo: &Repository{db: &fakeAuthRepositoryDB{beginErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.CreatePasswordResetCode(context.Background(), PasswordResetCode{UserID: userID})
				return err
			},
			want: "begin password reset code creation",
		},
		{
			name: "reset expire",
			repo: &Repository{db: &fakeAuthRepositoryDB{tx: &fakeAuthTx{execErr: dbErr}}},
			call: func(repo *Repository) error {
				_, err := repo.CreatePasswordResetCode(context.Background(), PasswordResetCode{UserID: userID})
				return err
			},
			want: "expire previous password reset codes",
		},
		{
			name: "reset create",
			repo: &Repository{db: &fakeAuthRepositoryDB{tx: &fakeAuthTx{rowErr: dbErr}}},
			call: func(repo *Repository) error {
				_, err := repo.CreatePasswordResetCode(context.Background(), PasswordResetCode{UserID: userID})
				return err
			},
			want: "create password reset code",
		},
		{
			name: "reset commit",
			repo: &Repository{db: &fakeAuthRepositoryDB{rows: []pgx.Row{fakePGXRow{values: resetCodeRowValues(PasswordResetCode{ID: codeID, UserID: userID})}}, tx: &fakeAuthTx{commitErr: dbErr}}},
			call: func(repo *Repository) error {
				_, err := repo.CreatePasswordResetCode(context.Background(), PasswordResetCode{ID: codeID, UserID: userID})
				return err
			},
			want: "commit password reset code creation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(tt.repo)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

type fakeAuthRepositoryDB struct {
	rows     []pgx.Row
	execErr  error
	beginErr error
	tx       *fakeAuthTx
}

func (db *fakeAuthRepositoryDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	if db.execErr != nil {
		return pgconn.CommandTag{}, db.execErr
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (db *fakeAuthRepositoryDB) Begin(context.Context) (pgx.Tx, error) {
	if db.beginErr != nil {
		return nil, db.beginErr
	}
	if db.tx != nil {
		db.tx.db = db
		return db.tx, nil
	}
	return &fakeAuthTx{db: db}, nil
}

func (db *fakeAuthRepositoryDB) QueryRow(context.Context, string, ...any) pgx.Row {
	if len(db.rows) == 0 {
		return fakePGXRow{err: pgx.ErrNoRows}
	}
	row := db.rows[0]
	db.rows = db.rows[1:]
	return row
}

type fakeAuthTx struct {
	db        *fakeAuthRepositoryDB
	execErr   error
	commitErr error
	rowErr    error
}

func (tx *fakeAuthTx) Begin(context.Context) (pgx.Tx, error) { return tx, nil }
func (tx *fakeAuthTx) Commit(context.Context) error          { return tx.commitErr }
func (tx *fakeAuthTx) Rollback(context.Context) error        { return nil }
func (tx *fakeAuthTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("unused")
}
func (tx *fakeAuthTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (tx *fakeAuthTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *fakeAuthTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("unused")
}
func (tx *fakeAuthTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	if tx.execErr != nil {
		return pgconn.CommandTag{}, tx.execErr
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}
func (tx *fakeAuthTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unused")
}
func (tx *fakeAuthTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if tx.rowErr != nil {
		return fakePGXRow{err: tx.rowErr}
	}
	return tx.db.QueryRow(ctx, sql, args...)
}
func (tx *fakeAuthTx) Conn() *pgx.Conn { return nil }

type fakePGXRow struct {
	values []any
	err    error
}

func (r fakePGXRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("fake row destination count mismatch")
	}
	for i := range dest {
		target := reflect.ValueOf(dest[i])
		if target.Kind() != reflect.Pointer || target.IsNil() {
			return errors.New("fake row destination must be pointer")
		}
		value := reflect.ValueOf(r.values[i])
		elem := target.Elem()
		if !value.IsValid() {
			elem.Set(reflect.Zero(elem.Type()))
			continue
		}
		if value.Type().AssignableTo(elem.Type()) {
			elem.Set(value)
			continue
		}
		if value.Type().ConvertibleTo(elem.Type()) {
			elem.Set(value.Convert(elem.Type()))
			continue
		}
		return errors.New("fake row value type mismatch")
	}
	return nil
}

func userRowValues(user User) []any {
	return []any{user.ID, user.Email, user.Username, user.DisplayName, user.PasswordHash, user.CreatedAt, user.UpdatedAt, user.Role, user.RoleVersion}
}

func oauthAccountRowValues(account OAuthAccount) []any {
	return []any{account.ID, account.UserID, account.Provider, account.ProviderUserID, account.ProviderEmail, account.CreatedAt}
}

func sessionRowValues(session Session) []any {
	return []any{session.ID, session.UserID, session.TokenHash, session.UserAgent, session.IPAddress, session.ExpiresAt, session.CreatedAt}
}

func resetCodeRowValues(code PasswordResetCode) []any {
	return []any{code.ID, code.UserID, code.CodeHash, code.ExpiresAt, code.UsedAt, code.Attempts, code.CreatedAt}
}
