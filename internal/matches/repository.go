package matches

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateMatch(ctx context.Context, match Match) (Match, error) {
	query := `
		insert into matches (
			id, mode, player1_id, player2_id, winner_id, status,
			is_rated, rating_applied, started_at, finished_at, created_at, updated_at
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), now())
		returning
			id, mode, player1_id, player2_id, winner_id, status,
			is_rated, rating_applied, started_at, finished_at, created_at, updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		match.ID,
		match.Mode,
		match.Player1ID,
		match.Player2ID,
		match.WinnerID,
		match.Status,
		match.IsRated,
		match.RatingApplied,
		match.StartedAt,
		match.FinishedAt,
	).Scan(
		&match.ID,
		&match.Mode,
		&match.Player1ID,
		&match.Player2ID,
		&match.WinnerID,
		&match.Status,
		&match.IsRated,
		&match.RatingApplied,
		&match.StartedAt,
		&match.FinishedAt,
		&match.CreatedAt,
		&match.UpdatedAt,
	)
	if err != nil {
		return Match{}, fmt.Errorf("create match: %w", err)
	}

	return match, nil
}

func (r *Repository) GetMatch(ctx context.Context, matchID uuid.UUID) (Match, error) {
	query := `
		select
			id, mode, player1_id, player2_id, winner_id, status,
			is_rated, rating_applied, started_at, finished_at, created_at, updated_at
		from matches
		where id = $1
	`

	var match Match
	err := r.db.QueryRow(ctx, query, matchID).Scan(
		&match.ID,
		&match.Mode,
		&match.Player1ID,
		&match.Player2ID,
		&match.WinnerID,
		&match.Status,
		&match.IsRated,
		&match.RatingApplied,
		&match.StartedAt,
		&match.FinishedAt,
		&match.CreatedAt,
		&match.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Match{}, ErrMatchNotFound
		}
		return Match{}, fmt.Errorf("get match: %w", err)
	}

	return match, nil
}

func (r *Repository) StartMatch(ctx context.Context, matchID uuid.UUID) (Match, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Match{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	match, err := r.getMatchForUpdate(ctx, tx, matchID)
	if err != nil {
		return Match{}, err
	}

	switch match.Status {
	case MatchStatusPending:
	case MatchStatusRunning:
		return Match{}, ErrMatchAlreadyStarted
	case MatchStatusFinished:
		return Match{}, ErrMatchAlreadyFinished
	case MatchStatusCancelled:
		return Match{}, ErrMatchCancelled
	default:
		return Match{}, ErrInvalidMatchStatus
	}

	query := `
		update matches
		set status = $2,
			started_at = now(),
			updated_at = now()
		where id = $1
		returning
			id, mode, player1_id, player2_id, winner_id, status,
			is_rated, rating_applied, started_at, finished_at, created_at, updated_at
	`

	var updated Match
	err = tx.QueryRow(ctx, query, matchID, MatchStatusRunning).Scan(
		&updated.ID,
		&updated.Mode,
		&updated.Player1ID,
		&updated.Player2ID,
		&updated.WinnerID,
		&updated.Status,
		&updated.IsRated,
		&updated.RatingApplied,
		&updated.StartedAt,
		&updated.FinishedAt,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		return Match{}, fmt.Errorf("start match: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Match{}, fmt.Errorf("commit tx: %w", err)
	}

	return updated, nil
}

func (r *Repository) CancelMatch(ctx context.Context, matchID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	match, err := r.getMatchForUpdate(ctx, tx, matchID)
	if err != nil {
		return err
	}

	switch match.Status {
	case MatchStatusPending, MatchStatusRunning:
	case MatchStatusFinished:
		return ErrMatchAlreadyFinished
	case MatchStatusCancelled:
		return ErrMatchCancelled
	default:
		return ErrInvalidMatchStatus
	}

	query := `
		update matches
		set status = $2,
			updated_at = now()
		where id = $1
	`
	ct, err := tx.Exec(ctx, query, matchID, MatchStatusCancelled)
	if err != nil {
		return fmt.Errorf("cancel match: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrMatchNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *Repository) ListUserMatches(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Match, error) {
	query := `
		select
			id, mode, player1_id, player2_id, winner_id, status,
			is_rated, rating_applied, started_at, finished_at, created_at, updated_at
		from matches
		where player1_id = $1 or player2_id = $1
		order by created_at desc
		limit $2 offset $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list user matches: %w", err)
	}
	defer rows.Close()

	matches := make([]Match, 0, limit)
	for rows.Next() {
		var match Match
		if err := rows.Scan(
			&match.ID,
			&match.Mode,
			&match.Player1ID,
			&match.Player2ID,
			&match.WinnerID,
			&match.Status,
			&match.IsRated,
			&match.RatingApplied,
			&match.StartedAt,
			&match.FinishedAt,
			&match.CreatedAt,
			&match.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user matches: %w", err)
		}
		matches = append(matches, match)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user matches: %w", err)
	}

	return matches, nil
}

func (r *Repository) FinishMatch(ctx context.Context, input FinishMatchInput) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	match, err := r.getMatchForUpdate(ctx, tx, input.MatchID)
	if err != nil {
		return err
	}

	switch match.Status {
	case MatchStatusPending, MatchStatusRunning:
	case MatchStatusFinished:
		if match.RatingApplied {
			return ErrRatingAlreadyApplied
		}
		return ErrMatchAlreadyFinished
	case MatchStatusCancelled:
		return ErrMatchCancelled
	default:
		return ErrInvalidMatchStatus
	}

	if err := validateWinner(match, input.WinnerID); err != nil {
		return err
	}

	now := time.Now().UTC()

	updateMatchQuery := `
		update matches
		set winner_id = $2,
			status = $3,
			finished_at = $4,
			updated_at = now()
		where id = $1
	`
	_, err = tx.Exec(ctx, updateMatchQuery, input.MatchID, input.WinnerID, MatchStatusFinished, now)
	if err != nil {
		return fmt.Errorf("update match result: %w", err)
	}

	if !match.IsRated {
		_, err = tx.Exec(ctx, `
			update matches
			set rating_applied = true,
				updated_at = now()
			where id = $1
		`, input.MatchID)
		if err != nil {
			return fmt.Errorf("mark unrated match processed: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit tx: %w", err)
		}
		return nil
	}

	player1Rating, err := r.getUserRatingForUpdate(ctx, tx, match.Player1ID, match.Mode)
	if err != nil {
		return err
	}
	player2Rating, err := r.getUserRatingForUpdate(ctx, tx, match.Player2ID, match.Mode)
	if err != nil {
		return err
	}

	var result1, result2 float64
	switch {
	case input.WinnerID == nil:
		result1 = 0.5
		result2 = 0.5
	case *input.WinnerID == match.Player1ID:
		result1 = 1.0
		result2 = 0.0
	case *input.WinnerID == match.Player2ID:
		result1 = 0.0
		result2 = 1.0
	default:
		return ErrInvalidWinner
	}

	newRating1 := calculateNewRating(player1Rating.Rating, player2Rating.Rating, player1Rating.RatedGames, result1)
	newRating2 := calculateNewRating(player2Rating.Rating, player1Rating.Rating, player2Rating.RatedGames, result2)

	if err := r.updateUserRatingAfterMatch(ctx, tx, player1Rating, newRating1, result1); err != nil {
		return err
	}
	if err := r.updateUserRatingAfterMatch(ctx, tx, player2Rating, newRating2, result2); err != nil {
		return err
	}

	if err := r.insertRatingHistory(ctx, tx, player1Rating.UserID, input.MatchID, match.Mode, player1Rating.Rating, newRating1); err != nil {
		return err
	}
	if err := r.insertRatingHistory(ctx, tx, player2Rating.UserID, input.MatchID, match.Mode, player2Rating.Rating, newRating2); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		update matches
		set rating_applied = true,
			updated_at = now()
		where id = $1
	`, input.MatchID)
	if err != nil {
		return fmt.Errorf("mark rating applied: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *Repository) getMatchForUpdate(ctx context.Context, tx pgx.Tx, matchID uuid.UUID) (Match, error) {
	query := `
		select
			id, mode, player1_id, player2_id, winner_id, status,
			is_rated, rating_applied, started_at, finished_at, created_at, updated_at
		from matches
		where id = $1
		for update
	`

	var match Match
	err := tx.QueryRow(ctx, query, matchID).Scan(
		&match.ID,
		&match.Mode,
		&match.Player1ID,
		&match.Player2ID,
		&match.WinnerID,
		&match.Status,
		&match.IsRated,
		&match.RatingApplied,
		&match.StartedAt,
		&match.FinishedAt,
		&match.CreatedAt,
		&match.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Match{}, ErrMatchNotFound
		}
		return Match{}, fmt.Errorf("get match for update: %w", err)
	}

	return match, nil
}

func (r *Repository) getUserRatingForUpdate(ctx context.Context, tx pgx.Tx, userID uuid.UUID, mode string) (UserRating, error) {
	query := `
		select
			user_id, mode, rating, rated_games, wins, losses, draws, created_at, updated_at
		from user_ratings
		where user_id = $1 and mode = $2
		for update
	`

	var rating UserRating
	err := tx.QueryRow(ctx, query, userID, mode).Scan(
		&rating.UserID,
		&rating.Mode,
		&rating.Rating,
		&rating.RatedGames,
		&rating.Wins,
		&rating.Losses,
		&rating.Draws,
		&rating.CreatedAt,
		&rating.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserRating{}, ErrRatingNotFound
		}
		return UserRating{}, fmt.Errorf("get user rating for update: %w", err)
	}

	return rating, nil
}

func (r *Repository) updateUserRatingAfterMatch(ctx context.Context, tx pgx.Tx, rating UserRating, newRating int, result float64) error {
	wins := rating.Wins
	losses := rating.Losses
	draws := rating.Draws

	switch result {
	case 1.0:
		wins++
	case 0.5:
		draws++
	case 0.0:
		losses++
	}

	query := `
		update user_ratings
		set rating = $3,
			rated_games = rated_games + 1,
			wins = $4,
			losses = $5,
			draws = $6,
			updated_at = now()
		where user_id = $1 and mode = $2
	`
	ct, err := tx.Exec(ctx, query, rating.UserID, rating.Mode, newRating, wins, losses, draws)
	if err != nil {
		return fmt.Errorf("update user rating: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrRatingNotFound
	}

	return nil
}

func (r *Repository) insertRatingHistory(ctx context.Context, tx pgx.Tx, userID, matchID uuid.UUID, mode string, oldRating, newRating int) error {
	query := `
		insert into rating_history (
			id, user_id, match_id, mode, old_rating, new_rating, delta, created_at
		)
		values ($1, $2, $3, $4, $5, $6, $7, now())
	`

	_, err := tx.Exec(ctx, query,
		uuid.New(),
		userID,
		matchID,
		mode,
		oldRating,
		newRating,
		newRating-oldRating,
	)
	if err != nil {
		return fmt.Errorf("insert rating history: %w", err)
	}

	return nil
}

func validateWinner(match Match, winnerID *uuid.UUID) error {
	if winnerID == nil {
		return nil
	}
	if *winnerID != match.Player1ID && *winnerID != match.Player2ID {
		return ErrInvalidWinner
	}
	return nil
}

func expectedScore(playerRating, opponentRating int) float64 {
	return 1.0 / (1.0 + math.Pow(10, float64(opponentRating-playerRating)/400.0))
}

func kFactor(ratedGames int) int {
	if ratedGames < 30 {
		return 40
	}
	return 20
}

func calculateNewRating(currentRating, opponentRating, ratedGames int, actualScore float64) int {
	expected := expectedScore(currentRating, opponentRating)
	k := float64(kFactor(ratedGames))
	newRating := float64(currentRating) + k*(actualScore-expected)
	return int(math.Round(newRating))
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}