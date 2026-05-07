package leaderboard

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, mode string, limit int, offset int) ([]Entry, int, error) {
	rows, err := r.db.Query(ctx, leaderboardQuery(`
		select *
		from ranked
		order by rank asc
		limit $2 offset $3
	`), mode, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list leaderboard: %w", err)
	}
	defer rows.Close()

	items := make([]Entry, 0, limit)
	total := 0

	for rows.Next() {
		entry, rowTotal, err := scanEntry(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, entry)
		total = rowTotal
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate leaderboard: %w", err)
	}

	if len(items) == 0 {
		if err := r.db.QueryRow(ctx, `
			select count(*)
			from users u
			left join user_ratings ur
				on ur.user_id = u.id
				and ur.mode = $1
		`, mode).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("count leaderboard: %w", err)
		}
	}

	return items, total, nil
}

func (r *Repository) GetViewerRank(ctx context.Context, mode string, userID uuid.UUID) (*Entry, error) {
	row := r.db.QueryRow(ctx, leaderboardQuery(`
		select *
		from ranked
		where user_id = $2
	`), mode, userID)

	entry, _, err := scanEntry(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &entry, nil
}

func leaderboardQuery(selectClause string) string {
	return `
		with ranked as (
			select
				row_number() over (
					order by coalesce(ur.rating, 1000) desc,
						coalesce(ur.rated_games, 0) desc,
						lower(u.username) asc,
						u.id asc
				)::int as rank,
				count(*) over ()::int as total,
				u.id as user_id,
				u.username,
				u.display_name,
				case
					when u.avatar_source = 'custom' and u.avatar_url <> '' then u.avatar_url
					when u.avatar_source = 'provider' and u.provider_avatar_url <> '' then u.provider_avatar_url
					when u.avatar_url <> '' then u.avatar_url
					else u.provider_avatar_url
				end as avatar_url,
				u.country,
				u.country_code,
				coalesce(ur.rating, 1000)::int as rating,
				coalesce(ur.rated_games, 0)::int as rated_games,
				coalesce(ur.wins, 0)::int as wins,
				coalesce(ur.losses, 0)::int as losses,
				coalesce(ur.draws, 0)::int as draws
			from users u
			left join user_ratings ur
				on ur.user_id = u.id
				and ur.mode = $1
		)
	` + selectClause
}

func scanEntry(row pgx.Row) (Entry, int, error) {
	var entry Entry
	total := 0

	if err := row.Scan(
		&entry.Rank,
		&total,
		&entry.UserID,
		&entry.Username,
		&entry.DisplayName,
		&entry.AvatarURL,
		&entry.Country,
		&entry.CountryCode,
		&entry.Rating,
		&entry.RatedGames,
		&entry.Wins,
		&entry.Losses,
		&entry.Draws,
	); err != nil {
		return Entry{}, 0, err
	}

	if strings.TrimSpace(entry.DisplayName) == "" {
		entry.DisplayName = entry.Username
	}
	if entry.RatedGames > 0 {
		entry.WinRate = float64(entry.Wins) / float64(entry.RatedGames)
	}

	return entry, total, nil
}
