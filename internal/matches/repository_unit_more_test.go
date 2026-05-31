package matches

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRepositoryReadAndQueueMethodsWithFakeDB(t *testing.T) {
	now := time.Now().UTC()
	taskID := uuid.New()
	difficulty := TaskDifficultyMedium
	player1 := uuid.New()
	player2 := uuid.New()
	matchID := uuid.New()
	matchValues := fakeMatchRowValues(Match{
		ID:                     matchID,
		RatingMode:             "global",
		TaskMode:               TaskModeNormal,
		TaskID:                 &taskID,
		TaskDifficultySnapshot: &difficulty,
		Player1ID:              player1,
		Player2ID:              player2,
		Status:                 MatchStatusPending,
		JudgeStatus:            JudgeStatusNotStarted,
		CreatedAt:              now,
		UpdatedAt:              now,
	})

	db := &fakeMatchDB{
		rows: []*fakePGRows{
			{rows: [][]any{matchValues, matchValues}},
			{rows: [][]any{{matchID, matchID, taskID, player1, "go", "1.24", "finished", nil, 1, 1, 100, now, nil, nil, now}}},
			{rows: [][]any{{player1, "alice", "Alice", ""}, {player2, "bob", "Bob", "avatar"}}},
			{rows: [][]any{fakeReplaySubmissionRow(matchID, taskID, player1, now)}},
		},
		rowResults: []*fakePGRow{
			{values: matchValues},
			{values: matchValues},
			{values: []any{player2}},
			{err: pgx.ErrNoRows},
			{values: []any{player1, "global", TaskModeNormal, true, now}},
			{values: []any{3, 4}},
		},
		execTags: []pgconn.CommandTag{
			pgconn.NewCommandTag("INSERT 0 1"),
			pgconn.NewCommandTag("DELETE 1"),
		},
	}
	repo := &Repository{db: db}

	created, err := repo.CreateMatch(context.Background(), Match{ID: matchID})
	if err != nil || created.ID != matchID {
		t.Fatalf("CreateMatch = %#v, %v", created, err)
	}
	loaded, err := repo.GetMatch(context.Background(), matchID)
	if err != nil || loaded.ID != matchID {
		t.Fatalf("GetMatch = %#v, %v", loaded, err)
	}
	matches, err := repo.ListUserMatches(context.Background(), player1, 10, 0)
	if err != nil || len(matches) != 2 {
		t.Fatalf("ListUserMatches = %d, %v", len(matches), err)
	}
	opponent, err := repo.FindQueuedOpponent(context.Background(), JoinQueueInput{UserID: player1})
	if err != nil || opponent == nil || *opponent != player2 {
		t.Fatalf("FindQueuedOpponent = %v, %v", opponent, err)
	}
	opponent, err = repo.FindQueuedOpponent(context.Background(), JoinQueueInput{UserID: player1})
	if err != nil || opponent != nil {
		t.Fatalf("FindQueuedOpponent empty = %v, %v", opponent, err)
	}
	if err := repo.EnqueuePlayer(context.Background(), JoinQueueInput{UserID: player1}); err != nil {
		t.Fatalf("EnqueuePlayer returned error: %v", err)
	}
	queued, err := repo.GetQueuedPlayer(context.Background(), player1)
	if err != nil || queued == nil || queued.UserID != player1 || queued.TaskMode != TaskModeNormal {
		t.Fatalf("GetQueuedPlayer = %#v, %v", queued, err)
	}
	if err := repo.RemoveQueuedPlayers(context.Background()); err != nil {
		t.Fatalf("RemoveQueuedPlayers empty returned error: %v", err)
	}
	if err := repo.RemoveQueuedPlayers(context.Background(), player1); err != nil {
		t.Fatalf("RemoveQueuedPlayers returned error: %v", err)
	}
	stats, err := repo.GetPublicStats(context.Background())
	if err != nil || stats.QueuedPlayers != 3 || stats.ActiveMatchPlayers != 4 || stats.CachedAt.IsZero() {
		t.Fatalf("GetPublicStats = %#v, %v", stats, err)
	}
	submissions, err := repo.ListMatchSubmissions(context.Background(), matchID)
	if err != nil || len(submissions) != 1 || submissions[0].Score != 100 {
		t.Fatalf("ListMatchSubmissions = %#v, %v", submissions, err)
	}
	players, err := repo.ListMatchReplayPlayers(context.Background(), player1, player2)
	if err != nil || len(players) != 2 || players[1].AvatarURL != "avatar" {
		t.Fatalf("ListMatchReplayPlayers = %#v, %v", players, err)
	}
	replaySubmissions, err := repo.ListMatchReplaySubmissions(context.Background(), matchID)
	if err != nil || len(replaySubmissions) != 1 || replaySubmissions[0].SourceCode != "package main" {
		t.Fatalf("ListMatchReplaySubmissions = %#v, %v", replaySubmissions, err)
	}
}

func TestRepositoryGetCurrentMatchHandlesNoRows(t *testing.T) {
	repo := &Repository{db: &fakeMatchDB{rowResults: []*fakePGRow{{err: pgx.ErrNoRows}}}}

	match, err := repo.GetCurrentMatch(context.Background(), uuid.New())
	if err != nil || match.ID != uuid.Nil {
		t.Fatalf("GetCurrentMatch = %#v, %v", match, err)
	}
}

func TestRepositoryReadMethodsReturnDatabaseErrors(t *testing.T) {
	now := time.Now().UTC()
	matchID := uuid.New()
	player1 := uuid.New()
	player2 := uuid.New()
	matchValues := fakeMatchRowValues(Match{
		ID:            matchID,
		RatingMode:    "global",
		Player1ID:     player1,
		Player2ID:     player2,
		Status:        MatchStatusFinished,
		IsRated:       true,
		RatingApplied: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	dbErr := errors.New("db failed")

	tests := []struct {
		name string
		call func(*Repository) error
		repo *Repository
		want string
	}{
		{
			name: "create scan",
			repo: &Repository{db: &fakeMatchDB{rowResults: []*fakePGRow{{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.CreateMatch(context.Background(), Match{ID: matchID})
				return err
			},
			want: "create match",
		},
		{
			name: "get generic",
			repo: &Repository{db: &fakeMatchDB{rowResults: []*fakePGRow{{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.GetMatch(context.Background(), matchID)
				return err
			},
			want: "get match",
		},
		{
			name: "get hydrate",
			repo: &Repository{db: &fakeMatchDB{rowResults: []*fakePGRow{{values: matchValues}}, queryErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.GetMatch(context.Background(), matchID)
				return err
			},
			want: "get match rating deltas",
		},
		{
			name: "list query",
			repo: &Repository{db: &fakeMatchDB{queryErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.ListUserMatches(context.Background(), player1, 10, 0)
				return err
			},
			want: "list user matches",
		},
		{
			name: "list scan",
			repo: &Repository{db: &fakeMatchDB{rows: []*fakePGRows{{rows: [][]any{{matchID}}}}}},
			call: func(repo *Repository) error {
				_, err := repo.ListUserMatches(context.Background(), player1, 10, 0)
				return err
			},
			want: "scan user matches",
		},
		{
			name: "list iterate",
			repo: &Repository{db: &fakeMatchDB{rows: []*fakePGRows{{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.ListUserMatches(context.Background(), player1, 10, 0)
				return err
			},
			want: "iterate user matches",
		},
		{
			name: "current generic",
			repo: &Repository{db: &fakeMatchDB{rowResults: []*fakePGRow{{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.GetCurrentMatch(context.Background(), player1)
				return err
			},
			want: "get current match",
		},
		{
			name: "queued generic",
			repo: &Repository{db: &fakeMatchDB{rowResults: []*fakePGRow{{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.GetQueuedPlayer(context.Background(), player1)
				return err
			},
			want: "get queued player",
		},
		{
			name: "enqueue",
			repo: &Repository{db: &fakeMatchDB{execErr: dbErr}},
			call: func(repo *Repository) error {
				return repo.EnqueuePlayer(context.Background(), JoinQueueInput{UserID: player1})
			},
			want: "enqueue player",
		},
		{
			name: "remove queued",
			repo: &Repository{db: &fakeMatchDB{execErr: dbErr}},
			call: func(repo *Repository) error {
				return repo.RemoveQueuedPlayers(context.Background(), player1)
			},
			want: "remove queued players",
		},
		{
			name: "public stats",
			repo: &Repository{db: &fakeMatchDB{rowResults: []*fakePGRow{{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.GetPublicStats(context.Background())
				return err
			},
			want: "get public match stats",
		},
		{
			name: "cancel expired",
			repo: &Repository{db: &fakeMatchDB{execErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.CancelExpiredPendingMatches(context.Background(), now, 0, 0)
				return err
			},
			want: "cancel expired pending matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(tt.repo)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestRepositoryListReplayAndSubmissionErrorBranches(t *testing.T) {
	dbErr := errors.New("db failed")
	matchID := uuid.New()

	tests := []struct {
		name string
		call func(*Repository) error
		repo *Repository
		want string
	}{
		{
			name: "submissions query",
			repo: &Repository{db: &fakeMatchDB{queryErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.ListMatchSubmissions(context.Background(), matchID)
				return err
			},
			want: "list match submissions",
		},
		{
			name: "submissions scan",
			repo: &Repository{db: &fakeMatchDB{rows: []*fakePGRows{{rows: [][]any{{matchID}}}}}},
			call: func(repo *Repository) error {
				_, err := repo.ListMatchSubmissions(context.Background(), matchID)
				return err
			},
			want: "scan match submission",
		},
		{
			name: "submissions iterate",
			repo: &Repository{db: &fakeMatchDB{rows: []*fakePGRows{{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.ListMatchSubmissions(context.Background(), matchID)
				return err
			},
			want: "iterate match submissions",
		},
		{
			name: "players query",
			repo: &Repository{db: &fakeMatchDB{queryErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.ListMatchReplayPlayers(context.Background(), uuid.New(), uuid.New())
				return err
			},
			want: "list match replay players",
		},
		{
			name: "players scan",
			repo: &Repository{db: &fakeMatchDB{rows: []*fakePGRows{{rows: [][]any{{matchID}}}}}},
			call: func(repo *Repository) error {
				_, err := repo.ListMatchReplayPlayers(context.Background(), uuid.New(), uuid.New())
				return err
			},
			want: "scan match replay player",
		},
		{
			name: "players iterate",
			repo: &Repository{db: &fakeMatchDB{rows: []*fakePGRows{{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.ListMatchReplayPlayers(context.Background(), uuid.New(), uuid.New())
				return err
			},
			want: "iterate match replay players",
		},
		{
			name: "replay submissions query",
			repo: &Repository{db: &fakeMatchDB{queryErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.ListMatchReplaySubmissions(context.Background(), matchID)
				return err
			},
			want: "list match replay submissions",
		},
		{
			name: "replay submissions scan",
			repo: &Repository{db: &fakeMatchDB{rows: []*fakePGRows{{rows: [][]any{{matchID}}}}}},
			call: func(repo *Repository) error {
				_, err := repo.ListMatchReplaySubmissions(context.Background(), matchID)
				return err
			},
			want: "scan match replay submission",
		},
		{
			name: "replay submissions iterate",
			repo: &Repository{db: &fakeMatchDB{rows: []*fakePGRows{{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.ListMatchReplaySubmissions(context.Background(), matchID)
				return err
			},
			want: "iterate match replay submissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(tt.repo)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestRepositoryApplySubmissionProgressValidationAndParticipants(t *testing.T) {
	matchID := uuid.New()
	player1 := uuid.New()
	player2 := uuid.New()
	now := time.Now().UTC()
	matchValues := fakeMatchRowValues(Match{
		ID:          matchID,
		RatingMode:  "global",
		TaskMode:    TaskModeNormal,
		Player1ID:   player1,
		Player2ID:   player2,
		Status:      MatchStatusRunning,
		JudgeStatus: JudgeStatusRunning,
		CreatedAt:   now,
		UpdatedAt:   now,
	})

	repo := &Repository{db: &fakeMatchDB{
		rowResults: []*fakePGRow{{values: matchValues}, {values: matchValues}, {values: matchValues}},
		execTags:   []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 1"), pgconn.NewCommandTag("UPDATE 1")},
	}}
	if err := repo.ApplySubmissionProgress(context.Background(), ApplySubmissionProgressInput{}); err == nil {
		t.Fatal("empty input returned nil error")
	}
	if err := repo.ApplySubmissionProgress(context.Background(), ApplySubmissionProgressInput{MatchID: matchID}); err == nil {
		t.Fatal("empty user returned nil error")
	}
	if err := repo.ApplySubmissionProgress(context.Background(), ApplySubmissionProgressInput{MatchID: matchID, UserID: player1, Score: 10}); err != nil {
		t.Fatalf("player1 progress returned error: %v", err)
	}
	if err := repo.ApplySubmissionProgress(context.Background(), ApplySubmissionProgressInput{MatchID: matchID, UserID: player2, Score: 10}); err != nil {
		t.Fatalf("player2 progress returned error: %v", err)
	}
	if err := repo.ApplySubmissionProgress(context.Background(), ApplySubmissionProgressInput{MatchID: matchID, UserID: uuid.New()}); !errors.Is(err, ErrSubmissionUserNotInMatch) {
		t.Fatalf("non participant error = %v", err)
	}
}

func TestRepositoryScanAndCalculationHelpers(t *testing.T) {
	now := time.Now().UTC()
	matchID := uuid.New()
	player1 := uuid.New()
	player2 := uuid.New()
	match, err := scanMatch(&fakePGRow{values: fakeMatchRowValues(Match{
		ID:          matchID,
		RatingMode:  "global",
		TaskMode:    TaskModeHard,
		Player1ID:   player1,
		Player2ID:   player2,
		Status:      MatchStatusRunning,
		JudgeStatus: JudgeStatusRunning,
		CreatedAt:   now,
		UpdatedAt:   now,
	})})
	if err != nil || match.ID != matchID || match.TaskMode != TaskModeHard {
		t.Fatalf("scanMatch = %#v, %v", match, err)
	}

	if got := calculateNewRating(1000, 1000, 0, 1); got != 1020 {
		t.Fatalf("new rating = %d, want 1020", got)
	}
	if got := kFactor(30); got != 20 {
		t.Fatalf("kFactor = %d, want 20", got)
	}
	if _, _, err := eloResultsForMatch(Match{}, FinishMatchInput{ResultType: MatchResultTypeCancelled}); !errors.Is(err, ErrInvalidMatchStatus) {
		t.Fatalf("eloResultsForMatch error = %v", err)
	}
}

func TestRepositoryMatchTransactionsWithFakeTx(t *testing.T) {
	now := time.Now().UTC()
	matchID := uuid.New()
	player1 := uuid.New()
	player2 := uuid.New()
	pending := Match{
		ID:          matchID,
		RatingMode:  "global",
		TaskMode:    TaskModeNormal,
		Player1ID:   player1,
		Player2ID:   player2,
		Status:      MatchStatusPending,
		JudgeStatus: JudgeStatusNotStarted,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	accepted := pending
	accepted.Player1Ready = true
	ready := accepted
	running := pending
	running.Player1Ready = true
	running.Player2Ready = true
	running.Status = MatchStatusRunning
	running.JudgeStatus = JudgeStatusRunning
	finished := pending
	finished.IsRated = false

	tx := &fakeMatchTx{
		rowResults: []*fakePGRow{
			{values: fakeMatchRowValues(pending)},
			{values: fakeMatchRowValues(accepted)},
			{values: fakeMatchRowValues(ready)},
			{values: fakeMatchRowValues(running)},
			{values: fakeMatchRowValues(pending)},
			{values: fakeMatchRowValues(finished)},
		},
		execTags: []pgconn.CommandTag{
			pgconn.NewCommandTag("UPDATE 1"),
			pgconn.NewCommandTag("UPDATE 1"),
			pgconn.NewCommandTag("UPDATE 1"),
		},
	}
	repo := &Repository{db: &fakeMatchDB{tx: tx}}

	acceptedMatch, err := repo.StartMatch(context.Background(), matchID, player1, time.Minute)
	if err != nil || !acceptedMatch.Player1Ready || acceptedMatch.Status != MatchStatusPending {
		t.Fatalf("first StartMatch = %#v, %v", acceptedMatch, err)
	}
	runningMatch, err := repo.StartMatch(context.Background(), matchID, player2, time.Minute)
	if err != nil || runningMatch.Status != MatchStatusRunning {
		t.Fatalf("second StartMatch = %#v, %v", runningMatch, err)
	}
	if err := repo.CancelMatch(context.Background(), matchID); err != nil {
		t.Fatalf("CancelMatch returned error: %v", err)
	}
	if err := repo.FinishMatch(context.Background(), FinishMatchInput{
		MatchID:       matchID,
		ResultType:    MatchResultTypeDraw,
		WinningReason: WinningReasonDraw,
	}); err != nil {
		t.Fatalf("FinishMatch returned error: %v", err)
	}
	if tx.commits != 4 || tx.rollbacks != 4 {
		t.Fatalf("transaction counts commits=%d rollbacks=%d", tx.commits, tx.rollbacks)
	}
}

func TestRepositoryStartMatchValidationBranches(t *testing.T) {
	now := time.Now().UTC()
	matchID := uuid.New()
	player1 := uuid.New()
	player2 := uuid.New()
	base := Match{
		ID:          matchID,
		Player1ID:   player1,
		Player2ID:   player2,
		Status:      MatchStatusPending,
		JudgeStatus: JudgeStatusNotStarted,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	tests := []struct {
		name  string
		match Match
		actor uuid.UUID
		want  error
	}{
		{name: "already running", match: func() Match { m := base; m.Status = MatchStatusRunning; return m }(), actor: player1, want: ErrMatchAlreadyStarted},
		{name: "already finished", match: func() Match { m := base; m.Status = MatchStatusFinished; return m }(), actor: player1, want: ErrMatchAlreadyFinished},
		{name: "cancelled", match: func() Match { m := base; m.Status = MatchStatusCancelled; return m }(), actor: player1, want: ErrMatchCancelled},
		{name: "non participant", match: base, actor: uuid.New(), want: ErrSubmissionUserNotInMatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{rowResults: []*fakePGRow{{values: fakeMatchRowValues(tt.match)}}}}}
			_, err := repo.StartMatch(context.Background(), matchID, tt.actor, time.Minute)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestRepositoryTransactionErrorBranches(t *testing.T) {
	now := time.Now().UTC()
	matchID := uuid.New()
	player1 := uuid.New()
	player2 := uuid.New()
	running := Match{
		ID:          matchID,
		RatingMode:  "global",
		Player1ID:   player1,
		Player2ID:   player2,
		Status:      MatchStatusRunning,
		JudgeStatus: JudgeStatusRunning,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	pending := running
	pending.Status = MatchStatusPending
	dbErr := errors.New("db failed")

	tests := []struct {
		name string
		call func(*Repository) error
		repo *Repository
		want error
	}{
		{
			name: "start begin",
			repo: &Repository{db: &fakeMatchDB{beginErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.StartMatch(context.Background(), matchID, player1, time.Minute)
				return err
			},
			want: dbErr,
		},
		{
			name: "start commit",
			repo: &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
				rowResults: []*fakePGRow{{values: fakeMatchRowValues(pending)}, {values: fakeMatchRowValues(pending)}},
				commitErr:  dbErr,
			}}},
			call: func(repo *Repository) error {
				_, err := repo.StartMatch(context.Background(), matchID, player1, time.Minute)
				return err
			},
			want: dbErr,
		},
		{
			name: "cancel begin",
			repo: &Repository{db: &fakeMatchDB{beginErr: dbErr}},
			call: func(repo *Repository) error {
				return repo.CancelMatch(context.Background(), matchID)
			},
			want: dbErr,
		},
		{
			name: "cancel exec",
			repo: &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
				rowResults: []*fakePGRow{{values: fakeMatchRowValues(pending)}},
				execErr:    dbErr,
			}}},
			call: func(repo *Repository) error {
				return repo.CancelMatch(context.Background(), matchID)
			},
			want: dbErr,
		},
		{
			name: "cancel missing update",
			repo: &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
				rowResults: []*fakePGRow{{values: fakeMatchRowValues(pending)}},
			}}},
			call: func(repo *Repository) error {
				return repo.CancelMatch(context.Background(), matchID)
			},
			want: ErrMatchNotFound,
		},
		{
			name: "cancel commit",
			repo: &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
				rowResults: []*fakePGRow{{values: fakeMatchRowValues(pending)}},
				execTags:   []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 1")},
				commitErr:  dbErr,
			}}},
			call: func(repo *Repository) error {
				return repo.CancelMatch(context.Background(), matchID)
			},
			want: dbErr,
		},
		{
			name: "finish begin",
			repo: &Repository{db: &fakeMatchDB{beginErr: dbErr}},
			call: func(repo *Repository) error {
				return repo.FinishMatch(context.Background(), FinishMatchInput{MatchID: matchID, ResultType: MatchResultTypeDraw, WinningReason: WinningReasonDraw})
			},
			want: dbErr,
		},
		{
			name: "finish update result",
			repo: &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
				rowResults: []*fakePGRow{{values: fakeMatchRowValues(running)}},
				execErr:    dbErr,
			}}},
			call: func(repo *Repository) error {
				return repo.FinishMatch(context.Background(), FinishMatchInput{MatchID: matchID, ResultType: MatchResultTypeDraw, WinningReason: WinningReasonDraw})
			},
			want: dbErr,
		},
		{
			name: "finish unrated mark",
			repo: &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
				rowResults: []*fakePGRow{{values: fakeMatchRowValues(running)}},
				execTags:   []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 1")},
				execErr:    dbErr,
				execErrAt:  2,
			}}},
			call: func(repo *Repository) error {
				return repo.FinishMatch(context.Background(), FinishMatchInput{MatchID: matchID, ResultType: MatchResultTypeDraw, WinningReason: WinningReasonDraw})
			},
			want: dbErr,
		},
		{
			name: "finish unrated commit",
			repo: &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
				rowResults: []*fakePGRow{{values: fakeMatchRowValues(running)}},
				execTags:   []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 1"), pgconn.NewCommandTag("UPDATE 1")},
				commitErr:  dbErr,
			}}},
			call: func(repo *Repository) error {
				return repo.FinishMatch(context.Background(), FinishMatchInput{MatchID: matchID, ResultType: MatchResultTypeDraw, WinningReason: WinningReasonDraw})
			},
			want: dbErr,
		},
		{
			name: "finish rated missing rating",
			repo: &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
				rowResults: []*fakePGRow{{values: fakeMatchRowValues(func() Match {
					m := running
					m.IsRated = true
					return m
				}())}},
				execTags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 1")},
			}}},
			call: func(repo *Repository) error {
				return repo.FinishMatch(context.Background(), FinishMatchInput{MatchID: matchID, ResultType: MatchResultTypeDraw, WinningReason: WinningReasonDraw})
			},
			want: ErrRatingNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(tt.repo)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestRepositoryFinishRatedMatchAppliesRatings(t *testing.T) {
	now := time.Now().UTC()
	matchID := uuid.New()
	player1 := uuid.New()
	player2 := uuid.New()
	match := Match{
		ID:          matchID,
		RatingMode:  "global",
		TaskMode:    TaskModeNormal,
		Player1ID:   player1,
		Player2ID:   player2,
		Status:      MatchStatusRunning,
		JudgeStatus: JudgeStatusRunning,
		IsRated:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	tx := &fakeMatchTx{
		rowResults: []*fakePGRow{
			{values: fakeMatchRowValues(match)},
			{values: []any{player1, "global", 1000, 3, 1, 1, 1, now, now}},
			{values: []any{player2, "global", 1200, 31, 15, 10, 6, now, now}},
		},
		execTags: []pgconn.CommandTag{
			pgconn.NewCommandTag("UPDATE 1"),
			pgconn.NewCommandTag("UPDATE 1"),
			pgconn.NewCommandTag("UPDATE 1"),
			pgconn.NewCommandTag("INSERT 0 1"),
			pgconn.NewCommandTag("INSERT 0 1"),
			pgconn.NewCommandTag("UPDATE 1"),
		},
	}
	repo := &Repository{db: &fakeMatchDB{tx: tx}}
	winnerID := player1

	if err := repo.FinishMatch(context.Background(), FinishMatchInput{
		MatchID:       matchID,
		WinnerID:      &winnerID,
		ResultType:    MatchResultTypePlayer1Win,
		WinningReason: WinningReasonAcceptedFaster,
	}); err != nil {
		t.Fatalf("FinishMatch returned error: %v", err)
	}
	if tx.commits != 1 || tx.rollbacks != 1 {
		t.Fatalf("transaction counts commits=%d rollbacks=%d", tx.commits, tx.rollbacks)
	}
}

func TestRepositoryGetMatchHydratesRatingDeltas(t *testing.T) {
	now := time.Now().UTC()
	matchID := uuid.New()
	player1 := uuid.New()
	player2 := uuid.New()
	resultType := MatchResultTypePlayer1Win
	match := Match{
		ID:            matchID,
		RatingMode:    "global",
		Player1ID:     player1,
		Player2ID:     player2,
		Status:        MatchStatusFinished,
		JudgeStatus:   JudgeStatusFinished,
		ResultType:    &resultType,
		IsRated:       true,
		RatingApplied: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	repo := &Repository{db: &fakeMatchDB{
		rowResults: []*fakePGRow{{values: fakeMatchRowValues(match)}},
		rows:       []*fakePGRows{{rows: [][]any{{player1, 21}, {player2, -21}}}},
	}}

	got, err := repo.GetMatch(context.Background(), matchID)
	if err != nil {
		t.Fatalf("GetMatch returned error: %v", err)
	}
	if got.Player1RatingDelta == nil || *got.Player1RatingDelta != 21 ||
		got.Player2RatingDelta == nil || *got.Player2RatingDelta != -21 {
		t.Fatalf("rating deltas = %v/%v", got.Player1RatingDelta, got.Player2RatingDelta)
	}
}

func TestRepositoryIngestReplayEvents(t *testing.T) {
	matchID := uuid.New()
	userID := uuid.New()
	tx := &fakeMatchTx{
		rowResults: []*fakePGRow{{values: []any{int64(0), int64(0)}}},
		execTags: []pgconn.CommandTag{
			pgconn.NewCommandTag("INSERT 0 1"),
			pgconn.NewCommandTag("INSERT 0 1"),
			pgconn.NewCommandTag("INSERT 0 1"),
			pgconn.NewCommandTag("UPDATE 1"),
		},
	}
	repo := &Repository{db: &fakeMatchDB{tx: tx}}

	result, err := repo.IngestMatchReplayEvents(context.Background(), MatchReplayEventsIngestInput{
		MatchID: matchID,
		UserID:  userID,
		Events: []MatchReplayEventIngestItem{
			{Seq: 1, TMs: 0, EventType: "insert", Payload: json.RawMessage(`{"text":"a"}`)},
			{Seq: 2, TMs: 5, EventType: "cursor", Payload: json.RawMessage(`{"line":1}`)},
		},
	})
	if err != nil {
		t.Fatalf("IngestMatchReplayEvents returned error: %v", err)
	}
	if result.Ingested != 2 || result.LastSeq != 2 || result.ExpectedSeq != 3 {
		t.Fatalf("result = %#v", result)
	}
}

func TestRepositoryIngestReplayEventsDeduplicatesAndRejectsConflict(t *testing.T) {
	matchID := uuid.New()
	userID := uuid.New()
	repo := &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
		rowResults: []*fakePGRow{
			{values: []any{int64(3), int64(10)}},
			{values: []any{true}},
		},
		execTags: []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 0 1")},
	}}}

	result, err := repo.IngestMatchReplayEvents(context.Background(), MatchReplayEventsIngestInput{
		MatchID: matchID,
		UserID:  userID,
		Events:  []MatchReplayEventIngestItem{{Seq: 2, TMs: 5, EventType: "insert", Payload: json.RawMessage(`{"text":"a"}`)}},
	})
	if err != nil {
		t.Fatalf("deduplicate returned error: %v", err)
	}
	if result.Deduplicated != 1 || result.LastSeq != 3 {
		t.Fatalf("deduplicate result = %#v", result)
	}

	repo = &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
		rowResults: []*fakePGRow{
			{values: []any{int64(3), int64(10)}},
			{values: []any{false}},
			{values: []any{true}},
		},
		execTags: []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 0 1")},
	}}}
	_, err = repo.IngestMatchReplayEvents(context.Background(), MatchReplayEventsIngestInput{
		MatchID: matchID,
		UserID:  userID,
		Events:  []MatchReplayEventIngestItem{{Seq: 2, TMs: 5, EventType: "insert", Payload: json.RawMessage(`{"text":"b"}`)}},
	})
	if !errors.Is(err, ErrReplaySeqConflict) {
		t.Fatalf("conflict error = %v", err)
	}
}

func TestRepositoryIngestReplaySnapshot(t *testing.T) {
	matchID := uuid.New()
	userID := uuid.New()
	repo := &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
		rowResults: []*fakePGRow{{values: []any{int64(3), int64(20)}}},
		execTags: []pgconn.CommandTag{
			pgconn.NewCommandTag("INSERT 0 1"),
			pgconn.NewCommandTag("INSERT 0 1"),
			pgconn.NewCommandTag("UPDATE 1"),
		},
	}}}

	result, err := repo.IngestMatchReplaySnapshot(context.Background(), MatchReplaySnapshotIngestInput{
		MatchID:    matchID,
		UserID:     userID,
		Seq:        3,
		TMs:        20,
		Language:   "go",
		SourceCode: "package main",
	})
	if err != nil {
		t.Fatalf("IngestMatchReplaySnapshot returned error: %v", err)
	}
	if !result.Inserted || result.LastSeq != 3 || result.ExpectedSeq != 4 {
		t.Fatalf("snapshot result = %#v", result)
	}

	repo = &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
		rowResults: []*fakePGRow{{values: []any{int64(2), int64(20)}}},
		execTags:   []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 0 1")},
	}}}
	_, err = repo.IngestMatchReplaySnapshot(context.Background(), MatchReplaySnapshotIngestInput{
		MatchID: matchID,
		UserID:  userID,
		Seq:     3,
	})
	if !errors.Is(err, ErrReplaySnapshotAheadOfEvents) {
		t.Fatalf("ahead snapshot error = %v", err)
	}
}

func TestRepositoryListReplayTimelineRows(t *testing.T) {
	matchID := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()
	repo := &Repository{db: &fakeMatchDB{rows: []*fakePGRows{
		{rows: [][]any{{matchID, userID, int64(1), int64(0), "insert", json.RawMessage(`{"text":"a"}`), now}}},
		{rows: [][]any{{matchID, userID, int64(1), int64(0), "go", "package main", now}}},
	}}}

	events, err := repo.ListMatchReplayTimelineEvents(context.Background(), matchID)
	if err != nil || len(events) != 1 || events[0].EventType != "insert" {
		t.Fatalf("events = %#v, %v", events, err)
	}
	snapshots, err := repo.ListMatchReplayTimelineSnapshots(context.Background(), matchID)
	if err != nil || len(snapshots) != 1 || snapshots[0].Language != "go" {
		t.Fatalf("snapshots = %#v, %v", snapshots, err)
	}
}

func TestRepositoryReplayCaptureErrorBranches(t *testing.T) {
	matchID := uuid.New()
	userID := uuid.New()
	dbErr := errors.New("db failed")

	tests := []struct {
		name string
		repo *Repository
		call func(*Repository) error
		want string
	}{
		{
			name: "events begin",
			repo: &Repository{db: &fakeMatchDB{beginErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.IngestMatchReplayEvents(context.Background(), MatchReplayEventsIngestInput{MatchID: matchID, UserID: userID})
				return err
			},
			want: "begin tx",
		},
		{
			name: "events ensure",
			repo: &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{execErr: dbErr}}},
			call: func(repo *Repository) error {
				_, err := repo.IngestMatchReplayEvents(context.Background(), MatchReplayEventsIngestInput{MatchID: matchID, UserID: userID})
				return err
			},
			want: "ensure replay session",
		},
		{
			name: "events state missing",
			repo: &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{execTags: []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 0 1")}}}},
			call: func(repo *Repository) error {
				_, err := repo.IngestMatchReplayEvents(context.Background(), MatchReplayEventsIngestInput{MatchID: matchID, UserID: userID})
				return err
			},
			want: ErrMatchNotFound.Error(),
		},
		{
			name: "events insert",
			repo: &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
				rowResults: []*fakePGRow{{values: []any{int64(0), int64(0)}}},
				execTags:   []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 0 1")},
				execErr:    dbErr,
				execErrAt:  2,
			}}},
			call: func(repo *Repository) error {
				_, err := repo.IngestMatchReplayEvents(context.Background(), MatchReplayEventsIngestInput{
					MatchID: matchID,
					UserID:  userID,
					Events:  []MatchReplayEventIngestItem{{Seq: 1, TMs: 0, EventType: "insert", Payload: []byte(`{}`)}},
				})
				return err
			},
			want: "insert replay event",
		},
		{
			name: "events update",
			repo: &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
				rowResults: []*fakePGRow{{values: []any{int64(0), int64(0)}}},
				execTags:   []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 0 1"), pgconn.NewCommandTag("INSERT 0 1")},
				execErr:    dbErr,
				execErrAt:  3,
			}}},
			call: func(repo *Repository) error {
				_, err := repo.IngestMatchReplayEvents(context.Background(), MatchReplayEventsIngestInput{
					MatchID: matchID,
					UserID:  userID,
					Events:  []MatchReplayEventIngestItem{{Seq: 1, TMs: 0, EventType: "insert", Payload: []byte(`{}`)}},
				})
				return err
			},
			want: "update replay session after events",
		},
		{
			name: "events commit",
			repo: &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
				rowResults: []*fakePGRow{{values: []any{int64(0), int64(0)}}},
				execTags: []pgconn.CommandTag{
					pgconn.NewCommandTag("INSERT 0 1"),
					pgconn.NewCommandTag("INSERT 0 1"),
					pgconn.NewCommandTag("UPDATE 1"),
				},
				commitErr: dbErr,
			}}},
			call: func(repo *Repository) error {
				_, err := repo.IngestMatchReplayEvents(context.Background(), MatchReplayEventsIngestInput{
					MatchID: matchID,
					UserID:  userID,
					Events:  []MatchReplayEventIngestItem{{Seq: 1, TMs: 0, EventType: "insert", Payload: []byte(`{}`)}},
				})
				return err
			},
			want: "commit tx",
		},
		{
			name: "snapshot insert",
			repo: &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
				rowResults: []*fakePGRow{{values: []any{int64(1), int64(0)}}},
				execTags:   []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 0 1")},
				execErr:    dbErr,
				execErrAt:  2,
			}}},
			call: func(repo *Repository) error {
				_, err := repo.IngestMatchReplaySnapshot(context.Background(), MatchReplaySnapshotIngestInput{MatchID: matchID, UserID: userID, Seq: 1})
				return err
			},
			want: "insert replay snapshot",
		},
		{
			name: "snapshot update",
			repo: &Repository{db: &fakeMatchDB{tx: &fakeMatchTx{
				rowResults: []*fakePGRow{{values: []any{int64(1), int64(0)}}},
				execTags:   []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 0 1"), pgconn.NewCommandTag("INSERT 0 1")},
				execErr:    dbErr,
				execErrAt:  3,
			}}},
			call: func(repo *Repository) error {
				_, err := repo.IngestMatchReplaySnapshot(context.Background(), MatchReplaySnapshotIngestInput{MatchID: matchID, UserID: userID, Seq: 1})
				return err
			},
			want: "update replay session after snapshot",
		},
		{
			name: "timeline events query",
			repo: &Repository{db: &fakeMatchDB{queryErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.ListMatchReplayTimelineEvents(context.Background(), matchID)
				return err
			},
			want: "list replay timeline events",
		},
		{
			name: "timeline events scan",
			repo: &Repository{db: &fakeMatchDB{rows: []*fakePGRows{{rows: [][]any{{matchID}}}}}},
			call: func(repo *Repository) error {
				_, err := repo.ListMatchReplayTimelineEvents(context.Background(), matchID)
				return err
			},
			want: "scan replay timeline event",
		},
		{
			name: "timeline events iterate",
			repo: &Repository{db: &fakeMatchDB{rows: []*fakePGRows{{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.ListMatchReplayTimelineEvents(context.Background(), matchID)
				return err
			},
			want: "iterate replay timeline events",
		},
		{
			name: "timeline snapshots query",
			repo: &Repository{db: &fakeMatchDB{queryErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.ListMatchReplayTimelineSnapshots(context.Background(), matchID)
				return err
			},
			want: "list replay timeline snapshots",
		},
		{
			name: "timeline snapshots scan",
			repo: &Repository{db: &fakeMatchDB{rows: []*fakePGRows{{rows: [][]any{{matchID}}}}}},
			call: func(repo *Repository) error {
				_, err := repo.ListMatchReplayTimelineSnapshots(context.Background(), matchID)
				return err
			},
			want: "scan replay timeline snapshot",
		},
		{
			name: "timeline snapshots iterate",
			repo: &Repository{db: &fakeMatchDB{rows: []*fakePGRows{{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.ListMatchReplayTimelineSnapshots(context.Background(), matchID)
				return err
			},
			want: "iterate replay timeline snapshots",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(tt.repo)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestRepositoryCancelExpiredPendingMatches(t *testing.T) {
	repo := &Repository{db: &fakeMatchDB{execTags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 3")}}}

	count, err := repo.CancelExpiredPendingMatches(context.Background(), time.Now().UTC(), 0, 0)
	if err != nil {
		t.Fatalf("CancelExpiredPendingMatches returned error: %v", err)
	}
	if count != 3 {
		t.Fatalf("cancelled count = %d, want 3", count)
	}
}

func TestRepositoryFinishExpiredMatches(t *testing.T) {
	now := time.Now().UTC()
	matchID := uuid.New()
	player1 := uuid.New()
	player2 := uuid.New()
	match := Match{
		ID:           matchID,
		RatingMode:   "global",
		Player1ID:    player1,
		Player2ID:    player2,
		Status:       MatchStatusRunning,
		JudgeStatus:  JudgeStatusRunning,
		IsRated:      false,
		Player1Score: 5,
		Player2Score: 2,
		CreatedAt:    now.Add(-time.Minute),
		UpdatedAt:    now.Add(-time.Minute),
	}
	tx := &fakeMatchTx{
		rowResults: []*fakePGRow{{values: fakeMatchRowValues(match)}},
		execTags: []pgconn.CommandTag{
			pgconn.NewCommandTag("UPDATE 1"),
			pgconn.NewCommandTag("UPDATE 1"),
		},
	}
	repo := &Repository{db: &fakeMatchDB{
		rows:       []*fakePGRows{{rows: [][]any{{matchID}}}},
		rowResults: []*fakePGRow{{values: fakeMatchRowValues(match)}},
		tx:         tx,
	}}

	count, err := repo.FinishExpiredMatches(context.Background(), now, 0)
	if err != nil {
		t.Fatalf("FinishExpiredMatches returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("finished count = %d, want 1", count)
	}
}

func TestRepositoryFinishExpiredMatchIgnoresAlreadyTerminalErrors(t *testing.T) {
	now := time.Now().UTC()
	matchID := uuid.New()
	match := Match{
		ID:          matchID,
		RatingMode:  "global",
		Player1ID:   uuid.New(),
		Player2ID:   uuid.New(),
		Status:      MatchStatusRunning,
		JudgeStatus: JudgeStatusRunning,
		IsRated:     false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	repo := &Repository{db: &fakeMatchDB{
		rowResults: []*fakePGRow{{values: fakeMatchRowValues(match)}},
		tx: &fakeMatchTx{
			rowResults: []*fakePGRow{{values: fakeMatchRowValues(func() Match {
				m := match
				m.Status = MatchStatusFinished
				m.RatingApplied = true
				return m
			}())}},
		},
	}}

	ok, err := repo.finishExpiredMatch(context.Background(), matchID, now)
	if err != nil || ok {
		t.Fatalf("finishExpiredMatch = %v, %v", ok, err)
	}
}

type fakeMatchDB struct {
	rowResults []*fakePGRow
	rows       []*fakePGRows
	execTags   []pgconn.CommandTag
	tx         *fakeMatchTx
	execErr    error
	queryErr   error
	beginErr   error
}

func (db *fakeMatchDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	if db.execErr != nil {
		return pgconn.CommandTag{}, db.execErr
	}
	if len(db.execTags) == 0 {
		return pgconn.NewCommandTag("UPDATE 0"), nil
	}
	tag := db.execTags[0]
	db.execTags = db.execTags[1:]
	return tag, nil
}

func (db *fakeMatchDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	if db.queryErr != nil {
		return nil, db.queryErr
	}
	if len(db.rows) == 0 {
		return &fakePGRows{}, nil
	}
	rows := db.rows[0]
	db.rows = db.rows[1:]
	return rows, nil
}

func (db *fakeMatchDB) QueryRow(context.Context, string, ...any) pgx.Row {
	if len(db.rowResults) == 0 {
		return &fakePGRow{err: pgx.ErrNoRows}
	}
	row := db.rowResults[0]
	db.rowResults = db.rowResults[1:]
	return row
}

func (db *fakeMatchDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	if db.beginErr != nil {
		return nil, db.beginErr
	}
	if db.tx == nil {
		return nil, errors.New("tx not configured")
	}
	return db.tx, nil
}

type fakeMatchTx struct {
	rowResults []*fakePGRow
	rows       []*fakePGRows
	execTags   []pgconn.CommandTag
	commits    int
	rollbacks  int
	commitErr  error
	execErr    error
	execErrAt  int
	execCalls  int
}

func (tx *fakeMatchTx) Begin(context.Context) (pgx.Tx, error) { return tx, nil }
func (tx *fakeMatchTx) Commit(context.Context) error {
	tx.commits++
	return tx.commitErr
}
func (tx *fakeMatchTx) Rollback(context.Context) error {
	tx.rollbacks++
	return nil
}
func (tx *fakeMatchTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (tx *fakeMatchTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (tx *fakeMatchTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *fakeMatchTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (tx *fakeMatchTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	tx.execCalls++
	if tx.execErr != nil && (tx.execErrAt == 0 || tx.execCalls == tx.execErrAt) {
		return pgconn.CommandTag{}, tx.execErr
	}
	if len(tx.execTags) == 0 {
		return pgconn.NewCommandTag("UPDATE 0"), nil
	}
	tag := tx.execTags[0]
	tx.execTags = tx.execTags[1:]
	return tag, nil
}
func (tx *fakeMatchTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	if len(tx.rows) == 0 {
		return &fakePGRows{}, nil
	}
	rows := tx.rows[0]
	tx.rows = tx.rows[1:]
	return rows, nil
}
func (tx *fakeMatchTx) QueryRow(context.Context, string, ...any) pgx.Row {
	if len(tx.rowResults) == 0 {
		return &fakePGRow{err: pgx.ErrNoRows}
	}
	row := tx.rowResults[0]
	tx.rowResults = tx.rowResults[1:]
	return row
}
func (tx *fakeMatchTx) Conn() *pgx.Conn { return nil }

type fakePGRow struct {
	values []any
	err    error
}

func (r *fakePGRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("scan destination count mismatch")
	}
	for i := range dest {
		assignScanValue(dest[i], r.values[i])
	}
	return nil
}

type fakePGRows struct {
	rows   [][]any
	index  int
	err    error
	closed bool
}

func (r *fakePGRows) Close() {
	r.closed = true
}

func (r *fakePGRows) Err() error {
	return r.err
}

func (r *fakePGRows) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT")
}

func (r *fakePGRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *fakePGRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *fakePGRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.rows) {
		return errors.New("scan before next")
	}
	values := r.rows[r.index-1]
	if len(dest) != len(values) {
		return errors.New("scan destination count mismatch")
	}
	for i := range dest {
		assignScanValue(dest[i], values[i])
	}
	return nil
}

func (r *fakePGRows) Values() ([]any, error) {
	if r.index == 0 || r.index > len(r.rows) {
		return nil, errors.New("values before next")
	}
	return r.rows[r.index-1], nil
}

func (r *fakePGRows) RawValues() [][]byte {
	return nil
}

func (r *fakePGRows) Conn() *pgx.Conn {
	return nil
}

func assignScanValue(dest any, value any) {
	target := reflect.ValueOf(dest).Elem()
	if value == nil {
		target.Set(reflect.Zero(target.Type()))
		return
	}
	source := reflect.ValueOf(value)
	if source.Type().AssignableTo(target.Type()) {
		target.Set(source)
		return
	}
	if source.Type().ConvertibleTo(target.Type()) {
		target.Set(source.Convert(target.Type()))
		return
	}
	panic("cannot assign " + source.Type().String() + " to " + target.Type().String())
}

func fakeMatchRowValues(match Match) []any {
	return []any{
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
		match.CreatedAt,
		match.UpdatedAt,
	}
}

func fakeReplaySubmissionRow(matchID uuid.UUID, taskID uuid.UUID, userID uuid.UUID, now time.Time) []any {
	verdict := "accepted"
	judgeSubmissionID := "judge-1"
	judgeStatus := JudgeStatusFinished
	return []any{
		uuid.New(),
		matchID,
		taskID,
		userID,
		"go",
		"1.24",
		"package main",
		"finished",
		&verdict,
		&judgeSubmissionID,
		&judgeStatus,
		nil,
		1,
		1,
		nil,
		nil,
		nil,
		100,
		nil,
		nil,
		nil,
		nil,
		now,
		nil,
		nil,
		now,
	}
}

func TestContainsHelper(t *testing.T) {
	if !strings.Contains("submissions_judge_submission_id_uidx", "judge") {
		t.Fatal("stdlib sanity check failed")
	}
}
