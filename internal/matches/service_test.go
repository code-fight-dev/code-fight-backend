package matches

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestServiceCreateMatch(t *testing.T) {
	player1 := uuid.New()
	player2 := uuid.New()
	taskID := uuid.New()
	repo := &fakeMatchServiceRepo{}
	notifier := &fakeMatchNotifier{}
	metrics := &fakeMatchMetrics{}
	service := NewService(repo, fakeTaskSelector{taskID: taskID, difficulty: TaskDifficultyEasy}, nil)
	service.SetNotifier(notifier)
	service.SetMetrics(metrics)

	match, err := service.CreateMatch(context.Background(), CreateMatchInput{
		Player1ID: player1,
		Player2ID: player2,
		IsRated:   true,
	})
	if err != nil {
		t.Fatalf("CreateMatch returned error: %v", err)
	}
	if match.Player1ID != player1 || match.Player2ID != player2 || match.TaskID == nil || *match.TaskID != taskID {
		t.Fatalf("match = %#v", match)
	}
	if match.RatingMode != "global" || match.TaskMode != TaskModeNormal {
		t.Fatalf("defaults = %#v", match)
	}
	if len(repo.removedQueued) != 2 {
		t.Fatalf("removed queued = %#v", repo.removedQueued)
	}
	if notifier.eventType != "match.found" {
		t.Fatalf("event = %q", notifier.eventType)
	}
	if metrics.created != 1 {
		t.Fatalf("created metrics = %d", metrics.created)
	}
}

func TestServiceCreateMatchValidatesInput(t *testing.T) {
	service := NewService(&fakeMatchServiceRepo{}, fakeTaskSelector{}, nil)
	_, err := service.CreateMatch(context.Background(), CreateMatchInput{})
	if err == nil {
		t.Fatal("empty players returned nil error")
	}
	player := uuid.New()
	_, err = service.CreateMatch(context.Background(), CreateMatchInput{Player1ID: player, Player2ID: player})
	if !errors.Is(err, ErrSamePlayer) {
		t.Fatalf("same player error = %v", err)
	}
	_, err = service.CreateMatch(context.Background(), CreateMatchInput{Player1ID: uuid.New(), Player2ID: uuid.New(), TaskMode: "bad"})
	if !errors.Is(err, ErrInvalidTaskMode) {
		t.Fatalf("invalid task mode error = %v", err)
	}
}

func TestServiceStartMatch(t *testing.T) {
	taskID := uuid.New()
	difficulty := TaskDifficultyEasy
	player1 := uuid.New()
	player2 := uuid.New()
	matchID := uuid.New()
	repo := &fakeMatchServiceRepo{match: Match{
		ID:                     matchID,
		TaskID:                 &taskID,
		TaskDifficultySnapshot: &difficulty,
		Player1ID:              player1,
		Player2ID:              player2,
		Status:                 MatchStatusPending,
	}}
	service := NewService(repo, nil, nil)
	notifier := &fakeMatchNotifier{}
	service.SetNotifier(notifier)

	accepted, err := service.StartMatch(context.Background(), StartMatchInput{MatchID: matchID, ActorID: player1})
	if err != nil {
		t.Fatalf("StartMatch accepted returned error: %v", err)
	}
	if accepted.Status != MatchStatusPending || notifier.eventType != "match.accepted" {
		t.Fatalf("accepted = %#v event=%q", accepted, notifier.eventType)
	}

	repo.startAsRunning = true
	running, err := service.StartMatch(context.Background(), StartMatchInput{MatchID: matchID, ActorID: player2})
	if err != nil {
		t.Fatalf("StartMatch running returned error: %v", err)
	}
	if running.Status != MatchStatusRunning || notifier.eventType != "match.started" {
		t.Fatalf("running = %#v event=%q", running, notifier.eventType)
	}
}

func TestServiceJoinAndLeaveQueue(t *testing.T) {
	userID := uuid.New()
	opponentID := uuid.New()
	taskID := uuid.New()
	repo := &fakeMatchServiceRepo{opponentID: &opponentID}
	service := NewService(repo, fakeTaskSelector{taskID: taskID, difficulty: TaskDifficultyMedium}, nil)

	result, err := service.JoinQueue(context.Background(), JoinQueueInput{UserID: userID})
	if err != nil {
		t.Fatalf("JoinQueue matched returned error: %v", err)
	}
	if result.Status != "matched" || result.Match == nil {
		t.Fatalf("matched result = %#v", result)
	}

	queuedRepo := &fakeMatchServiceRepo{queued: &QueuedPlayer{UserID: userID, TaskMode: TaskModeNormal, RatingMode: "global", CreatedAt: time.Now()}}
	queuedService := NewService(queuedRepo, fakeTaskSelector{taskID: taskID}, nil)
	queuedResult, err := queuedService.JoinQueue(context.Background(), JoinQueueInput{UserID: userID})
	if err != nil {
		t.Fatalf("JoinQueue queued returned error: %v", err)
	}
	if queuedResult.Status != "queued" || !queuedRepo.enqueued {
		t.Fatalf("queued result = %#v enqueued=%v", queuedResult, queuedRepo.enqueued)
	}

	queuedRepo.current = Match{ID: uuid.New(), Player1ID: userID, Player2ID: uuid.New(), Status: MatchStatusPending}
	if err := queuedService.LeaveQueue(context.Background(), userID); err != nil {
		t.Fatalf("LeaveQueue returned error: %v", err)
	}
	if queuedRepo.cancelled == uuid.Nil {
		t.Fatal("pending match was not cancelled")
	}
}

func TestServiceFinishCancelAndSurrender(t *testing.T) {
	player1 := uuid.New()
	player2 := uuid.New()
	matchID := uuid.New()
	resultType := MatchResultTypePlayer1Win
	winningReason := WinningReasonSurrender
	repo := &fakeMatchServiceRepo{match: Match{
		ID:            matchID,
		Player1ID:     player1,
		Player2ID:     player2,
		Status:        MatchStatusRunning,
		ResultType:    &resultType,
		WinningReason: &winningReason,
	}}
	service := NewService(repo, nil, nil)
	metrics := &fakeMatchMetrics{}
	service.SetMetrics(metrics)

	if err := service.FinishMatch(context.Background(), FinishMatchInput{
		MatchID:       matchID,
		WinnerID:      &player1,
		ResultType:    MatchResultTypePlayer1Win,
		WinningReason: WinningReasonAcceptedFaster,
	}); err != nil {
		t.Fatalf("FinishMatch returned error: %v", err)
	}
	if repo.finishInput == nil || repo.finishInput.MatchID != matchID {
		t.Fatalf("finish input = %#v", repo.finishInput)
	}

	repo.match.Status = MatchStatusPending
	if err := service.CancelMatch(context.Background(), CancelMatchInput{MatchID: matchID, ActorID: player1}); err != nil {
		t.Fatalf("CancelMatch returned error: %v", err)
	}
	if repo.cancelled != matchID {
		t.Fatalf("cancelled = %s", repo.cancelled)
	}

	repo.match.Status = MatchStatusRunning
	if err := service.SurrenderMatch(context.Background(), SurrenderMatchInput{MatchID: matchID, ActorID: player1}); err != nil {
		t.Fatalf("SurrenderMatch returned error: %v", err)
	}
	if repo.finishInput == nil || repo.finishInput.WinnerID == nil || *repo.finishInput.WinnerID != player2 {
		t.Fatalf("surrender finish input = %#v", repo.finishInput)
	}
	if metrics.finished == 0 {
		t.Fatal("finished metric was not recorded")
	}
}

func TestServiceReadsAndPagination(t *testing.T) {
	player1 := uuid.New()
	player2 := uuid.New()
	matchID := uuid.New()
	repo := &fakeMatchServiceRepo{
		match:       Match{ID: matchID, Player1ID: player1, Player2ID: player2, Status: MatchStatusRunning},
		submissions: []MatchSubmission{{ID: uuid.New(), MatchID: matchID, UserID: player1}},
		publicStats: PublicStats{QueuedPlayers: 2},
	}
	service := NewService(repo, nil, nil)

	if _, err := service.GetMatch(context.Background(), matchID); err != nil {
		t.Fatalf("GetMatch returned error: %v", err)
	}
	if _, err := service.GetCurrentMatch(context.Background(), player1); err != nil {
		t.Fatalf("GetCurrentMatch returned error: %v", err)
	}
	if _, err := service.ListUserMatches(context.Background(), ListUserMatchesInput{UserID: player1, Limit: 200}); err != nil {
		t.Fatalf("ListUserMatches returned error: %v", err)
	}
	if repo.listLimit != 100 {
		t.Fatalf("list limit = %d, want 100", repo.listLimit)
	}
	if _, err := service.ListMatchSubmissions(context.Background(), matchID, player1); err != nil {
		t.Fatalf("ListMatchSubmissions returned error: %v", err)
	}
	if _, err := service.ListMatchSubmissions(context.Background(), matchID, uuid.New()); !errors.Is(err, ErrSubmissionUserNotInMatch) {
		t.Fatalf("non participant submissions error = %v", err)
	}

	stats, err := service.GetPublicStats(context.Background())
	if err != nil || stats.QueuedPlayers != 2 {
		t.Fatalf("stats = %#v, %v", stats, err)
	}
	_, _ = service.GetPublicStats(context.Background())
	if repo.publicStatsCalls != 1 {
		t.Fatalf("public stats calls = %d, want cache hit", repo.publicStatsCalls)
	}
}

func TestServiceApplySubmissionResult(t *testing.T) {
	player1 := uuid.New()
	player2 := uuid.New()
	matchID := uuid.New()
	submissionID := uuid.New()
	verdict := "accepted"
	finishedAt := time.Now()
	repo := &fakeMatchServiceRepo{match: Match{
		ID:        matchID,
		Player1ID: player1,
		Player2ID: player2,
		Status:    MatchStatusRunning,
	}}
	reader := fakeSubmissionReader{view: SubmissionMatchView{
		ID:         submissionID,
		MatchID:    &matchID,
		UserID:     player1,
		Status:     "finished",
		Verdict:    &verdict,
		Score:      100,
		FinishedAt: &finishedAt,
	}}
	service := NewService(repo, nil, reader)

	if err := service.ApplySubmissionResult(context.Background(), submissionID); err != nil {
		t.Fatalf("ApplySubmissionResult returned error: %v", err)
	}
	if repo.progressInput == nil || repo.progressInput.Score != 100 || !repo.progressInput.IsAccepted {
		t.Fatalf("progress input = %#v", repo.progressInput)
	}
	if repo.finishInput == nil || repo.finishInput.WinnerID == nil || *repo.finishInput.WinnerID != player1 {
		t.Fatalf("finish input = %#v", repo.finishInput)
	}
}

func TestMatchServiceHelpers(t *testing.T) {
	if delay, ok := nextMatchmakingExpansionDelay(30 * time.Second); !ok || delay <= 0 {
		t.Fatalf("delay = %s/%v", delay, ok)
	}
	if _, ok := nextMatchmakingExpansionDelay(10 * time.Minute); ok {
		t.Fatal("expected no expansion delay after all thresholds")
	}

	started := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	finished := started.Add(time.Second)
	match := Match{StartedAt: &started, FinishedAt: &finished, CreatedAt: started, UpdatedAt: finished}
	if got := calculateReplayDurationMs(match, nil, []MatchReplayEvent{{TMs: 2000}}, nil); got != 2000 {
		t.Fatalf("duration = %d", got)
	}
}

type fakeTaskSelector struct {
	taskID     uuid.UUID
	difficulty TaskDifficulty
	err        error
}

func (s fakeTaskSelector) SelectForMatch(context.Context, TaskMode) (SelectedTask, error) {
	if s.err != nil {
		return SelectedTask{}, s.err
	}
	if s.taskID == uuid.Nil {
		s.taskID = uuid.New()
	}
	if s.difficulty == "" {
		s.difficulty = TaskDifficultyEasy
	}
	return SelectedTask{TaskID: s.taskID, Difficulty: s.difficulty}, nil
}

type fakeSubmissionReader struct {
	view SubmissionMatchView
	err  error
}

func (r fakeSubmissionReader) GetSubmissionForMatchProcessing(context.Context, uuid.UUID) (SubmissionMatchView, error) {
	return r.view, r.err
}

type fakeMatchNotifier struct {
	eventType string
	userIDs   []uuid.UUID
}

func (n *fakeMatchNotifier) Publish(uuid.UUID, string, any) {}

func (n *fakeMatchNotifier) PublishMany(userIDs []uuid.UUID, eventType string, _ any) {
	n.userIDs = userIDs
	n.eventType = eventType
}

type fakeMatchMetrics struct {
	created   int
	finished  int
	cancelled int
}

func (m *fakeMatchMetrics) RecordMatchCreated(string, string, bool)  { m.created++ }
func (m *fakeMatchMetrics) RecordMatchFinished(string, string, bool) { m.finished++ }
func (m *fakeMatchMetrics) RecordMatchCancelled(string)              { m.cancelled++ }

type fakeMatchServiceRepo struct {
	match                Match
	current              Match
	opponentID           *uuid.UUID
	queued               *QueuedPlayer
	enqueued             bool
	removedQueued        []uuid.UUID
	cancelled            uuid.UUID
	startAsRunning       bool
	finishInput          *FinishMatchInput
	progressInput        *ApplySubmissionProgressInput
	submissions          []MatchSubmission
	publicStats          PublicStats
	publicStatsCalls     int
	listLimit            int
	finishExpiredLimit   int
	cancelExpiredLimit   int
	cancelExpiredTimeout time.Duration
	createErr            error
	matchErr             error
	startErr             error
	finishErr            error
	cancelErr            error
	listErr              error
	submissionsErr       error
	progressErr          error
	removeErr            error
	currentErr           error
	queuedErr            error
	findErr              error
	enqueueErr           error
	publicStatsErr       error
	finishExpiredErr     error
	cancelExpiredErr     error
	replayPlayersErr     error
	replaySubmissionsErr error
	replayEventsErr      error
	replaySnapshotsErr   error
	replayPlayers        []MatchReplayPlayer
	replaySubmissions    []MatchReplaySubmission
	replayEvents         []MatchReplayCapturedEvent
	replaySnapshots      []MatchReplayCapturedSnapshot
}

func (r *fakeMatchServiceRepo) CreateMatch(_ context.Context, match Match) (Match, error) {
	if r.createErr != nil {
		return Match{}, r.createErr
	}
	match.CreatedAt = time.Now()
	match.UpdatedAt = match.CreatedAt
	r.match = match
	return match, nil
}

func (r *fakeMatchServiceRepo) GetMatch(context.Context, uuid.UUID) (Match, error) {
	if r.matchErr != nil {
		return Match{}, r.matchErr
	}
	return r.match, nil
}

func (r *fakeMatchServiceRepo) StartMatch(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ time.Duration) (Match, error) {
	if r.startErr != nil {
		return Match{}, r.startErr
	}
	if r.startAsRunning {
		r.match.Status = MatchStatusRunning
	} else {
		r.match.Player1Ready = true
	}
	return r.match, nil
}

func (r *fakeMatchServiceRepo) FinishMatch(_ context.Context, input FinishMatchInput) error {
	if r.finishErr != nil {
		return r.finishErr
	}
	copied := input
	r.finishInput = &copied
	r.match.Status = MatchStatusFinished
	r.match.WinnerID = input.WinnerID
	r.match.ResultType = &input.ResultType
	r.match.WinningReason = &input.WinningReason
	return nil
}

func (r *fakeMatchServiceRepo) CancelMatch(_ context.Context, matchID uuid.UUID) error {
	if r.cancelErr != nil {
		return r.cancelErr
	}
	r.cancelled = matchID
	r.match.Status = MatchStatusCancelled
	return nil
}

func (r *fakeMatchServiceRepo) ListUserMatches(_ context.Context, _ uuid.UUID, limit, _ int) ([]Match, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	r.listLimit = limit
	return []Match{r.match}, nil
}

func (r *fakeMatchServiceRepo) GetCurrentMatch(context.Context, uuid.UUID) (Match, error) {
	if r.currentErr != nil {
		return Match{}, r.currentErr
	}
	return r.current, nil
}

func (r *fakeMatchServiceRepo) ListMatchSubmissions(context.Context, uuid.UUID) ([]MatchSubmission, error) {
	if r.submissionsErr != nil {
		return nil, r.submissionsErr
	}
	return r.submissions, nil
}

func (r *fakeMatchServiceRepo) ListMatchReplayPlayers(context.Context, uuid.UUID, uuid.UUID) ([]MatchReplayPlayer, error) {
	if r.replayPlayersErr != nil {
		return nil, r.replayPlayersErr
	}
	return r.replayPlayers, nil
}

func (r *fakeMatchServiceRepo) ListMatchReplaySubmissions(context.Context, uuid.UUID) ([]MatchReplaySubmission, error) {
	if r.replaySubmissionsErr != nil {
		return nil, r.replaySubmissionsErr
	}
	return r.replaySubmissions, nil
}

func (r *fakeMatchServiceRepo) FinishExpiredMatches(_ context.Context, _ time.Time, limit int) (int, error) {
	r.finishExpiredLimit = limit
	if r.finishExpiredErr != nil {
		return 0, r.finishExpiredErr
	}
	return 0, nil
}

func (r *fakeMatchServiceRepo) CancelExpiredPendingMatches(_ context.Context, _ time.Time, pendingTimeout time.Duration, limit int) (int, error) {
	r.cancelExpiredTimeout = pendingTimeout
	r.cancelExpiredLimit = limit
	if r.cancelExpiredErr != nil {
		return 0, r.cancelExpiredErr
	}
	if r.match.ID != uuid.Nil && r.match.Status == MatchStatusPending {
		r.match.Status = MatchStatusCancelled
		return 1, nil
	}
	return 0, nil
}

func (r *fakeMatchServiceRepo) FindQueuedOpponent(context.Context, JoinQueueInput) (*uuid.UUID, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	return r.opponentID, nil
}

func (r *fakeMatchServiceRepo) EnqueuePlayer(context.Context, JoinQueueInput) error {
	if r.enqueueErr != nil {
		return r.enqueueErr
	}
	r.enqueued = true
	return nil
}

func (r *fakeMatchServiceRepo) GetQueuedPlayer(context.Context, uuid.UUID) (*QueuedPlayer, error) {
	if r.queuedErr != nil {
		return nil, r.queuedErr
	}
	return r.queued, nil
}

func (r *fakeMatchServiceRepo) RemoveQueuedPlayers(_ context.Context, userIDs ...uuid.UUID) error {
	if r.removeErr != nil {
		return r.removeErr
	}
	r.removedQueued = append(r.removedQueued, userIDs...)
	return nil
}

func (r *fakeMatchServiceRepo) GetPublicStats(context.Context) (PublicStats, error) {
	if r.publicStatsErr != nil {
		return PublicStats{}, r.publicStatsErr
	}
	r.publicStatsCalls++
	return r.publicStats, nil
}

func (r *fakeMatchServiceRepo) ApplySubmissionProgress(_ context.Context, input ApplySubmissionProgressInput) error {
	if r.progressErr != nil {
		return r.progressErr
	}
	copied := input
	r.progressInput = &copied
	return nil
}

func (r *fakeMatchServiceRepo) ListMatchReplayTimelineEvents(context.Context, uuid.UUID) ([]MatchReplayCapturedEvent, error) {
	if r.replayEventsErr != nil {
		return nil, r.replayEventsErr
	}
	return r.replayEvents, nil
}

func (r *fakeMatchServiceRepo) ListMatchReplayTimelineSnapshots(context.Context, uuid.UUID) ([]MatchReplayCapturedSnapshot, error) {
	if r.replaySnapshotsErr != nil {
		return nil, r.replaySnapshotsErr
	}
	return r.replaySnapshots, nil
}
