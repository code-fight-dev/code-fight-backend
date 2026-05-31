package matches

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestServiceFinishMatchValidationBranches(t *testing.T) {
	matchID := uuid.New()
	winnerID := uuid.New()
	tests := []struct {
		name  string
		input FinishMatchInput
		want  error
	}{
		{name: "empty match", input: FinishMatchInput{}, want: nil},
		{name: "bad result", input: FinishMatchInput{MatchID: matchID, ResultType: "bad", WinningReason: WinningReasonDraw}, want: ErrInvalidResultType},
		{name: "bad winning reason", input: FinishMatchInput{MatchID: matchID, ResultType: MatchResultTypeDraw, WinningReason: "bad"}, want: ErrInvalidWinningReason},
		{name: "draw with winner", input: FinishMatchInput{MatchID: matchID, WinnerID: &winnerID, ResultType: MatchResultTypeDraw, WinningReason: WinningReasonDraw}, want: ErrInvalidWinner},
		{name: "win without winner", input: FinishMatchInput{MatchID: matchID, ResultType: MatchResultTypePlayer1Win, WinningReason: WinningReasonAcceptedFaster}, want: ErrInvalidWinner},
		{name: "bad judge status", input: FinishMatchInput{MatchID: matchID, ResultType: MatchResultTypeDraw, WinningReason: WinningReasonDraw, JudgeStatus: ptrJudgeStatus("bad")}, want: ErrInvalidJudgeStatus},
	}

	service := NewService(&fakeMatchServiceRepo{}, nil, nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.FinishMatch(context.Background(), tt.input)
			if tt.want == nil {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestServiceCancelAndSurrenderValidationBranches(t *testing.T) {
	player1 := uuid.New()
	player2 := uuid.New()
	matchID := uuid.New()
	repo := &fakeMatchServiceRepo{match: Match{
		ID:        matchID,
		Player1ID: player1,
		Player2ID: player2,
		Status:    MatchStatusRunning,
	}}
	service := NewService(repo, nil, nil)

	if err := service.CancelMatch(context.Background(), CancelMatchInput{}); err == nil {
		t.Fatal("empty cancel input returned nil error")
	}
	if err := service.CancelMatch(context.Background(), CancelMatchInput{MatchID: matchID, ActorID: uuid.New()}); !errors.Is(err, ErrSubmissionUserNotInMatch) {
		t.Fatalf("non participant cancel error = %v", err)
	}
	if err := service.CancelMatch(context.Background(), CancelMatchInput{MatchID: matchID, ActorID: player1}); !errors.Is(err, ErrMatchAlreadyStarted) {
		t.Fatalf("running cancel error = %v", err)
	}

	if err := service.SurrenderMatch(context.Background(), SurrenderMatchInput{}); err == nil {
		t.Fatal("empty surrender input returned nil error")
	}
	repo.match.Status = MatchStatusPending
	if err := service.SurrenderMatch(context.Background(), SurrenderMatchInput{MatchID: matchID, ActorID: player1}); !errors.Is(err, ErrInvalidMatchStatus) {
		t.Fatalf("pending surrender error = %v", err)
	}
}

func TestServiceExpiredMatchHelpersUseDefaults(t *testing.T) {
	repo := &fakeMatchServiceRepo{}
	service := NewService(repo, nil, nil)

	if _, err := service.FinishExpiredMatches(context.Background(), time.Now(), 0); err != nil {
		t.Fatalf("FinishExpiredMatches returned error: %v", err)
	}
	if repo.finishExpiredLimit != 50 {
		t.Fatalf("finish expired limit = %d, want 50", repo.finishExpiredLimit)
	}

	if _, err := service.CancelExpiredPendingMatches(context.Background(), time.Now(), 0, 0); err != nil {
		t.Fatalf("CancelExpiredPendingMatches returned error: %v", err)
	}
	if repo.cancelExpiredLimit != 50 || repo.cancelExpiredTimeout != defaultPendingMatchTimeout {
		t.Fatalf("cancel expired args = %d/%s", repo.cancelExpiredLimit, repo.cancelExpiredTimeout)
	}

	repoErr := errors.New("cleanup failed")
	service = NewService(&fakeMatchServiceRepo{finishExpiredErr: repoErr}, nil, nil)
	if _, err := service.FinishExpiredMatches(context.Background(), time.Now(), 1); !errors.Is(err, repoErr) {
		t.Fatalf("finish expired error = %v, want %v", err, repoErr)
	}

	service = NewService(&fakeMatchServiceRepo{cancelExpiredErr: repoErr}, nil, nil)
	if _, err := service.CancelExpiredPendingMatches(context.Background(), time.Now(), time.Second, 1); !errors.Is(err, repoErr) {
		t.Fatalf("cancel expired error = %v, want %v", err, repoErr)
	}
}

func TestServiceCancelExpiredPendingMatchOnRead(t *testing.T) {
	player1 := uuid.New()
	matchID := uuid.New()
	repo := &fakeMatchServiceRepo{match: Match{ID: matchID, Player1ID: player1, Player2ID: uuid.New(), Status: MatchStatusPending}}
	service := NewService(repo, nil, nil)
	metrics := &fakeMatchMetrics{}
	service.SetMetrics(metrics)

	match, err := service.GetMatch(context.Background(), matchID)
	if err != nil {
		t.Fatalf("GetMatch returned error: %v", err)
	}
	if match.Status != MatchStatusCancelled || metrics.cancelled != 1 {
		t.Fatalf("match status=%q cancelled metrics=%d", match.Status, metrics.cancelled)
	}
}

func TestServiceApplySubmissionResultEdges(t *testing.T) {
	matchID := uuid.New()
	player1 := uuid.New()
	player2 := uuid.New()
	submissionID := uuid.New()

	service := NewService(&fakeMatchServiceRepo{}, nil, nil)
	if err := service.ApplySubmissionResult(context.Background(), uuid.Nil); err == nil {
		t.Fatal("empty submission id returned nil error")
	}
	if err := service.ApplySubmissionResult(context.Background(), submissionID); err == nil {
		t.Fatal("missing submission reader returned nil error")
	}

	readerErr := errors.New("reader failed")
	service = NewService(&fakeMatchServiceRepo{}, nil, fakeSubmissionReader{err: readerErr})
	if err := service.ApplySubmissionResult(context.Background(), submissionID); !errors.Is(err, readerErr) {
		t.Fatalf("reader error = %v, want %v", err, readerErr)
	}

	service = NewService(&fakeMatchServiceRepo{}, nil, fakeSubmissionReader{
		view: SubmissionMatchView{ID: submissionID},
	})
	if err := service.ApplySubmissionResult(context.Background(), submissionID); err != nil {
		t.Fatalf("submission without match returned error: %v", err)
	}

	service = NewService(&fakeMatchServiceRepo{match: Match{
		ID:        matchID,
		Player1ID: player1,
		Player2ID: player2,
		Status:    MatchStatusFinished,
	}}, nil, fakeSubmissionReader{
		view: SubmissionMatchView{ID: submissionID, MatchID: &matchID, UserID: player1, Status: "finished"},
	})
	if err := service.ApplySubmissionResult(context.Background(), submissionID); err != nil {
		t.Fatalf("finished match returned error: %v", err)
	}

	service = NewService(&fakeMatchServiceRepo{match: Match{
		ID:        matchID,
		Player1ID: player1,
		Player2ID: player2,
		Status:    MatchStatusRunning,
	}}, nil, fakeSubmissionReader{
		view: SubmissionMatchView{ID: submissionID, MatchID: &matchID, UserID: uuid.New(), Status: "finished"},
	})
	if err := service.ApplySubmissionResult(context.Background(), submissionID); !errors.Is(err, ErrSubmissionUserNotInMatch) {
		t.Fatalf("non participant error = %v", err)
	}

	repoErr := errors.New("repo failed")
	service = NewService(&fakeMatchServiceRepo{matchErr: repoErr}, nil, fakeSubmissionReader{
		view: SubmissionMatchView{ID: submissionID, MatchID: &matchID, UserID: player1, Status: "finished"},
	})
	if err := service.ApplySubmissionResult(context.Background(), submissionID); !errors.Is(err, repoErr) {
		t.Fatalf("match read error = %v, want %v", err, repoErr)
	}

	service = NewService(&fakeMatchServiceRepo{match: Match{
		ID:        matchID,
		Player1ID: player1,
		Player2ID: player2,
		Status:    MatchStatusRunning,
	}, progressErr: repoErr}, nil, fakeSubmissionReader{
		view: SubmissionMatchView{ID: submissionID, MatchID: &matchID, UserID: player1, Status: "finished"},
	})
	if err := service.ApplySubmissionResult(context.Background(), submissionID); !errors.Is(err, repoErr) {
		t.Fatalf("progress error = %v, want %v", err, repoErr)
	}
}

func TestServiceApplySubmissionResultProgressOnly(t *testing.T) {
	matchID := uuid.New()
	player1 := uuid.New()
	player2 := uuid.New()
	submissionID := uuid.New()
	repo := &fakeMatchServiceRepo{match: Match{
		ID:        matchID,
		Player1ID: player1,
		Player2ID: player2,
		Status:    MatchStatusRunning,
	}}
	service := NewService(repo, nil, fakeSubmissionReader{
		view: SubmissionMatchView{ID: submissionID, MatchID: &matchID, UserID: player1, Status: "failed", Score: 3},
	})

	if err := service.ApplySubmissionResult(context.Background(), submissionID); err != nil {
		t.Fatalf("ApplySubmissionResult returned error: %v", err)
	}
	if repo.progressInput == nil || repo.progressInput.UserID != player1 || repo.progressInput.Score != 3 {
		t.Fatalf("progress input = %#v", repo.progressInput)
	}
	if repo.finishInput != nil {
		t.Fatalf("finish input = %#v, want nil", repo.finishInput)
	}
}

func TestServiceRunQueuedMatchCheck(t *testing.T) {
	userID := uuid.New()
	opponentID := uuid.New()
	taskID := uuid.New()
	repo := &fakeMatchServiceRepo{
		queued:     &QueuedPlayer{UserID: userID, RatingMode: "global", TaskMode: TaskModeNormal, CreatedAt: time.Now().Add(-2 * time.Minute)},
		opponentID: &opponentID,
	}
	service := NewService(repo, fakeTaskSelector{taskID: taskID}, nil)
	notifier := &fakeMatchNotifier{}
	service.SetNotifier(notifier)

	service.runQueuedMatchCheck(userID)

	if repo.match.ID == uuid.Nil || repo.match.Player1ID != opponentID || repo.match.Player2ID != userID {
		t.Fatalf("created match = %#v", repo.match)
	}
	if notifier.eventType != "matchmaking.matched" {
		t.Fatalf("event type = %q", notifier.eventType)
	}
}

func TestServiceRunQueuedMatchCheckReschedulesWhenStillWaiting(t *testing.T) {
	userID := uuid.New()
	repo := &fakeMatchServiceRepo{
		queued: &QueuedPlayer{UserID: userID, RatingMode: "global", TaskMode: TaskModeNormal, CreatedAt: time.Now()},
	}
	service := NewService(repo, fakeTaskSelector{}, nil)

	service.runQueuedMatchCheck(userID)

	service.queueTimersMu.Lock()
	timer := service.queueTimers[userID]
	if timer != nil {
		timer.Stop()
	}
	service.queueTimersMu.Unlock()
	if timer == nil {
		t.Fatal("expected queued match check to be rescheduled")
	}
}

func TestServiceRunQueuedMatchCheckErrorBranches(t *testing.T) {
	userID := uuid.New()
	repoErr := errors.New("repo failed")

	service := NewService(&fakeMatchServiceRepo{queuedErr: repoErr}, fakeTaskSelector{}, nil)
	service.runQueuedMatchCheck(userID)

	service = NewService(&fakeMatchServiceRepo{
		queued:     &QueuedPlayer{UserID: userID, RatingMode: "global", TaskMode: TaskModeNormal, CreatedAt: time.Now()},
		currentErr: repoErr,
	}, fakeTaskSelector{}, nil)
	service.runQueuedMatchCheck(userID)

	service = NewService(&fakeMatchServiceRepo{
		queued:  &QueuedPlayer{UserID: userID, RatingMode: "global", TaskMode: TaskModeNormal, CreatedAt: time.Now()},
		findErr: repoErr,
	}, fakeTaskSelector{}, nil)
	service.runQueuedMatchCheck(userID)

	service.queueTimersMu.Lock()
	timer := service.queueTimers[userID]
	if timer != nil {
		timer.Stop()
	}
	service.queueTimersMu.Unlock()
	if timer == nil {
		t.Fatal("expected failed find to reschedule")
	}

	opponentID := uuid.New()
	service = NewService(&fakeMatchServiceRepo{
		queued:     &QueuedPlayer{UserID: userID, RatingMode: "global", TaskMode: TaskModeNormal, CreatedAt: time.Now()},
		opponentID: &opponentID,
		createErr:  repoErr,
	}, fakeTaskSelector{}, nil)
	service.runQueuedMatchCheck(userID)
}

func TestServiceSchedulePendingMatchCancel(t *testing.T) {
	player1 := uuid.New()
	player2 := uuid.New()
	repo := &fakeMatchServiceRepo{}
	metrics := &fakeMatchMetrics{}
	service := NewService(repo, fakeTaskSelector{taskID: uuid.New()}, nil)
	service.pendingTimeout = time.Millisecond
	service.SetMetrics(metrics)

	match, err := service.CreateMatch(context.Background(), CreateMatchInput{
		Player1ID: player1,
		Player2ID: player2,
	})
	if err != nil {
		t.Fatalf("CreateMatch returned error: %v", err)
	}

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if repo.cancelled == match.ID {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if repo.cancelled != match.ID || repo.match.Status != MatchStatusCancelled {
		t.Fatalf("cancelled=%s match=%#v", repo.cancelled, repo.match)
	}
	if metrics.cancelled != 1 {
		t.Fatalf("cancelled metrics = %d", metrics.cancelled)
	}
}

func TestServiceCreateMatchErrorBranches(t *testing.T) {
	player1 := uuid.New()
	player2 := uuid.New()
	service := NewService(&fakeMatchServiceRepo{}, nil, nil)
	if _, err := service.CreateMatch(context.Background(), CreateMatchInput{Player1ID: player1, Player2ID: player2}); err == nil {
		t.Fatal("missing task selector returned nil error")
	}

	selectErr := errors.New("select failed")
	service = NewService(&fakeMatchServiceRepo{}, fakeTaskSelector{err: selectErr}, nil)
	if _, err := service.CreateMatch(context.Background(), CreateMatchInput{Player1ID: player1, Player2ID: player2}); !errors.Is(err, selectErr) {
		t.Fatalf("selector error = %v, want %v", err, selectErr)
	}

	createErr := errors.New("create failed")
	service = NewService(&fakeMatchServiceRepo{createErr: createErr}, fakeTaskSelector{}, nil)
	if _, err := service.CreateMatch(context.Background(), CreateMatchInput{Player1ID: player1, Player2ID: player2}); !errors.Is(err, createErr) {
		t.Fatalf("create error = %v, want %v", err, createErr)
	}

	removeErr := errors.New("remove failed")
	service = NewService(&fakeMatchServiceRepo{removeErr: removeErr}, fakeTaskSelector{}, nil)
	if _, err := service.CreateMatch(context.Background(), CreateMatchInput{Player1ID: player1, Player2ID: player2}); !errors.Is(err, removeErr) {
		t.Fatalf("remove queued error = %v, want %v", err, removeErr)
	}
}

func TestServiceReadAndQueueErrorBranches(t *testing.T) {
	userID := uuid.New()
	matchID := uuid.New()
	repoErr := errors.New("repo failed")

	service := NewService(&fakeMatchServiceRepo{currentErr: repoErr}, fakeTaskSelector{}, nil)
	if _, err := service.JoinQueue(context.Background(), JoinQueueInput{UserID: userID}); !errors.Is(err, repoErr) {
		t.Fatalf("current match error = %v, want %v", err, repoErr)
	}

	service = NewService(&fakeMatchServiceRepo{findErr: repoErr}, fakeTaskSelector{}, nil)
	if _, err := service.JoinQueue(context.Background(), JoinQueueInput{UserID: userID}); !errors.Is(err, repoErr) {
		t.Fatalf("find opponent error = %v, want %v", err, repoErr)
	}

	service = NewService(&fakeMatchServiceRepo{enqueueErr: repoErr}, fakeTaskSelector{}, nil)
	if _, err := service.JoinQueue(context.Background(), JoinQueueInput{UserID: userID}); !errors.Is(err, repoErr) {
		t.Fatalf("enqueue error = %v, want %v", err, repoErr)
	}

	service = NewService(&fakeMatchServiceRepo{removeErr: repoErr}, nil, nil)
	if err := service.LeaveQueue(context.Background(), userID); !errors.Is(err, repoErr) {
		t.Fatalf("leave remove error = %v, want %v", err, repoErr)
	}

	service = NewService(&fakeMatchServiceRepo{currentErr: repoErr}, nil, nil)
	if err := service.LeaveQueue(context.Background(), userID); !errors.Is(err, repoErr) {
		t.Fatalf("leave current error = %v, want %v", err, repoErr)
	}

	service = NewService(&fakeMatchServiceRepo{publicStatsErr: repoErr}, nil, nil)
	if _, err := service.GetPublicStats(context.Background()); !errors.Is(err, repoErr) {
		t.Fatalf("public stats error = %v, want %v", err, repoErr)
	}

	service = NewService(&fakeMatchServiceRepo{matchErr: repoErr}, nil, nil)
	if _, err := service.GetMatch(context.Background(), matchID); !errors.Is(err, repoErr) {
		t.Fatalf("get match error = %v, want %v", err, repoErr)
	}

	service = NewService(&fakeMatchServiceRepo{}, nil, nil)
	if _, err := service.ListUserMatches(context.Background(), ListUserMatchesInput{}); err == nil {
		t.Fatal("empty user list returned nil error")
	}
	if _, err := service.ListUserMatches(context.Background(), ListUserMatchesInput{UserID: userID, Limit: -1}); !errors.Is(err, ErrInvalidPagination) {
		t.Fatalf("invalid pagination error = %v", err)
	}
}

func TestServiceStartMatchErrorBranches(t *testing.T) {
	player1 := uuid.New()
	player2 := uuid.New()
	matchID := uuid.New()
	taskID := uuid.New()
	difficulty := TaskDifficultyEasy
	repoErr := errors.New("repo failed")

	service := NewService(&fakeMatchServiceRepo{matchErr: repoErr}, nil, nil)
	if _, err := service.StartMatch(context.Background(), StartMatchInput{MatchID: matchID, ActorID: player1}); !errors.Is(err, repoErr) {
		t.Fatalf("get match error = %v, want %v", err, repoErr)
	}

	service = NewService(&fakeMatchServiceRepo{match: Match{ID: matchID, Player1ID: player1, Player2ID: player2}}, nil, nil)
	if _, err := service.StartMatch(context.Background(), StartMatchInput{MatchID: matchID, ActorID: uuid.New()}); !errors.Is(err, ErrSubmissionUserNotInMatch) {
		t.Fatalf("non participant error = %v", err)
	}
	if _, err := service.StartMatch(context.Background(), StartMatchInput{MatchID: matchID, ActorID: player1}); !errors.Is(err, ErrTaskNotAssigned) {
		t.Fatalf("missing task error = %v", err)
	}

	service = NewService(&fakeMatchServiceRepo{
		match:    Match{ID: matchID, Player1ID: player1, Player2ID: player2, TaskID: &taskID, TaskDifficultySnapshot: &difficulty},
		startErr: repoErr,
	}, nil, nil)
	if _, err := service.StartMatch(context.Background(), StartMatchInput{MatchID: matchID, ActorID: player1}); !errors.Is(err, repoErr) {
		t.Fatalf("start error = %v, want %v", err, repoErr)
	}
}

func TestServiceReplayBranches(t *testing.T) {
	player1 := uuid.New()
	player2 := uuid.New()
	matchID := uuid.New()
	now := time.Now().UTC()
	repo := &fakeMatchServiceRepo{
		match: Match{
			ID:         matchID,
			Player1ID:  player1,
			Player2ID:  player2,
			Status:     MatchStatusFinished,
			CreatedAt:  now.Add(-time.Minute),
			UpdatedAt:  now,
			FinishedAt: &now,
		},
		replayPlayers: []MatchReplayPlayer{{ID: player1, Username: "p1"}, {ID: player2, Username: "p2"}},
		replaySubmissions: []MatchReplaySubmission{{
			ID:        uuid.New(),
			MatchID:   matchID,
			UserID:    player1,
			UpdatedAt: now.Add(time.Second),
		}},
		replayEvents: []MatchReplayCapturedEvent{{
			UserID:    player1,
			Seq:       1,
			TMs:       1500,
			EventType: "insert",
			Payload:   []byte(`{"text":"x"}`),
		}},
		replaySnapshots: []MatchReplayCapturedSnapshot{{
			UserID:     player2,
			Seq:        1,
			TMs:        1200,
			Language:   "go",
			SourceCode: "package main",
		}},
	}
	service := NewService(repo, nil, nil)

	replay, err := service.GetMatchReplay(context.Background(), matchID, player1)
	if err != nil {
		t.Fatalf("GetMatchReplay returned error: %v", err)
	}
	if !replay.Permissions.CanViewReplay || len(replay.Players) != 2 || len(replay.Timeline.Events) != 1 || len(replay.Timeline.Snapshots) != 1 {
		t.Fatalf("replay = %#v", replay)
	}

	if _, err := service.GetMatchReplay(context.Background(), matchID, uuid.New()); !errors.Is(err, ErrMatchReplayForbidden) {
		t.Fatalf("forbidden replay error = %v", err)
	}

	repo.match.Status = MatchStatusRunning
	if _, err := service.GetMatchReplay(context.Background(), matchID, player1); !errors.Is(err, ErrMatchReplayUnavailable) {
		t.Fatalf("unavailable replay error = %v", err)
	}

	if _, err := service.GetMatchReplay(context.Background(), uuid.Nil, player1); err == nil {
		t.Fatal("empty replay match id returned nil error")
	}
	if _, err := service.GetMatchReplay(context.Background(), matchID, uuid.Nil); err == nil {
		t.Fatal("empty replay viewer id returned nil error")
	}

	repo.match.Status = MatchStatusFinished
	repo.replaySubmissionsErr = errors.New("submissions failed")
	if _, err := service.GetMatchReplay(context.Background(), matchID, player1); !errors.Is(err, repo.replaySubmissionsErr) {
		t.Fatalf("replay submissions error = %v", err)
	}
	repo.replaySubmissionsErr = nil
	repo.replayEventsErr = errors.New("events failed")
	if _, err := service.GetMatchReplay(context.Background(), matchID, player1); !errors.Is(err, repo.replayEventsErr) {
		t.Fatalf("replay events error = %v", err)
	}
	repo.replayEventsErr = nil
	repo.replaySnapshotsErr = errors.New("snapshots failed")
	if _, err := service.GetMatchReplay(context.Background(), matchID, player1); !errors.Is(err, repo.replaySnapshotsErr) {
		t.Fatalf("replay snapshots error = %v", err)
	}
}
