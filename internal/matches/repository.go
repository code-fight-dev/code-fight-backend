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
	db matchDB
}

type matchDB interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateMatch(ctx context.Context, match Match) (Match, error) {
	row := r.db.QueryRow(
		ctx,
		createMatchQuery,
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
	match, err := scanMatch(r.db.QueryRow(ctx, getMatchQuery, matchID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Match{}, ErrMatchNotFound
		}
		return Match{}, fmt.Errorf("get match: %w", err)
	}

	if err := r.hydrateMatchRatingDeltas(ctx, &match); err != nil {
		return Match{}, err
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
		updated, err := scanMatch(tx.QueryRow(ctx, acceptMatchQuery, matchID, player1Ready, player2Ready))
		if err != nil {
			return Match{}, fmt.Errorf("accept match: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return Match{}, fmt.Errorf("commit tx: %w", err)
		}

		return updated, nil
	}

	if duration <= 0 {
		duration = 20 * time.Minute
	}

	updated, err := scanMatch(tx.QueryRow(
		ctx,
		startMatchQuery,
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

	ct, err := tx.Exec(
		ctx,
		cancelMatchQuery,
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
	rows, err := r.db.Query(ctx, listUserMatchesQuery, userID, limit, offset)
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

	for index := range matches {
		if err := r.hydrateMatchRatingDeltas(ctx, &matches[index]); err != nil {
			return nil, err
		}
	}

	return matches, nil
}

func (r *Repository) GetCurrentMatch(ctx context.Context, userID uuid.UUID) (Match, error) {
	match, err := scanMatch(r.db.QueryRow(ctx, getCurrentMatchQuery, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Match{}, nil
		}
		return Match{}, fmt.Errorf("get current match: %w", err)
	}

	if err := r.hydrateMatchRatingDeltas(ctx, &match); err != nil {
		return Match{}, err
	}

	return match, nil
}

func (r *Repository) hydrateMatchRatingDeltas(ctx context.Context, match *Match) error {
	if match == nil ||
		match.ID == uuid.Nil ||
		!match.IsRated ||
		!match.RatingApplied ||
		match.Status != MatchStatusFinished {
		return nil
	}

	rows, err := r.db.Query(ctx, getMatchRatingDeltasQuery, match.ID, match.RatingMode)
	if err != nil {
		return fmt.Errorf("get match rating deltas: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID uuid.UUID
		var delta int
		if err := rows.Scan(&userID, &delta); err != nil {
			return fmt.Errorf("scan match rating delta: %w", err)
		}

		switch userID {
		case match.Player1ID:
			match.Player1RatingDelta = &delta
		case match.Player2ID:
			match.Player2RatingDelta = &delta
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate match rating deltas: %w", err)
	}

	return nil
}

func (r *Repository) FindQueuedOpponent(ctx context.Context, input JoinQueueInput) (*uuid.UUID, error) {
	if input.RatingMode == "" {
		input.RatingMode = "global"
	}
	if input.TaskMode == "" {
		input.TaskMode = TaskModeNormal
	}

	var opponentID uuid.UUID
	if err := r.db.QueryRow(ctx, findQueuedOpponentQuery, input.UserID, input.RatingMode, input.TaskMode, input.IsRated).Scan(&opponentID); err != nil {
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

	_, err := r.db.Exec(ctx, enqueuePlayerQuery, input.UserID, input.RatingMode, input.TaskMode, input.IsRated)
	if err != nil {
		return fmt.Errorf("enqueue player: %w", err)
	}

	return nil
}

func (r *Repository) GetQueuedPlayer(ctx context.Context, userID uuid.UUID) (*QueuedPlayer, error) {
	var player QueuedPlayer
	err := r.db.QueryRow(ctx, getQueuedPlayerQuery, userID).Scan(
		&player.UserID,
		&player.RatingMode,
		&player.TaskMode,
		&player.IsRated,
		&player.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get queued player: %w", err)
	}

	return &player, nil
}

func (r *Repository) RemoveQueuedPlayers(ctx context.Context, userIDs ...uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	_, err := r.db.Exec(ctx, removeQueuedPlayersQuery, userIDs)
	if err != nil {
		return fmt.Errorf("remove queued players: %w", err)
	}

	return nil
}

func (r *Repository) GetPublicStats(ctx context.Context) (PublicStats, error) {
	var stats PublicStats
	if err := r.db.QueryRow(ctx, publicStatsQuery).Scan(
		&stats.QueuedPlayers,
		&stats.ActiveMatchPlayers,
	); err != nil {
		return PublicStats{}, fmt.Errorf("get public match stats: %w", err)
	}
	stats.CachedAt = time.Now().UTC()

	return stats, nil
}

func (r *Repository) ListMatchSubmissions(ctx context.Context, matchID uuid.UUID) ([]MatchSubmission, error) {
	rows, err := r.db.Query(ctx, listMatchSubmissionsQuery, matchID)
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

func (r *Repository) ListMatchReplayPlayers(ctx context.Context, player1ID uuid.UUID, player2ID uuid.UUID) ([]MatchReplayPlayer, error) {
	rows, err := r.db.Query(ctx, listMatchReplayPlayersQuery, player1ID, player2ID)
	if err != nil {
		return nil, fmt.Errorf("list match replay players: %w", err)
	}
	defer rows.Close()

	items := make([]MatchReplayPlayer, 0, 2)
	for rows.Next() {
		var item MatchReplayPlayer
		if err := rows.Scan(
			&item.ID,
			&item.Username,
			&item.DisplayName,
			&item.AvatarURL,
		); err != nil {
			return nil, fmt.Errorf("scan match replay player: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate match replay players: %w", err)
	}

	return items, nil
}

func (r *Repository) ListMatchReplaySubmissions(ctx context.Context, matchID uuid.UUID) ([]MatchReplaySubmission, error) {
	rows, err := r.db.Query(ctx, listMatchReplaySubmissionsQuery, matchID)
	if err != nil {
		return nil, fmt.Errorf("list match replay submissions: %w", err)
	}
	defer rows.Close()

	items := make([]MatchReplaySubmission, 0)
	for rows.Next() {
		var item MatchReplaySubmission
		if err := rows.Scan(
			&item.ID,
			&item.MatchID,
			&item.TaskID,
			&item.UserID,
			&item.Language,
			&item.LanguageVersion,
			&item.SourceCode,
			&item.Status,
			&item.Verdict,
			&item.JudgeSubmissionID,
			&item.JudgeStatus,
			&item.ErrorMessage,
			&item.PassedTests,
			&item.TotalTests,
			&item.CompileTimeMs,
			&item.RunTimeMs,
			&item.PeakMemoryKb,
			&item.Score,
			&item.ExitCode,
			&item.CompileLog,
			&item.StdoutTruncated,
			&item.StderrTruncated,
			&item.CreatedAt,
			&item.StartedAt,
			&item.FinishedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan match replay submission: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate match replay submissions: %w", err)
	}

	return items, nil
}
func (r *Repository) FinishExpiredMatches(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.Query(ctx, listExpiredMatchesQuery, now, limit)
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

	ct, err := r.db.Exec(
		ctx,
		cancelExpiredPendingMatchesQuery,
		pendingTimeout.String(),
		limit,
		MatchStatusCancelled,
		MatchResultTypeCancelled,
		WinningReasonCancelled,
		JudgeStatusFailed,
	)
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

	_, err = tx.Exec(
		ctx,
		finishMatchQuery,
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
		_, err = tx.Exec(ctx, markMatchRatingAppliedQuery, input.MatchID)
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

	_, err = tx.Exec(ctx, markMatchRatingAppliedQuery, input.MatchID)
	if err != nil {
		return fmt.Errorf("mark rating applied: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *Repository) getMatchForUpdate(ctx context.Context, tx pgx.Tx, matchID uuid.UUID) (Match, error) {
	match, err := scanMatch(tx.QueryRow(ctx, getMatchForUpdateQuery, matchID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Match{}, ErrMatchNotFound
		}
		return Match{}, fmt.Errorf("get match for update: %w", err)
	}

	return match, nil
}

func (r *Repository) getUserRatingForUpdate(ctx context.Context, tx pgx.Tx, userID uuid.UUID, mode string) (UserRating, error) {
	var rating UserRating
	err := tx.QueryRow(ctx, getUserRatingForUpdateQuery, userID, mode).Scan(
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

	ct, err := tx.Exec(ctx, updateUserRatingAfterMatchQuery, rating.UserID, rating.Mode, newRating, wins, losses, draws)
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
	_, err := tx.Exec(
		ctx,
		insertRatingHistoryQuery,
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
		_, err := r.db.Exec(ctx, applyPlayer1SubmissionProgressQuery,
			input.MatchID,
			input.Score,
			input.IsAccepted,
			input.SolvedAt,
			input.JudgeStatus,
		)
		return err
	}

	if input.UserID == match.Player2ID {
		_, err := r.db.Exec(ctx, applyPlayer2SubmissionProgressQuery,
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
