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

const matchSelectColumns = `
	id,
	rating_mode,
	task_mode,
	task_id,
	task_difficulty_snapshot,
	player1_id,
	player2_id,
	player1_ready,
	player2_ready,
	winner_id,
	status,
	judge_status,
	result_type,
	winning_reason,
	is_rated,
	rating_applied,
	player1_score,
	player2_score,
	player1_attempts,
	player2_attempts,
	player1_solved,
	player2_solved,
	player1_solved_at,
	player2_solved_at,
	started_at,
	finished_at,
	expires_at,
	created_at,
	updated_at
`

func (r *Repository) CreateMatch(ctx context.Context, match Match) (Match, error) {
	query := `
		insert into matches (
			id,
			rating_mode,
			task_mode,
			task_id,
			task_difficulty_snapshot,
			player1_id,
			player2_id,
			player1_ready,
			player2_ready,
			winner_id,
			status,
			judge_status,
			result_type,
			winning_reason,
			is_rated,
			rating_applied,
			player1_score,
			player2_score,
			player1_attempts,
			player2_attempts,
			player1_solved,
			player2_solved,
			player1_solved_at,
			player2_solved_at,
			started_at,
			finished_at,
			expires_at,
			created_at,
			updated_at
		)
		values (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25,
			$26, $27, now(), now()
		)
		returning ` + matchSelectColumns

	row := r.db.QueryRow(
		ctx,
		query,
		match.ID,
		match.RatingMode,
		match.TaskMode,
		match.TaskID,
		match.TaskDifficultySnapshot,
		match.Player1ID,
		match.Player2ID,
		match.Player1Ready,
		match.Player2Ready,
		match.WinnerID,
		match.Status,
		match.JudgeStatus,
		match.ResultType,
		match.WinningReason,
		match.IsRated,
		match.RatingApplied,
		match.Player1Score,
		match.Player2Score,
		match.Player1Attempts,
		match.Player2Attempts,
		match.Player1Solved,
		match.Player2Solved,
		match.Player1SolvedAt,
		match.Player2SolvedAt,
		match.StartedAt,
		match.FinishedAt,
		match.ExpiresAt,
	)

	created, err := scanMatch(row)
	if err != nil {
		return Match{}, fmt.Errorf("create match: %w", err)
	}

	return created, nil
}

func (r *Repository) GetMatch(ctx context.Context, matchID uuid.UUID) (Match, error) {
	query := `
		select ` + matchSelectColumns + `
		from matches
		where id = $1
	`

	match, err := scanMatch(r.db.QueryRow(ctx, query, matchID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Match{}, ErrMatchNotFound
		}
		return Match{}, fmt.Errorf("get match: %w", err)
	}

	return match, nil
}

func (r *Repository) StartMatch(ctx context.Context, matchID uuid.UUID, actorID uuid.UUID, duration time.Duration) (Match, error) {
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

	if actorID != match.Player1ID && actorID != match.Player2ID {
		return Match{}, ErrSubmissionUserNotInMatch
	}

	player1Ready := match.Player1Ready || actorID == match.Player1ID
	player2Ready := match.Player2Ready || actorID == match.Player2ID
	if !player1Ready || !player2Ready {
		query := `
			update matches
			set player1_ready = $2,
				player2_ready = $3,
				updated_at = now()
			where id = $1
			returning ` + matchSelectColumns

		updated, err := scanMatch(tx.QueryRow(ctx, query, matchID, player1Ready, player2Ready))
		if err != nil {
			return Match{}, fmt.Errorf("accept match: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return Match{}, fmt.Errorf("commit tx: %w", err)
		}

		return updated, nil
	}

	query := `
		update matches
		set status = $2,
			judge_status = $3,
			player1_ready = true,
			player2_ready = true,
			started_at = now(),
			expires_at = now() + ($4::text)::interval,
			updated_at = now()
		where id = $1
		returning ` + matchSelectColumns

	if duration <= 0 {
		duration = 20 * time.Minute
	}

	updated, err := scanMatch(tx.QueryRow(
		ctx,
		query,
		matchID,
		MatchStatusRunning,
		JudgeStatusRunning,
		duration.String(),
	))
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
	case MatchStatusPending:
	case MatchStatusRunning:
		return ErrMatchAlreadyStarted
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
			result_type = $3,
			winning_reason = $4,
			judge_status = $5,
			finished_at = coalesce(finished_at, now()),
			updated_at = now()
		where id = $1
	`

	ct, err := tx.Exec(
		ctx,
		query,
		matchID,
		MatchStatusCancelled,
		MatchResultTypeCancelled,
		WinningReasonCancelled,
		JudgeStatusFailed,
	)
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
		select ` + matchSelectColumns + `
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
		match, err := scanMatch(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user matches: %w", err)
		}
		matches = append(matches, match)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user matches: %w", err)
	}

	return matches, nil
}

func (r *Repository) GetCurrentMatch(ctx context.Context, userID uuid.UUID) (Match, error) {
	query := `
		select ` + matchSelectColumns + `
		from matches
		where (player1_id = $1 or player2_id = $1)
		  and status in ('pending', 'running')
		order by
			case when status = 'running' then 0 else 1 end,
			created_at desc
		limit 1
	`

	match, err := scanMatch(r.db.QueryRow(ctx, query, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Match{}, nil
		}
		return Match{}, fmt.Errorf("get current match: %w", err)
	}

	return match, nil
}

func (r *Repository) FindQueuedOpponent(ctx context.Context, input JoinQueueInput) (*uuid.UUID, error) {
	if input.RatingMode == "" {
		input.RatingMode = "global"
	}
	if input.TaskMode == "" {
		input.TaskMode = TaskModeNormal
	}

	const query = `
		with requester as (
			select coalesce((
				select rating
				from user_ratings
				where user_id = $1
				  and mode = $2
			), 1000) as rating
		)
		select q.user_id
		from matchmaking_queue q
		cross join requester
		left join user_ratings opponent_rating
		  on opponent_rating.user_id = q.user_id
		 and opponent_rating.mode = q.rating_mode
		where q.user_id <> $1
		  and q.rating_mode = $2
		  and q.task_mode = $3
		  and q.is_rated = $4
		  and abs(coalesce(opponent_rating.rating, 1000) - requester.rating) <= case
			when now() - q.created_at < interval '1 minute' then 60
			when now() - q.created_at < interval '3 minutes' then 120
			when now() - q.created_at < interval '5 minutes' then 160
			else 250
		  end
		order by abs(coalesce(opponent_rating.rating, 1000) - requester.rating) asc,
		         q.created_at asc
		limit 1
	`

	var opponentID uuid.UUID
	if err := r.db.QueryRow(ctx, query, input.UserID, input.RatingMode, input.TaskMode, input.IsRated).Scan(&opponentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find queued opponent: %w", err)
	}

	return &opponentID, nil
}

func (r *Repository) EnqueuePlayer(ctx context.Context, input JoinQueueInput) error {
	if input.RatingMode == "" {
		input.RatingMode = "global"
	}
	if input.TaskMode == "" {
		input.TaskMode = TaskModeNormal
	}

	_, err := r.db.Exec(ctx, `
		insert into matchmaking_queue (user_id, rating_mode, task_mode, is_rated, created_at)
		values ($1, $2, $3, $4, now())
		on conflict (user_id) do update set
			rating_mode = excluded.rating_mode,
			task_mode = excluded.task_mode,
			is_rated = excluded.is_rated,
			created_at = now()
	`, input.UserID, input.RatingMode, input.TaskMode, input.IsRated)
	if err != nil {
		return fmt.Errorf("enqueue player: %w", err)
	}

	return nil
}

func (r *Repository) RemoveQueuedPlayers(ctx context.Context, userIDs ...uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	_, err := r.db.Exec(ctx, `
		delete from matchmaking_queue
		where user_id = any($1)
	`, userIDs)
	if err != nil {
		return fmt.Errorf("remove queued players: %w", err)
	}

	return nil
}

func (r *Repository) ListMatchSubmissions(ctx context.Context, matchID uuid.UUID) ([]MatchSubmission, error) {
	const query = `
		select
			id,
			match_id,
			task_id,
			user_id,
			language,
			language_version,
			status,
			verdict,
			passed_tests,
			total_tests,
			score,
			created_at,
			started_at,
			finished_at,
			updated_at
		from submissions
		where match_id = $1
		order by created_at desc
		limit 100
	`

	rows, err := r.db.Query(ctx, query, matchID)
	if err != nil {
		return nil, fmt.Errorf("list match submissions: %w", err)
	}
	defer rows.Close()

	items := make([]MatchSubmission, 0)
	for rows.Next() {
		var item MatchSubmission
		if err := rows.Scan(
			&item.ID,
			&item.MatchID,
			&item.TaskID,
			&item.UserID,
			&item.Language,
			&item.LanguageVersion,
			&item.Status,
			&item.Verdict,
			&item.PassedTests,
			&item.TotalTests,
			&item.Score,
			&item.CreatedAt,
			&item.StartedAt,
			&item.FinishedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan match submission: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate match submissions: %w", err)
	}

	return items, nil
}

func (r *Repository) FinishExpiredMatches(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.Query(ctx, `
		select id
		from matches
		where status = 'running'
		  and expires_at is not null
		  and expires_at <= $1
		order by expires_at asc
		limit $2
	`, now, limit)
	if err != nil {
		return 0, fmt.Errorf("list expired matches: %w", err)
	}
	defer rows.Close()

	matchIDs := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("scan expired match id: %w", err)
		}
		matchIDs = append(matchIDs, id)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate expired matches: %w", err)
	}

	finished := 0
	for _, matchID := range matchIDs {
		ok, err := r.finishExpiredMatch(ctx, matchID, now)
		if err != nil {
			return finished, err
		}
		if ok {
			finished++
		}
	}

	return finished, nil
}

func (r *Repository) CancelExpiredPendingMatches(ctx context.Context, now time.Time, pendingTimeout time.Duration, limit int) (int, error) {
	if pendingTimeout <= 0 {
		pendingTimeout = 30 * time.Second
	}
	if limit <= 0 {
		limit = 50
	}

	ct, err := r.db.Exec(ctx, `
		with expired as (
			select id
			from matches
			where status = 'pending'
			  and created_at <= now() - ($1::text)::interval
			order by created_at asc
			limit $2
			for update skip locked
		)
		update matches m
		set status = $3,
			result_type = $4,
			winning_reason = $5,
			judge_status = $6,
			finished_at = coalesce(m.finished_at, now()),
			updated_at = now()
		from expired
		where m.id = expired.id
	`, pendingTimeout.String(), limit, MatchStatusCancelled, MatchResultTypeCancelled, WinningReasonCancelled, JudgeStatusFailed)
	if err != nil {
		return 0, fmt.Errorf("cancel expired pending matches: %w", err)
	}

	return int(ct.RowsAffected()), nil
}

func (r *Repository) finishExpiredMatch(ctx context.Context, matchID uuid.UUID, now time.Time) (bool, error) {
	match, err := r.GetMatch(ctx, matchID)
	if err != nil {
		return false, err
	}
	if match.Status != MatchStatusRunning {
		return false, nil
	}

	input := FinishMatchInput{
		MatchID:       match.ID,
		ResultType:    MatchResultTypeDraw,
		WinningReason: WinningReasonDraw,
		FinishedAt:    &now,
		JudgeStatus:   ptrJudgeStatus(JudgeStatusFinished),
	}

	if match.Player1Score > match.Player2Score {
		winnerID := match.Player1ID
		input.WinnerID = &winnerID
		input.ResultType = MatchResultTypePlayer1Win
		input.WinningReason = WinningReasonAcceptedMoreTests
	} else if match.Player2Score > match.Player1Score {
		winnerID := match.Player2ID
		input.WinnerID = &winnerID
		input.ResultType = MatchResultTypePlayer2Win
		input.WinningReason = WinningReasonAcceptedMoreTests
	}

	if err := r.FinishMatch(ctx, input); err != nil {
		if errors.Is(err, ErrMatchAlreadyFinished) ||
			errors.Is(err, ErrRatingAlreadyApplied) ||
			errors.Is(err, ErrMatchCancelled) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func ptrJudgeStatus(value JudgeStatus) *JudgeStatus {
	return &value
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

	if err := validateFinishInput(match, input); err != nil {
		return err
	}

	finishedAt := time.Now().UTC()
	if input.FinishedAt != nil {
		finishedAt = input.FinishedAt.UTC()
	}

	judgeStatus := JudgeStatusFinished
	if input.JudgeStatus != nil {
		judgeStatus = *input.JudgeStatus
	}

	updateMatchQuery := `
		update matches
		set winner_id = $2,
			status = $3,
			result_type = $4,
			winning_reason = $5,
			judge_status = $6,
			finished_at = $7,
			updated_at = now()
		where id = $1
	`

	_, err = tx.Exec(
		ctx,
		updateMatchQuery,
		input.MatchID,
		input.WinnerID,
		MatchStatusFinished,
		input.ResultType,
		input.WinningReason,
		judgeStatus,
		finishedAt,
	)
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

	player1Rating, err := r.getUserRatingForUpdate(ctx, tx, match.Player1ID, match.RatingMode)
	if err != nil {
		return err
	}
	player2Rating, err := r.getUserRatingForUpdate(ctx, tx, match.Player2ID, match.RatingMode)
	if err != nil {
		return err
	}

	result1, result2, err := eloResultsForMatch(match, input)
	if err != nil {
		return err
	}

	newRating1 := calculateNewRating(
		player1Rating.Rating,
		player2Rating.Rating,
		player1Rating.RatedGames,
		result1,
	)
	newRating2 := calculateNewRating(
		player2Rating.Rating,
		player1Rating.Rating,
		player2Rating.RatedGames,
		result2,
	)

	if err := r.updateUserRatingAfterMatch(ctx, tx, player1Rating, newRating1, result1); err != nil {
		return err
	}
	if err := r.updateUserRatingAfterMatch(ctx, tx, player2Rating, newRating2, result2); err != nil {
		return err
	}

	if err := r.insertRatingHistory(
		ctx,
		tx,
		player1Rating.UserID,
		input.MatchID,
		match.RatingMode,
		player1Rating.Rating,
		newRating1,
	); err != nil {
		return err
	}
	if err := r.insertRatingHistory(
		ctx,
		tx,
		player2Rating.UserID,
		input.MatchID,
		match.RatingMode,
		player2Rating.Rating,
		newRating2,
	); err != nil {
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
		select ` + matchSelectColumns + `
		from matches
		where id = $1
		for update
	`

	match, err := scanMatch(tx.QueryRow(ctx, query, matchID))
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

func (r *Repository) insertRatingHistory(
	ctx context.Context,
	tx pgx.Tx,
	userID, matchID uuid.UUID,
	mode string,
	oldRating, newRating int,
) error {
	query := `
		insert into rating_history (
			id, user_id, match_id, mode, old_rating, new_rating, delta, created_at
		)
		values ($1, $2, $3, $4, $5, $6, $7, now())
	`

	_, err := tx.Exec(
		ctx,
		query,
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

type matchScanner interface {
	Scan(dest ...any) error
}

func scanMatch(row matchScanner) (Match, error) {
	var match Match

	err := row.Scan(
		&match.ID,
		&match.RatingMode,
		&match.TaskMode,
		&match.TaskID,
		&match.TaskDifficultySnapshot,
		&match.Player1ID,
		&match.Player2ID,
		&match.Player1Ready,
		&match.Player2Ready,
		&match.WinnerID,
		&match.Status,
		&match.JudgeStatus,
		&match.ResultType,
		&match.WinningReason,
		&match.IsRated,
		&match.RatingApplied,
		&match.Player1Score,
		&match.Player2Score,
		&match.Player1Attempts,
		&match.Player2Attempts,
		&match.Player1Solved,
		&match.Player2Solved,
		&match.Player1SolvedAt,
		&match.Player2SolvedAt,
		&match.StartedAt,
		&match.FinishedAt,
		&match.ExpiresAt,
		&match.CreatedAt,
		&match.UpdatedAt,
	)
	if err != nil {
		return Match{}, err
	}

	return match, nil
}

func validateFinishInput(match Match, input FinishMatchInput) error {
	switch input.ResultType {
	case MatchResultTypeDraw:
		if input.WinnerID != nil {
			return ErrInvalidWinner
		}
	case MatchResultTypePlayer1Win:
		if input.WinnerID == nil || *input.WinnerID != match.Player1ID {
			return ErrInvalidWinner
		}
	case MatchResultTypePlayer2Win:
		if input.WinnerID == nil || *input.WinnerID != match.Player2ID {
			return ErrInvalidWinner
		}
	case MatchResultTypeCancelled:
		return ErrInvalidMatchStatus
	default:
		return ErrInvalidMatchStatus
	}

	return nil
}

func eloResultsForMatch(match Match, input FinishMatchInput) (float64, float64, error) {
	switch input.ResultType {
	case MatchResultTypeDraw:
		return 0.5, 0.5, nil
	case MatchResultTypePlayer1Win:
		return 1.0, 0.0, nil
	case MatchResultTypePlayer2Win:
		return 0.0, 1.0, nil
	default:
		return 0, 0, ErrInvalidMatchStatus
	}
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

func (r *Repository) ApplySubmissionProgress(ctx context.Context, input ApplySubmissionProgressInput) error {
	if input.MatchID == uuid.Nil {
		return fmt.Errorf("apply submission progress: empty match id")
	}
	if input.UserID == uuid.Nil {
		return fmt.Errorf("apply submission progress: empty user id")
	}

	match, err := r.GetMatch(ctx, input.MatchID)
	if err != nil {
		return err
	}

	if match.Status == MatchStatusFinished || match.Status == MatchStatusCancelled {
		return nil
	}

	if input.UserID == match.Player1ID {
		const q = `
update matches
set
	player1_attempts = player1_attempts + 1,
	player1_score = greatest(player1_score, $2),
	player1_solved = case
		when $3 then true
		else player1_solved
	end,
	player1_solved_at = case
		when $3 and player1_solved_at is null then $4
		else player1_solved_at
	end,
	judge_status = coalesce($5, judge_status)
where id = $1
`
		_, err := r.db.Exec(ctx, q,
			input.MatchID,
			input.Score,
			input.IsAccepted,
			input.SolvedAt,
			input.JudgeStatus,
		)
		return err
	}

	if input.UserID == match.Player2ID {
		const q = `
update matches
set
	player2_attempts = player2_attempts + 1,
	player2_score = greatest(player2_score, $2),
	player2_solved = case
		when $3 then true
		else player2_solved
	end,
	player2_solved_at = case
		when $3 and player2_solved_at is null then $4
		else player2_solved_at
	end,
	judge_status = coalesce($5, judge_status)
where id = $1
`
		_, err := r.db.Exec(ctx, q,
			input.MatchID,
			input.Score,
			input.IsAccepted,
			input.SolvedAt,
			input.JudgeStatus,
		)
		return err
	}

	return ErrSubmissionUserNotInMatch
}
