package users

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

const profileRecordSelectQuery = `
	select
		u.id,
		u.username,
		u.display_name,
		u.bio,
		u.country,
		u.country_code,
		u.state_province,
		u.city,
		u.avatar_url,
		u.avatar_object_key,
		u.provider_avatar_url,
		u.avatar_source,
		coalesce((
			select oa.provider
			from oauth_accounts oa
			where oa.user_id = u.id
			order by oa.created_at asc
			limit 1
		), ''),
		u.created_at
	from users u
`

type profileRecord struct {
	ID                uuid.UUID
	Username          string
	DisplayName       string
	Bio               string
	Country           string
	CountryCode       string
	StateProvince     string
	City              string
	AvatarURL         string
	AvatarObjectKey   string
	ProviderAvatarURL string
	AvatarSource      AvatarSource
	AuthProvider      string
	CreatedAt         time.Time
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetProfileByUsername(ctx context.Context, username string) (Profile, error) {
	record, err := r.getProfileRecordByUsername(ctx, username)
	if err != nil {
		return Profile{}, err
	}

	return r.buildProfile(ctx, record)
}

func (r *Repository) UpdateProfile(ctx context.Context, viewerID uuid.UUID, username string, input UpdateProfileInput) (Profile, error) {
	ownerID, err := r.getUserIDByUsername(ctx, username)
	if err != nil {
		return Profile{}, err
	}
	if ownerID != viewerID {
		return Profile{}, ErrForbidden
	}

	query := `
		update users
		set display_name = $2,
			bio = $3,
			country = $4,
			country_code = $5,
			state_province = $6,
			city = $7,
			avatar_url = case
				when $8 <> '' then $8
				when $9 then ''
				else avatar_url
			end,
			avatar_object_key = case
				when $8 <> '' then ''
				when $9 then ''
				else avatar_object_key
			end,
			avatar_source = case
				when $8 <> '' then 'custom'
				when $9 and provider_avatar_url <> '' then 'provider'
				when $9 then 'none'
				else avatar_source
			end,
			updated_at = now()
		where id = $1
	`

	if _, err := r.db.Exec(
		ctx,
		query,
		viewerID,
		input.DisplayName,
		input.Bio,
		input.Country,
		input.CountryCode,
		input.StateProvince,
		input.City,
		input.AvatarDataURL,
		input.RemoveCustomAvatar,
	); err != nil {
		return Profile{}, fmt.Errorf("update profile: %w", err)
	}

	record, err := r.getProfileRecordByID(ctx, viewerID)
	if err != nil {
		return Profile{}, err
	}

	return r.buildProfile(ctx, record)
}

func (r *Repository) GetAvatarObjectKey(ctx context.Context, userID uuid.UUID) (string, error) {
	const query = `
		select avatar_object_key
		from users
		where id = $1
	`

	var objectKey string
	if err := r.db.QueryRow(ctx, query, userID).Scan(&objectKey); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}

		return "", fmt.Errorf("get avatar object key: %w", err)
	}

	return objectKey, nil
}

func (r *Repository) UpdateCustomAvatar(
	ctx context.Context,
	viewerID uuid.UUID,
	username string,
	avatarURL string,
	avatarObjectKey string,
) (Profile, string, error) {
	ownerID, err := r.getUserIDByUsername(ctx, username)
	if err != nil {
		return Profile{}, "", err
	}
	if ownerID != viewerID {
		return Profile{}, "", ErrForbidden
	}

	const query = `
		with previous as (
			select avatar_object_key
			from users
			where id = $1
		),
		updated as (
			update users
			set avatar_url = $2,
				avatar_object_key = $3,
				avatar_source = 'custom',
				updated_at = now()
			where id = $1
			returning id
		)
		select previous.avatar_object_key
		from previous
		join updated on true
	`

	var oldAvatarObjectKey string
	if err := r.db.QueryRow(ctx, query, viewerID, avatarURL, avatarObjectKey).Scan(&oldAvatarObjectKey); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Profile{}, "", ErrNotFound
		}

		return Profile{}, "", fmt.Errorf("update custom avatar: %w", err)
	}

	record, err := r.getProfileRecordByID(ctx, viewerID)
	if err != nil {
		return Profile{}, "", err
	}

	profile, err := r.buildProfile(ctx, record)
	if err != nil {
		return Profile{}, "", err
	}

	return profile, oldAvatarObjectKey, nil
}

func (r *Repository) buildProfile(ctx context.Context, record profileRecord) (Profile, error) {
	stats, err := r.getProfileStats(ctx, record.ID, record.CountryCode, record.Country)
	if err != nil {
		return Profile{}, err
	}

	eloHistory, err := r.getEloHistory(ctx, record.ID)
	if err != nil {
		return Profile{}, err
	}

	topLanguages, err := r.getTopLanguages(ctx, record.ID)
	if err != nil {
		return Profile{}, err
	}

	recentMatches, err := r.getRecentMatches(ctx, record.ID)
	if err != nil {
		return Profile{}, err
	}

	return Profile{
		ID:                record.ID,
		Username:          record.Username,
		DisplayName:       record.DisplayName,
		Bio:               record.Bio,
		Country:           record.Country,
		CountryCode:       record.CountryCode,
		StateProvince:     record.StateProvince,
		City:              record.City,
		AvatarURL:         resolveAvatarURL(record.AvatarSource, record.AvatarURL, record.ProviderAvatarURL),
		ProviderAvatarURL: record.ProviderAvatarURL,
		AvatarSource:      record.AvatarSource,
		AuthProvider:      record.AuthProvider,
		CreatedAt:         record.CreatedAt,
		Stats:             stats,
		EloHistory:        eloHistory,
		TopLanguages:      topLanguages,
		RecentMatches:     recentMatches,
	}, nil
}

func (r *Repository) getProfileRecordByUsername(ctx context.Context, username string) (profileRecord, error) {
	const query = profileRecordSelectQuery + `
		where lower(u.username) = lower($1)
	`

	return scanProfileRecord(r.db.QueryRow(ctx, query, username))
}

func (r *Repository) getProfileRecordByID(ctx context.Context, userID uuid.UUID) (profileRecord, error) {
	const query = profileRecordSelectQuery + `
		where u.id = $1
	`

	return scanProfileRecord(r.db.QueryRow(ctx, query, userID))
}

func (r *Repository) getUserIDByUsername(ctx context.Context, username string) (uuid.UUID, error) {
	const query = `
		select id
		from users
		where lower(username) = lower($1)
	`

	var userID uuid.UUID
	if err := r.db.QueryRow(ctx, query, username).Scan(&userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}

		return uuid.Nil, fmt.Errorf("get user id by username: %w", err)
	}

	return userID, nil
}

func (r *Repository) getProfileStats(ctx context.Context, userID uuid.UUID, countryCode string, countryName string) (ProfileStats, error) {
	const query = `
		select
			coalesce(ur.rating, 1000) as elo_rating,
			count(m.id) as total_matches,
			count(m.id) filter (where m.winner_id = $1) as wins,
			count(m.id) filter (where m.winner_id is not null and m.winner_id <> $1) as losses,
			count(m.id) filter (where m.winner_id is null) as draws
		from users u
		left join user_ratings ur
			on ur.user_id = u.id
			and ur.mode = 'global'
		left join matches m
			on (m.player1_id = u.id or m.player2_id = u.id)
			and m.status = 'finished'
		where u.id = $1
		group by ur.rating
	`

	var stats ProfileStats
	if err := r.db.QueryRow(ctx, query, userID).Scan(
		&stats.EloRating,
		&stats.TotalMatches,
		&stats.Wins,
		&stats.Losses,
		&stats.Draws,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProfileStats{EloRating: 1000}, nil
		}

		return ProfileStats{}, fmt.Errorf("get profile stats: %w", err)
	}

	if stats.TotalMatches > 0 {
		stats.WinRate = float64(stats.Wins) / float64(stats.TotalMatches)
	}

	maxWinStreak, err := r.getMaxWinStreak(ctx, userID)
	if err != nil {
		return ProfileStats{}, err
	}
	stats.MaxWinStreak = maxWinStreak

	globalRank, globalPlayersCount, err := r.getGlobalRank(ctx, userID)
	if err != nil {
		return ProfileStats{}, err
	}
	stats.GlobalRank = globalRank
	stats.GlobalPlayersCount = globalPlayersCount

	if countryCode != "" || countryName != "" {
		regionalRank, regionalPlayersCount, err := r.getRegionalRank(
			ctx,
			userID,
			countryCode,
			countryName,
		)
		if err != nil {
			return ProfileStats{}, err
		}

		stats.RegionalRank = regionalRank
		stats.RegionalPlayersCount = regionalPlayersCount
	}

	return stats, nil
}

func (r *Repository) getGlobalRank(ctx context.Context, userID uuid.UUID) (*int, *int, error) {
	const query = `
		with ranked as (
			select
				u.id,
				row_number() over (
					order by coalesce(ur.rating, 1000) desc, lower(u.username) asc, u.id asc
				) as global_rank,
				count(*) over () as global_players_count
			from users u
			left join user_ratings ur
				on ur.user_id = u.id
				and ur.mode = 'global'
		)
		select global_rank, global_players_count
		from ranked
		where id = $1
	`

	var globalRank int
	var globalPlayersCount int
	if err := r.db.QueryRow(ctx, query, userID).Scan(&globalRank, &globalPlayersCount); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}

		return nil, nil, fmt.Errorf("get global rank: %w", err)
	}

	return &globalRank, &globalPlayersCount, nil
}

func (r *Repository) getRegionalRank(ctx context.Context, userID uuid.UUID, countryCode string, countryName string) (*int, *int, error) {
	const query = `
		with ranked as (
			select
				u.id,
				row_number() over (
					order by coalesce(ur.rating, 1000) desc, lower(u.username) asc, u.id asc
				) as regional_rank,
				count(*) over () as regional_players_count
			from users u
			left join user_ratings ur
				on ur.user_id = u.id
				and ur.mode = 'global'
				where (
					($2 <> '' and upper(u.country_code) = upper($2))
					or (
						$3 <> ''
						and (
							($2 = '' and lower(u.country) = lower($3))
							or ($2 <> '' and u.country_code = '' and lower(u.country) = lower($3))
						)
					)
				)
		)
		select regional_rank, regional_players_count
		from ranked
		where id = $1
	`

	var regionalRank int
	var regionalPlayersCount int
	if err := r.db.QueryRow(ctx, query, userID, countryCode, countryName).Scan(&regionalRank, &regionalPlayersCount); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}

		return nil, nil, fmt.Errorf("get regional rank: %w", err)
	}

	return &regionalRank, &regionalPlayersCount, nil
}

func (r *Repository) getMaxWinStreak(ctx context.Context, userID uuid.UUID) (int, error) {
	const query = `
		select winner_id
		from matches
		where status = 'finished'
			and (player1_id = $1 or player2_id = $1)
		order by coalesce(finished_at, created_at) asc, created_at asc, id asc
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return 0, fmt.Errorf("list matches for streak: %w", err)
	}
	defer rows.Close()

	current := 0
	maximum := 0

	for rows.Next() {
		var winnerID *uuid.UUID
		if err := rows.Scan(&winnerID); err != nil {
			return 0, fmt.Errorf("scan streak match: %w", err)
		}

		if winnerID != nil && *winnerID == userID {
			current++
			if current > maximum {
				maximum = current
			}
			continue
		}

		current = 0
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate streak matches: %w", err)
	}

	return maximum, nil
}

func (r *Repository) getEloHistory(ctx context.Context, userID uuid.UUID) ([]EloHistoryPoint, error) {
	const query = `
		with filtered as (
			select
				(created_at at time zone 'utc')::date as bucket_day,
				delta,
				new_rating,
				created_at
			from rating_history
			where user_id = $1
				and mode = 'global'
				and created_at >= now() - interval '12 months'
		),
		daily as (
			select
				bucket_day,
				sum(delta) as delta,
				count(*) as matches_played
			from filtered
			group by bucket_day
		),
		last_rating as (
			select distinct on (bucket_day)
				bucket_day,
				new_rating
			from filtered
			order by bucket_day, created_at desc
		)
		select
			to_char(daily.bucket_day, 'YYYY-MM-DD') as bucket_day,
			daily.delta,
			last_rating.new_rating,
			daily.matches_played
		from daily
		join last_rating using (bucket_day)
		order by daily.bucket_day asc
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get elo history: %w", err)
	}
	defer rows.Close()

	points := make([]EloHistoryPoint, 0)
	for rows.Next() {
		var point EloHistoryPoint
		if err := rows.Scan(&point.Date, &point.Delta, &point.Rating, &point.MatchesPlayed); err != nil {
			return nil, fmt.Errorf("scan elo history: %w", err)
		}
		points = append(points, point)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate elo history: %w", err)
	}

	return points, nil
}

func (r *Repository) getTopLanguages(ctx context.Context, userID uuid.UUID) ([]TopLanguage, error) {
	const query = `
		with language_counts as (
			select
				lower(trim(language)) as language,
				count(*)::float8 as submissions_count
			from submissions
			where user_id = $1
				and trim(language) <> ''
			group by lower(trim(language))
		),
		total as (
			select sum(submissions_count) as submissions_count
			from language_counts
		)
		select
			language_counts.language,
			language_counts.submissions_count / total.submissions_count as usage_share
		from language_counts
		cross join total
		order by language_counts.submissions_count desc, language_counts.language asc
		limit 5
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get top languages: %w", err)
	}
	defer rows.Close()

	languages := make([]TopLanguage, 0, 5)
	for rows.Next() {
		var language TopLanguage
		if err := rows.Scan(&language.Name, &language.UsageShare); err != nil {
			return nil, fmt.Errorf("scan top language: %w", err)
		}

		language.Name = displayLanguageName(language.Name)
		languages = append(languages, language)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top languages: %w", err)
	}

	return languages, nil
}

func (r *Repository) getRecentMatches(ctx context.Context, userID uuid.UUID) ([]RecentMatch, error) {
	const query = `
		select
			m.id,
			case
				when m.status = 'cancelled' then 'cancelled'
				when m.winner_id is null then 'draw'
				when m.winner_id = $1 then 'win'
				else 'loss'
			end as result,
			opponent.id,
			opponent.username,
			opponent.display_name,
			case
				when opponent.avatar_source = 'custom' and opponent.avatar_url <> '' then opponent.avatar_url
				when opponent.avatar_source = 'provider' and opponent.provider_avatar_url <> '' then opponent.provider_avatar_url
				when opponent.avatar_url <> '' then opponent.avatar_url
				else opponent.provider_avatar_url
			end as opponent_avatar_url,
			m.task_difficulty_snapshot,
			case
				when m.is_rated and m.rating_applied then rh.delta
				else null
			end as elo_delta,
			m.is_rated,
			coalesce(m.finished_at, m.updated_at, m.created_at) as finished_at
		from matches m
		join users opponent
			on opponent.id = case
				when m.player1_id = $1 then m.player2_id
				else m.player1_id
			end
		left join lateral (
			select delta
			from rating_history
			where match_id = m.id
				and user_id = $1
				and mode = m.rating_mode
			order by created_at desc, id desc
			limit 1
		) rh on true
		where (m.player1_id = $1 or m.player2_id = $1)
			and m.status in ('finished', 'cancelled')
		order by coalesce(m.finished_at, m.updated_at, m.created_at) desc, m.id desc
		limit 10
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get recent matches: %w", err)
	}
	defer rows.Close()

	matches := make([]RecentMatch, 0, 10)
	for rows.Next() {
		var match RecentMatch
		if err := rows.Scan(
			&match.ID,
			&match.Result,
			&match.Opponent.ID,
			&match.Opponent.Username,
			&match.Opponent.DisplayName,
			&match.Opponent.AvatarURL,
			&match.Difficulty,
			&match.EloDelta,
			&match.IsRated,
			&match.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan recent match: %w", err)
		}

		if match.Opponent.DisplayName == "" {
			match.Opponent.DisplayName = match.Opponent.Username
		}

		matches = append(matches, match)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent matches: %w", err)
	}

	return matches, nil
}

func scanProfileRecord(row pgx.Row) (profileRecord, error) {
	var record profileRecord
	if err := row.Scan(
		&record.ID,
		&record.Username,
		&record.DisplayName,
		&record.Bio,
		&record.Country,
		&record.CountryCode,
		&record.StateProvince,
		&record.City,
		&record.AvatarURL,
		&record.AvatarObjectKey,
		&record.ProviderAvatarURL,
		&record.AvatarSource,
		&record.AuthProvider,
		&record.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return profileRecord{}, ErrNotFound
		}

		return profileRecord{}, fmt.Errorf("scan profile record: %w", err)
	}

	return record, nil
}

func resolveAvatarURL(source AvatarSource, avatarURL string, providerAvatarURL string) string {
	switch source {
	case AvatarSourceCustom:
		if avatarURL != "" {
			return avatarURL
		}
	case AvatarSourceProvider:
		if providerAvatarURL != "" {
			return providerAvatarURL
		}
	}

	if avatarURL != "" {
		return avatarURL
	}

	return providerAvatarURL
}

func displayLanguageName(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "typescript":
		return "TypeScript"
	case "javascript":
		return "JavaScript"
	case "python":
		return "Python"
	case "java":
		return "Java"
	case "cpp":
		return "C++"
	case "go":
		return "Go"
	case "rust":
		return "Rust"
	case "zig":
		return "Zig"
	case "csharp":
		return "C#"
	case "c":
		return "C"
	case "ruby":
		return "Ruby"
	case "php":
		return "PHP"
	case "sql":
		return "SQL"
	default:
		return language
	}
}
