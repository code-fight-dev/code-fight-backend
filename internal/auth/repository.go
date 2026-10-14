package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultGlobalRating      = 1000
	oauthAccountSelectCols   = "id, user_id, provider, provider_user_id, provider_email, created_at"
	passwordResetMaxAttempts = 5
	userSelectCols           = "id, email, username, display_name, password_hash, created_at, updated_at, role, role_version"
)

type Repository struct {
	db authDB
}

type authDB interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, user User) (User, error) {
	query := `
		with inserted_user as (
			insert into users (
				id,
				email,
				username,
				display_name,
				password_hash,
				avatar_url,
				provider_avatar_url,
				avatar_source
			)
			values ($1, $2, $3, $4, $5, $6, $7, $8)
			returning ` + userSelectCols + `
		),
		inserted_rating as (
			insert into user_ratings (user_id, mode, rating, rated_games, wins, losses, draws)
			select id, 'global', $9, 0, 0, 0, 0
			from inserted_user
			on conflict (user_id, mode) do nothing
		)
		select ` + userSelectCols + `
		from inserted_user
	`

	if err := r.db.QueryRow(ctx, query,
		user.ID,
		user.Email,
		user.Username,
		user.DisplayName,
		user.PasswordHash,
		user.AvatarURL,
		user.ProviderAvatarURL,
		user.AvatarSource,
		defaultGlobalRating,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.DisplayName,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Role,
		&user.RoleVersion,
	); err != nil {
		if isUniqueViolation(err, "users_email_lower_key") {
			return User{}, ErrEmailTaken
		}
		if isUniqueViolation(err, "users_username_lower_key") {
			return User{}, ErrUsernameTaken
		}

		return User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	query := `
		select ` + userSelectCols + `
		from users
		where lower(email) = lower($1)
	`

	return scanUser(r.db.QueryRow(ctx, query, email))
}

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (User, error) {
	query := `
		select ` + userSelectCols + `
		from users
		where lower(username) = lower($1)
	`

	return scanUser(r.db.QueryRow(ctx, query, username))
}

func (r *Repository) GetUserByID(ctx context.Context, userID uuid.UUID) (User, error) {
	query := `
		select ` + userSelectCols + `
		from users
		where id = $1
	`

	return scanUser(r.db.QueryRow(ctx, query, userID))
}

func (r *Repository) GetOAuthAccount(ctx context.Context, provider OAuthProvider, providerUserID string) (OAuthAccount, error) {
	query := `
		select ` + oauthAccountSelectCols + `
		from oauth_accounts
		where provider = $1 and provider_user_id = $2
	`

	return scanOAuthAccount(r.db.QueryRow(ctx, query, provider, providerUserID))
}

func (r *Repository) GetOAuthAccountByUserProvider(ctx context.Context, userID uuid.UUID, provider OAuthProvider) (OAuthAccount, error) {
	query := `
		select ` + oauthAccountSelectCols + `
		from oauth_accounts
		where user_id = $1 and provider = $2
	`

	return scanOAuthAccount(r.db.QueryRow(ctx, query, userID, provider))
}

func (r *Repository) CreateOAuthAccount(ctx context.Context, account OAuthAccount) (OAuthAccount, error) {
	query := `
		insert into oauth_accounts (id, user_id, provider, provider_user_id, provider_email)
		values ($1, $2, $3, $4, $5)
		returning id, user_id, provider, provider_user_id, provider_email, created_at
	`

	if err := r.db.QueryRow(ctx, query,
		account.ID,
		account.UserID,
		account.Provider,
		account.ProviderUserID,
		account.ProviderEmail,
	).Scan(
		&account.ID,
		&account.UserID,
		&account.Provider,
		&account.ProviderUserID,
		&account.ProviderEmail,
		&account.CreatedAt,
	); err != nil {
		if isUniqueViolation(err, "oauth_accounts_provider_external_key") || isUniqueViolation(err, "oauth_accounts_provider_user_key") {
			return OAuthAccount{}, ErrOAuthAccountLinked
		}

		return OAuthAccount{}, fmt.Errorf("create oauth account: %w", err)
	}

	return account, nil
}

func (r *Repository) UpdateProviderAvatar(ctx context.Context, userID uuid.UUID, avatarURL string) error {
	query := `
		update users
		set provider_avatar_url = $2,
			avatar_source = case
				when avatar_source = '` + AvatarSourceCustom + `' then avatar_source
				when $2 <> '' then '` + AvatarSourceProvider + `'
				else '` + AvatarSourceNone + `'
			end,
			updated_at = now()
		where id = $1
	`

	if _, err := r.db.Exec(ctx, query, userID, avatarURL); err != nil {
		return fmt.Errorf("update provider avatar: %w", err)
	}

	return nil
}

func (r *Repository) CreateSession(ctx context.Context, session Session) (Session, error) {
	query := `
		insert into sessions (id, user_id, token_hash, user_agent, ip_address, expires_at)
		values ($1, $2, $3, $4, $5, $6)
		returning id, user_id, token_hash, user_agent, ip_address, expires_at, created_at
	`

	if err := r.db.QueryRow(ctx, query,
		session.ID,
		session.UserID,
		session.TokenHash,
		session.UserAgent,
		session.IPAddress,
		session.ExpiresAt,
	).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.UserAgent,
		&session.IPAddress,
		&session.ExpiresAt,
		&session.CreatedAt,
	); err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}

	return session, nil
}

func (r *Repository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (Session, error) {
	query := `
		select id, user_id, token_hash, user_agent, ip_address, expires_at, created_at
		from sessions
		where token_hash = $1
	`

	row := r.db.QueryRow(ctx, query, tokenHash)
	var session Session
	if err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.UserAgent,
		&session.IPAddress,
		&session.ExpiresAt,
		&session.CreatedAt,
	); err != nil {
		return Session{}, normalizeNotFound(err)
	}

	return session, nil
}

func (r *Repository) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	if _, err := r.db.Exec(ctx, `
		delete from sessions where token_hash = $1
	`, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

func (r *Repository) DeleteExpiredSessions(ctx context.Context, before time.Time) error {
	if _, err := r.db.Exec(ctx, `
		delete from sessions where expires_at <= $1
	`, before); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}

	return nil
}

func (r *Repository) DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `
		delete from sessions where user_id = $1
	`, userID); err != nil {
		return fmt.Errorf("delete user sessions: %w", err)
	}

	return nil
}

func (r *Repository) CreatePasswordResetCode(ctx context.Context, resetCode PasswordResetCode) (PasswordResetCode, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return PasswordResetCode{}, fmt.Errorf("begin password reset code creation: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		update password_reset_codes
		set used_at = now()
		where user_id = $1 and used_at is null
	`, resetCode.UserID); err != nil {
		return PasswordResetCode{}, fmt.Errorf("expire previous password reset codes: %w", err)
	}

	query := `
		insert into password_reset_codes (id, user_id, code_hash, expires_at)
		values ($1, $2, $3, $4)
		returning id, user_id, code_hash, expires_at, used_at, attempts, created_at
	`

	if err := tx.QueryRow(ctx, query,
		resetCode.ID,
		resetCode.UserID,
		resetCode.CodeHash,
		resetCode.ExpiresAt,
	).Scan(
		&resetCode.ID,
		&resetCode.UserID,
		&resetCode.CodeHash,
		&resetCode.ExpiresAt,
		&resetCode.UsedAt,
		&resetCode.Attempts,
		&resetCode.CreatedAt,
	); err != nil {
		return PasswordResetCode{}, fmt.Errorf("create password reset code: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return PasswordResetCode{}, fmt.Errorf("commit password reset code creation: %w", err)
	}

	return resetCode, nil
}

func (r *Repository) GetActivePasswordResetCode(ctx context.Context, userID uuid.UUID, now time.Time) (PasswordResetCode, error) {
	query := `
		select id, user_id, code_hash, expires_at, used_at, attempts, created_at
		from password_reset_codes
		where user_id = $1 and used_at is null and expires_at > $2
		order by created_at desc
		limit 1
	`

	var resetCode PasswordResetCode
	if err := r.db.QueryRow(ctx, query, userID, now).Scan(
		&resetCode.ID,
		&resetCode.UserID,
		&resetCode.CodeHash,
		&resetCode.ExpiresAt,
		&resetCode.UsedAt,
		&resetCode.Attempts,
		&resetCode.CreatedAt,
	); err != nil {
		return PasswordResetCode{}, normalizeNotFound(err)
	}

	return resetCode, nil
}

func (r *Repository) IncrementPasswordResetAttempts(ctx context.Context, codeID uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `
		update password_reset_codes
		set attempts = attempts + 1
		where id = $1
	`, codeID); err != nil {
		return fmt.Errorf("increment password reset attempts: %w", err)
	}

	return nil
}

func (r *Repository) MarkPasswordResetCodeUsed(ctx context.Context, codeID uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `
		update password_reset_codes
		set used_at = now()
		where id = $1
	`, codeID); err != nil {
		return fmt.Errorf("mark password reset code used: %w", err)
	}

	return nil
}

func (r *Repository) UpdateUserPasswordHash(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	if _, err := r.db.Exec(ctx, `
		update users
		set password_hash = $2,
			updated_at = now()
		where id = $1
	`, userID, passwordHash); err != nil {
		return fmt.Errorf("update user password hash: %w", err)
	}

	return nil
}

func scanUser(row pgx.Row) (User, error) {
	var user User
	if err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.DisplayName,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Role,
		&user.RoleVersion,
	); err != nil {
		return User{}, normalizeNotFound(err)
	}

	return user, nil
}

func scanOAuthAccount(row pgx.Row) (OAuthAccount, error) {
	var account OAuthAccount
	if err := row.Scan(
		&account.ID,
		&account.UserID,
		&account.Provider,
		&account.ProviderUserID,
		&account.ProviderEmail,
		&account.CreatedAt,
	); err != nil {
		return OAuthAccount{}, normalizeNotFound(err)
	}

	return account, nil
}

func normalizeNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	return err
}

func isUniqueViolation(err error, constraint string) bool {
	var pgError *pgconn.PgError
	if !errors.As(err, &pgError) {
		return false
	}

	return pgError.Code == "23505" && pgError.ConstraintName == constraint
}
