package matches

import (
	"context"
	"testing"
	"time"

	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/testsupport"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
