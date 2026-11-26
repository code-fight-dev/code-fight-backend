package matches

import (
	"context"
	"errors"
	"testing"
	"time"

	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/testsupport"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryConcurrentRatedMatchFinalization(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	repo := NewRepository(db)
	player1 := createMatchTestUser(t, db, ctx, "p1")
	player2 := createMatchTestUser(t, db, ctx, "p2")
	taskID := createMatchTestTask(t, db, ctx)
	match, err := repo.CreateMatch(ctx, Match{
		ID: uuid.New(), Player1ID: player1, Player2ID: player2,
		TaskID: &taskID, TaskMode: TaskModeNormal, RatingMode: "global",
		Status: MatchStatusPending, JudgeStatus: JudgeStatusNotStarted, IsRated: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.StartMatch(ctx, match.ID, player1, time.Minute); err != nil {
		t.Fatal(err)
	}
	running, err := repo.StartMatch(ctx, match.ID, player2, time.Minute)
	if err != nil || running.ExpiresAt == nil {
		t.Fatalf("start match: %#v, %v", running, err)
	}
	reconnected, err := repo.GetCurrentMatch(ctx, player1)
	if err != nil || reconnected.ID != running.ID || reconnected.ExpiresAt == nil ||
		!reconnected.ExpiresAt.Equal(*running.ExpiresAt) {
		t.Fatalf("reconnected match: %#v, %v", reconnected, err)
	}

	start := make(chan struct{})
	type outcome struct {
		applied bool
		err     error
	}
	outcomes := make(chan outcome, 4)
	results := []FinishMatchInput{
		{MatchID: match.ID, WinnerID: &player1, ResultType: MatchResultTypePlayer1Win, WinningReason: WinningReasonAcceptedFaster},
		{MatchID: match.ID, WinnerID: &player2, ResultType: MatchResultTypePlayer2Win, WinningReason: WinningReasonAcceptedFaster},
	}
	for _, input := range []FinishMatchInput{results[0], results[1], results[0]} {
		go func() {
			<-start
			err := repo.FinishMatch(ctx, input)
			if errors.Is(err, ErrRatingAlreadyApplied) {
				outcomes <- outcome{}
				return
			}
			outcomes <- outcome{applied: err == nil, err: err}
		}()
	}
	go func() {
		<-start
		applied, err := repo.finishExpiredMatch(ctx, match.ID, running.ExpiresAt.Add(time.Second))
		outcomes <- outcome{applied: applied, err: err}
	}()
	close(start)

	applied := 0
	for range cap(outcomes) {
		outcome := <-outcomes
		if outcome.err != nil {
			t.Errorf("finalize match: %v", outcome.err)
		}
		if outcome.applied {
			applied++
		}
	}
	if applied != 1 {
		t.Fatalf("finalized %d times, want once", applied)
	}
	final, err := repo.GetMatch(ctx, match.ID)
	if err != nil || final.Status != MatchStatusFinished || !final.RatingApplied || final.ResultType == nil {
		t.Fatalf("final match: %#v, %v", final, err)
	}
	var entries, players int
	err = db.QueryRow(ctx, "select count(*), count(distinct user_id) from rating_history where match_id = $1", match.ID).
		Scan(&entries, &players)
	if err != nil || entries != 2 || players != 2 {
		t.Fatalf("rating history: %d entries for %d players, %v", entries, players, err)
	}
}

func TestRepositoryIntegrationMatchLifecycleAndQueue(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	repo := NewRepository(db)

	player1 := createMatchTestUser(t, db, ctx, "p1")
	player2 := createMatchTestUser(t, db, ctx, "p2")
	taskID := createMatchTestTask(t, db, ctx)
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "delete from matchmaking_queue where user_id = any($1)", []uuid.UUID{player1, player2})
	})
	difficulty := TaskDifficultyEasy

	match, err := repo.CreateMatch(ctx, Match{
		ID:                     uuid.New(),
		RatingMode:             "global",
		TaskMode:               TaskModeNormal,
		TaskID:                 &taskID,
		TaskDifficultySnapshot: &difficulty,
		Player1ID:              player1,
		Player2ID:              player2,
		Status:                 MatchStatusPending,
		JudgeStatus:            JudgeStatusNotStarted,
		IsRated:                false,
	})
	if err != nil {
		t.Fatalf("CreateMatch returned error: %v", err)
	}

	loaded, err := repo.GetMatch(ctx, match.ID)
	if err != nil {
		t.Fatalf("GetMatch returned error: %v", err)
	}
	if loaded.ID != match.ID || loaded.TaskID == nil || *loaded.TaskID != taskID {
		t.Fatalf("loaded match = %#v", loaded)
	}

	accepted, err := repo.StartMatch(ctx, match.ID, player1, time.Minute)
	if err != nil {
		t.Fatalf("StartMatch first player returned error: %v", err)
	}
	if !accepted.Player1Ready || accepted.Status != MatchStatusPending {
		t.Fatalf("accepted match = %#v", accepted)
	}

	running, err := repo.StartMatch(ctx, match.ID, player2, time.Minute)
	if err != nil {
		t.Fatalf("StartMatch second player returned error: %v", err)
	}
	if running.Status != MatchStatusRunning || running.ExpiresAt == nil {
		t.Fatalf("running match = %#v", running)
	}

	current, err := repo.GetCurrentMatch(ctx, player1)
	if err != nil {
		t.Fatalf("GetCurrentMatch returned error: %v", err)
	}
	if current.ID != match.ID {
		t.Fatalf("current match = %#v", current)
	}

	matches, err := repo.ListUserMatches(ctx, player1, 10, 0)
	if err != nil {
		t.Fatalf("ListUserMatches returned error: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("ListUserMatches returned no matches")
	}

	finishedAt := time.Now().UTC()
	if err := repo.FinishMatch(ctx, FinishMatchInput{
		MatchID:       match.ID,
		ResultType:    MatchResultTypeDraw,
		WinningReason: WinningReasonDraw,
		FinishedAt:    &finishedAt,
	}); err != nil {
		t.Fatalf("FinishMatch returned error: %v", err)
	}
	finished, err := repo.GetMatch(ctx, match.ID)
	if err != nil {
		t.Fatalf("GetMatch finished returned error: %v", err)
	}
	if finished.Status != MatchStatusFinished || !finished.RatingApplied {
		t.Fatalf("finished match = %#v", finished)
	}

	if err := repo.EnqueuePlayer(ctx, JoinQueueInput{UserID: player1, RatingMode: "global", TaskMode: TaskModeNormal}); err != nil {
		t.Fatalf("EnqueuePlayer returned error: %v", err)
	}
	queued, err := repo.GetQueuedPlayer(ctx, player1)
	if err != nil {
		t.Fatalf("GetQueuedPlayer returned error: %v", err)
	}
	if queued == nil || queued.UserID != player1 {
		t.Fatalf("queued player = %#v", queued)
	}
	opponent, err := repo.FindQueuedOpponent(ctx, JoinQueueInput{UserID: player2, RatingMode: "global", TaskMode: TaskModeNormal})
	if err != nil {
		t.Fatalf("FindQueuedOpponent returned error: %v", err)
	}
	if opponent == nil || *opponent != player1 {
		t.Fatalf("opponent = %v", opponent)
	}
	if err := repo.RemoveQueuedPlayers(ctx, player1); err != nil {
		t.Fatalf("RemoveQueuedPlayers returned error: %v", err)
	}
	if stats, err := repo.GetPublicStats(ctx); err != nil || stats.QueuedPlayers < 0 {
		t.Fatalf("GetPublicStats = %#v, %v", stats, err)
	}
}

func createMatchTestUser(t *testing.T, db *pgxpool.Pool, ctx context.Context, prefix string) uuid.UUID {
	t.Helper()
	suffix := uuid.NewString()
	user, err := auth.NewRepository(db).CreateUser(ctx, auth.User{
		ID:           uuid.New(),
		Email:        "matches-" + prefix + "-" + suffix + "@example.com",
		Username:     "m_" + prefix + "_" + suffix[:8],
		DisplayName:  "Match " + prefix,
		PasswordHash: "hash",
		AvatarSource: auth.AvatarSourceNone,
	})
	if err != nil {
		t.Fatalf("create fixture user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "delete from users where id = $1", user.ID)
	})
	return user.ID
}

func createMatchTestTask(t *testing.T, db *pgxpool.Pool, ctx context.Context) uuid.UUID {
	t.Helper()
	taskID := uuid.New()
	slug := "match-it-" + taskID.String()
	_, err := db.Exec(ctx, `
		insert into tasks (id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb)
		values ($1, $2, 'Match Integration Task', 'Solve it', 'easy', 1500, 131072)
	`, taskID, slug)
	if err != nil {
		t.Fatalf("insert fixture task: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "delete from matches where task_id = $1", taskID)
		_, _ = db.Exec(ctx, "delete from tasks where id = $1", taskID)
	})
	return taskID
}
