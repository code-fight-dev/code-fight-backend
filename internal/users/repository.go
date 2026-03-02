package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetProfileByUsername(ctx context.Context, username string) (Profile, error) {
	const query = `
		select id, username, created_at
		from users
		where lower(username) = lower($1)
	`

	row := r.db.QueryRow(ctx, query, username)
	var profile Profile
	if err := row.Scan(
		&profile.ID,
		&profile.Username,
		&profile.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Profile{}, ErrNotFound
		}

		return Profile{}, fmt.Errorf("get profile by username: %w", err)
	}

	return profile, nil
}
