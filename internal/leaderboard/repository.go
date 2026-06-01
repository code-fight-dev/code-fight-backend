package leaderboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db leaderboardDB
}

type leaderboardDB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
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

func (r *Repository) GetStats(ctx context.Context, mode string) (Stats, error) {
	var stats Stats
	if err := r.db.QueryRow(ctx, leaderboardStatsQuery, mode).Scan(
		&stats.QueuedPlayers,
		&stats.ActiveMatchPlayers,
		&stats.RankedPlayers,
		&stats.MatchesToday,
		&stats.AverageDailyMatches,
	); err != nil {
		return Stats{}, fmt.Errorf("get leaderboard stats: %w", err)
	}
	if stats.AverageDailyMatches > 0 {
		stats.MatchesTodayGrowthPercent = ((float64(stats.MatchesToday) - stats.AverageDailyMatches) / stats.AverageDailyMatches) * 100
	}
	stats.CachedAt = time.Now().UTC()

	return stats, nil
}

const leaderboardStatsQuery = `
	with active_players as (
		select player1_id as user_id
		from matches
		where status in ('pending', 'running')
		union
		select player2_id as user_id
		from matches
		where status in ('pending', 'running')
	),
	days as (
		select generate_series(
			date_trunc('day', now()) - interval '30 days',
			date_trunc('day', now()) - interval '1 day',
			interval '1 day'
		) as day
	),
	daily_matches as (
		select date_trunc('day', created_at) as day, count(*)::float8 as count
		from matches
		where rating_mode = $1
		  and created_at >= date_trunc('day', now()) - interval '30 days'
		  and created_at < date_trunc('day', now())
		group by 1
	)
	select
		(select count(*)::int from matchmaking_queue) as queued_players,
		(select count(*)::int from active_players) as active_match_players,
		(select count(*)::int from user_ratings where mode = $1 and rated_games > 0) as ranked_players,
		(
			select count(*)::int
			from matches
			where rating_mode = $1
			  and created_at >= date_trunc('day', now())
		) as matches_today,
		coalesce(avg(coalesce(daily_matches.count, 0)), 0)::float8 as average_daily_matches
	from days
	left join daily_matches on daily_matches.day = days.day
`

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
