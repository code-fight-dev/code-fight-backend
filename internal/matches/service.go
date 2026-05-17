package matches

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	defaultPendingMatchTimeout = 30 * time.Second
	matchmakingCheckSlack      = time.Second
)

var matchmakingExpansionDelays = []time.Duration{
	time.Minute,
	3 * time.Minute,
	5 * time.Minute,
}

type RepositoryInterface interface {
	CreateMatch(ctx context.Context, match Match) (Match, error)
	GetMatch(ctx context.Context, matchID uuid.UUID) (Match, error)
	StartMatch(ctx context.Context, matchID uuid.UUID, actorID uuid.UUID, duration time.Duration) (Match, error)
	FinishMatch(ctx context.Context, input FinishMatchInput) error
	CancelMatch(ctx context.Context, matchID uuid.UUID) error
	ListUserMatches(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Match, error)
	GetCurrentMatch(ctx context.Context, userID uuid.UUID) (Match, error)
	ListMatchSubmissions(ctx context.Context, matchID uuid.UUID) ([]MatchSubmission, error)
	FinishExpiredMatches(ctx context.Context, now time.Time, limit int) (int, error)
	CancelExpiredPendingMatches(ctx context.Context, now time.Time, pendingTimeout time.Duration, limit int) (int, error)
	FindQueuedOpponent(ctx context.Context, input JoinQueueInput) (*uuid.UUID, error)
	EnqueuePlayer(ctx context.Context, input JoinQueueInput) error
	GetQueuedPlayer(ctx context.Context, userID uuid.UUID) (*QueuedPlayer, error)
	RemoveQueuedPlayers(ctx context.Context, userIDs ...uuid.UUID) error

	ApplySubmissionProgress(ctx context.Context, input ApplySubmissionProgressInput) error
}

type Notifier interface {
	Publish(userID uuid.UUID, eventType string, payload any)
	PublishMany(userIDs []uuid.UUID, eventType string, payload any)
}

type Service struct {
	repo             RepositoryInterface
	taskSelector     TaskSelector
	submissionReader SubmissionReader
	pendingTimeout   time.Duration
	notifier         Notifier
	queueTimersMu    sync.Mutex
	queueTimers      map[uuid.UUID]*time.Timer
}

func NewService(
	repo RepositoryInterface,
	taskSelector TaskSelector,
	submissionReader SubmissionReader,
) *Service {
	return &Service{
		repo:             repo,
		taskSelector:     taskSelector,
		submissionReader: submissionReader,
		pendingTimeout:   defaultPendingMatchTimeout,
		queueTimers:      make(map[uuid.UUID]*time.Timer),
	}
}

func (s *Service) SetNotifier(notifier Notifier) {
	s.notifier = notifier
}

func (s *Service) CreateMatch(ctx context.Context, input CreateMatchInput) (Match, error) {
	if input.Player1ID == uuid.Nil || input.Player2ID == uuid.Nil {
		return Match{}, fmt.Errorf("create match: empty player id")
	}
	if input.Player1ID == input.Player2ID {
		return Match{}, ErrSamePlayer
	}

	if input.RatingMode == "" {
		input.RatingMode = "global"
	}

	if s.taskSelector == nil {
		return Match{}, fmt.Errorf("create match: task selector is not configured")
	}

	switch input.TaskMode {
	case "":
		input.TaskMode = TaskModeNormal
	case TaskModeNormal, TaskModeHard:
	default:
		return Match{}, ErrInvalidTaskMode
	}

	selected, err := s.taskSelector.SelectForMatch(ctx, input.TaskMode)
	if err != nil {
		return Match{}, err
	}

	match := Match{
		ID:                     uuid.New(),
		RatingMode:             input.RatingMode,
		TaskMode:               input.TaskMode,
		TaskID:                 &selected.TaskID,
		TaskDifficultySnapshot: &selected.Difficulty,
		Player1ID:              input.Player1ID,
		Player2ID:              input.Player2ID,
		Status:                 MatchStatusPending,
		JudgeStatus:            JudgeStatusNotStarted,
		IsRated:                input.IsRated,
		RatingApplied:          false,
	}

	created, err := s.repo.CreateMatch(ctx, match)
	if err != nil {
		return Match{}, err
	}

	if err := s.repo.RemoveQueuedPlayers(ctx, input.Player1ID, input.Player2ID); err != nil {
		return Match{}, err
	}
	s.cancelQueuedMatchCheck(input.Player1ID)
	s.cancelQueuedMatchCheck(input.Player2ID)

	s.schedulePendingMatchCancel(created.ID)
	s.publishMatch("match.found", created)

	return created, nil
}

func (s *Service) schedulePendingMatchCancel(matchID uuid.UUID) {
	if matchID == uuid.Nil {
		return
	}

	time.AfterFunc(s.pendingTimeout, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		match, err := s.repo.GetMatch(ctx, matchID)
		if err != nil || match.Status != MatchStatusPending {
			return
		}

		if err := s.repo.CancelMatch(ctx, matchID); err != nil {
			return
		}
		cancelled, err := s.repo.GetMatch(ctx, matchID)
		if err != nil {
			return
		}
		s.publishMatch("match.cancelled", cancelled)
	})
}

func (s *Service) StartMatch(ctx context.Context, input StartMatchInput) (Match, error) {
	if input.MatchID == uuid.Nil {
		return Match{}, fmt.Errorf("start match: empty match id")
	}
	if input.ActorID == uuid.Nil {
		return Match{}, fmt.Errorf("start match: empty actor id")
	}

	match, err := s.repo.GetMatch(ctx, input.MatchID)
	if err != nil {
		return Match{}, err
	}
	if input.ActorID != match.Player1ID && input.ActorID != match.Player2ID {
		return Match{}, ErrSubmissionUserNotInMatch
	}

	if match.TaskID == nil || match.TaskDifficultySnapshot == nil {
		return Match{}, ErrTaskNotAssigned
	}

	updated, err := s.repo.StartMatch(ctx, input.MatchID, input.ActorID, input.Duration)
	if err != nil {
		return Match{}, err
	}
	if updated.Status == MatchStatusRunning {
		s.publishMatch("match.started", updated)
	} else {
		s.publishMatch("match.accepted", updated)
	}

	return updated, nil
}

func (s *Service) JoinQueue(ctx context.Context, input JoinQueueInput) (QueueResult, error) {
	if input.UserID == uuid.Nil {
		return QueueResult{}, fmt.Errorf("join queue: empty user id")
	}
	if input.RatingMode == "" {
		input.RatingMode = "global"
	}
	switch input.TaskMode {
	case "":
		input.TaskMode = TaskModeNormal
	case TaskModeNormal, TaskModeHard:
	default:
		return QueueResult{}, ErrInvalidTaskMode
	}

	current, err := s.repo.GetCurrentMatch(ctx, input.UserID)
	if err != nil {
		return QueueResult{}, err
	}
	if current.ID != uuid.Nil {
		return QueueResult{Status: "matched", Match: &current}, nil
	}

	opponentID, err := s.repo.FindQueuedOpponent(ctx, input)
	if err != nil {
		return QueueResult{}, err
	}
	if opponentID == nil {
		if err := s.repo.EnqueuePlayer(ctx, input); err != nil {
			return QueueResult{}, err
		}
		if queued, err := s.repo.GetQueuedPlayer(ctx, input.UserID); err == nil && queued != nil {
			s.scheduleQueuedMatchCheck(*queued)
		}
		if s.notifier != nil {
			s.notifier.Publish(input.UserID, "matchmaking.queued", QueueResult{Status: "queued"})
		}
		return QueueResult{Status: "queued"}, nil
	}

	match, err := s.CreateMatch(ctx, CreateMatchInput{
		Player1ID:  *opponentID,
		Player2ID:  input.UserID,
		RatingMode: input.RatingMode,
		TaskMode:   input.TaskMode,
		IsRated:    input.IsRated,
	})
	if err != nil {
		return QueueResult{}, err
	}

	result := QueueResult{Status: "matched", Match: &match}
	s.publishMatch("matchmaking.matched", match)
	return result, nil
}

func (s *Service) LeaveQueue(ctx context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return fmt.Errorf("leave queue: empty user id")
	}

	if err := s.repo.RemoveQueuedPlayers(ctx, userID); err != nil {
		return err
	}
	s.cancelQueuedMatchCheck(userID)

	current, err := s.repo.GetCurrentMatch(ctx, userID)
	if err != nil {
		return err
	}
	if current.ID == uuid.Nil || current.Status != MatchStatusPending {
		return nil
	}

	if err := s.repo.CancelMatch(ctx, current.ID); err != nil {
		return err
	}
	cancelled, err := s.repo.GetMatch(ctx, current.ID)
	if err != nil {
		return err
	}
	s.publishMatch("match.cancelled", cancelled)
	return nil
}

func (s *Service) scheduleQueuedMatchCheck(player QueuedPlayer) {
	if player.UserID == uuid.Nil {
		return
	}

	delay, ok := nextMatchmakingExpansionDelay(time.Since(player.CreatedAt))
	if !ok {
		return
	}

	s.queueTimersMu.Lock()
	if existing := s.queueTimers[player.UserID]; existing != nil {
		existing.Stop()
	}
	s.queueTimers[player.UserID] = time.AfterFunc(delay, func() {
		s.runQueuedMatchCheck(player.UserID)
	})
	s.queueTimersMu.Unlock()
}

func (s *Service) cancelQueuedMatchCheck(userID uuid.UUID) {
	if userID == uuid.Nil {
		return
	}

	s.queueTimersMu.Lock()
	if existing := s.queueTimers[userID]; existing != nil {
		existing.Stop()
		delete(s.queueTimers, userID)
	}
	s.queueTimersMu.Unlock()
}

func (s *Service) runQueuedMatchCheck(userID uuid.UUID) {
	s.queueTimersMu.Lock()
	delete(s.queueTimers, userID)
	s.queueTimersMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	queued, err := s.repo.GetQueuedPlayer(ctx, userID)
	if err != nil || queued == nil {
		return
	}

	current, err := s.repo.GetCurrentMatch(ctx, userID)
	if err != nil || current.ID != uuid.Nil {
		return
	}

	input := JoinQueueInput{
		UserID:     queued.UserID,
		RatingMode: queued.RatingMode,
		TaskMode:   queued.TaskMode,
		IsRated:    queued.IsRated,
	}

	opponentID, err := s.repo.FindQueuedOpponent(ctx, input)
	if err != nil {
		s.scheduleQueuedMatchCheck(*queued)
		return
	}
	if opponentID == nil {
		s.scheduleQueuedMatchCheck(*queued)
		return
	}

	match, err := s.CreateMatch(ctx, CreateMatchInput{
		Player1ID:  *opponentID,
		Player2ID:  queued.UserID,
		RatingMode: queued.RatingMode,
		TaskMode:   queued.TaskMode,
		IsRated:    queued.IsRated,
	})
	if err != nil {
		s.scheduleQueuedMatchCheck(*queued)
		return
	}

	s.publishMatch("matchmaking.matched", match)
}

func nextMatchmakingExpansionDelay(elapsed time.Duration) (time.Duration, bool) {
	for _, threshold := range matchmakingExpansionDelays {
		if elapsed < threshold {
			return threshold - elapsed + matchmakingCheckSlack, true
		}
	}

	return 0, false
}

func (s *Service) FinishMatch(ctx context.Context, input FinishMatchInput) error {
	if input.MatchID == uuid.Nil {
		return fmt.Errorf("finish match: empty match id")
	}

	switch input.ResultType {
	case MatchResultTypePlayer1Win, MatchResultTypePlayer2Win, MatchResultTypeDraw:
	default:
		return ErrInvalidResultType
	}

	switch input.WinningReason {
	case WinningReasonAcceptedFaster,
		WinningReasonAcceptedMoreTests,
		WinningReasonOpponentFailed,
		WinningReasonSurrender,
		WinningReasonDraw:
	default:
		return ErrInvalidWinningReason
	}

	switch input.ResultType {
	case MatchResultTypeDraw:
		if input.WinnerID != nil {
			return ErrInvalidWinner
		}
	case MatchResultTypePlayer1Win, MatchResultTypePlayer2Win:
		if input.WinnerID == nil {
			return ErrInvalidWinner
		}
	}

	if input.JudgeStatus != nil {
		switch *input.JudgeStatus {
		case JudgeStatusNotStarted,
			JudgeStatusReady,
			JudgeStatusRunning,
			JudgeStatusFinished,
			JudgeStatusFailed:
		default:
			return ErrInvalidJudgeStatus
		}
	}

	if err := s.repo.FinishMatch(ctx, input); err != nil {
		return err
	}
	match, err := s.repo.GetMatch(ctx, input.MatchID)
	if err == nil {
		s.publishMatch("match.finished", match)
	}
	return nil
}

func (s *Service) CancelMatch(ctx context.Context, input CancelMatchInput) error {
	if input.MatchID == uuid.Nil {
		return fmt.Errorf("cancel match: empty match id")
	}
	if input.ActorID == uuid.Nil {
		return fmt.Errorf("cancel match: empty actor id")
	}
	match, err := s.repo.GetMatch(ctx, input.MatchID)
	if err != nil {
		return err
	}
	if input.ActorID != match.Player1ID && input.ActorID != match.Player2ID {
		return ErrSubmissionUserNotInMatch
	}
	if match.Status == MatchStatusRunning {
		return ErrMatchAlreadyStarted
	}

	if err := s.repo.CancelMatch(ctx, input.MatchID); err != nil {
		return err
	}
	cancelled, err := s.repo.GetMatch(ctx, input.MatchID)
	if err == nil {
		s.publishMatch("match.cancelled", cancelled)
	}
	return nil
}

func (s *Service) SurrenderMatch(ctx context.Context, input SurrenderMatchInput) error {
	if input.MatchID == uuid.Nil {
		return fmt.Errorf("surrender match: empty match id")
	}
	if input.ActorID == uuid.Nil {
		return fmt.Errorf("surrender match: empty actor id")
	}

	match, err := s.repo.GetMatch(ctx, input.MatchID)
	if err != nil {
		return err
	}
	if input.ActorID != match.Player1ID && input.ActorID != match.Player2ID {
		return ErrSubmissionUserNotInMatch
	}
	if match.Status != MatchStatusRunning {
		return ErrInvalidMatchStatus
	}

	winnerID := match.Player1ID
	resultType := MatchResultTypePlayer1Win
	if input.ActorID == match.Player1ID {
		winnerID = match.Player2ID
		resultType = MatchResultTypePlayer2Win
	}

	judgeStatus := JudgeStatusFinished
	if err := s.repo.FinishMatch(ctx, FinishMatchInput{
		MatchID:       match.ID,
		WinnerID:      &winnerID,
		ResultType:    resultType,
		WinningReason: WinningReasonSurrender,
		JudgeStatus:   &judgeStatus,
	}); err != nil {
		return err
	}
	finished, err := s.repo.GetMatch(ctx, match.ID)
	if err == nil {
		s.publishMatch("match.finished", finished)
	}
	return nil
}

func (s *Service) GetMatch(ctx context.Context, matchID uuid.UUID) (Match, error) {
	if matchID == uuid.Nil {
		return Match{}, fmt.Errorf("get match: empty match id")
	}

	match, err := s.repo.GetMatch(ctx, matchID)
	if err != nil {
		return Match{}, err
	}

	return s.cancelExpiredPendingMatchOnRead(ctx, match)
}

func (s *Service) ListUserMatches(ctx context.Context, input ListUserMatchesInput) ([]Match, error) {
	if input.UserID == uuid.Nil {
		return nil, fmt.Errorf("list user matches: empty user id")
	}
	if input.Limit < 0 || input.Offset < 0 {
		return nil, ErrInvalidPagination
	}
	if input.Limit == 0 {
		input.Limit = 20
	}
	if input.Limit > 100 {
		input.Limit = 100
	}

	return s.repo.ListUserMatches(ctx, input.UserID, input.Limit, input.Offset)
}

func (s *Service) ApplySubmissionResult(ctx context.Context, submissionID uuid.UUID) error {
	if submissionID == uuid.Nil {
		return fmt.Errorf("apply submission result: empty submission id")
	}
	if s.submissionReader == nil {
		return fmt.Errorf("apply submission result: submission reader is not configured")
	}

	submission, err := s.submissionReader.GetSubmissionForMatchProcessing(ctx, submissionID)
	if err != nil {
		return err
	}

	if submission.MatchID == nil || *submission.MatchID == uuid.Nil {
		return nil
	}

	match, err := s.repo.GetMatch(ctx, *submission.MatchID)
	if err != nil {
		return err
	}

	if match.Status == MatchStatusFinished || match.Status == MatchStatusCancelled {
		return nil
	}

	isPlayer1 := submission.UserID == match.Player1ID
	isPlayer2 := submission.UserID == match.Player2ID

	if !isPlayer1 && !isPlayer2 {
		return ErrSubmissionUserNotInMatch
	}

	judgeStatus := JudgeStatusRunning
	if submission.Status == "finished" {
		judgeStatus = JudgeStatusFinished
	}
	if submission.Status == "failed" {
		judgeStatus = JudgeStatusFailed
	}

	progressJudgeStatus := string(judgeStatus)
	isAccepted := submission.Verdict != nil && *submission.Verdict == "accepted"

	err = s.repo.ApplySubmissionProgress(ctx, ApplySubmissionProgressInput{
		MatchID:     match.ID,
		UserID:      submission.UserID,
		Score:       submission.Score,
		IsAccepted:  isAccepted,
		SolvedAt:    submission.FinishedAt,
		JudgeStatus: &progressJudgeStatus,
	})
	if err != nil {
		return err
	}
	if updated, err := s.repo.GetMatch(ctx, match.ID); err == nil {
		s.publishMatch("match.progress", updated)
	}

	if !isAccepted {
		return nil
	}

	updatedMatch, err := s.repo.GetMatch(ctx, match.ID)
	if err != nil {
		return err
	}

	if updatedMatch.Status == MatchStatusFinished || updatedMatch.Status == MatchStatusCancelled {
		return nil
	}

	winningReason := WinningReasonAcceptedFaster

	var resultType MatchResultType
	winnerID := submission.UserID
	finalJudgeStatus := JudgeStatusFinished

	switch submission.UserID {
	case updatedMatch.Player1ID:
		resultType = MatchResultTypePlayer1Win
	case updatedMatch.Player2ID:
		resultType = MatchResultTypePlayer2Win
	default:
		return ErrSubmissionUserNotInMatch
	}

	err = s.repo.FinishMatch(ctx, FinishMatchInput{
		MatchID:       updatedMatch.ID,
		WinnerID:      &winnerID,
		ResultType:    resultType,
		WinningReason: winningReason,
		JudgeStatus:   &finalJudgeStatus,
	})
	if err != nil {
		if errors.Is(err, ErrMatchAlreadyFinished) ||
			errors.Is(err, ErrRatingAlreadyApplied) ||
			errors.Is(err, ErrMatchCancelled) {
			return nil
		}
		return err
	}
	if finished, err := s.repo.GetMatch(ctx, updatedMatch.ID); err == nil {
		s.publishMatch("match.finished", finished)
	}

	return nil
}

func (s *Service) GetCurrentMatch(ctx context.Context, userID uuid.UUID) (Match, error) {
	if userID == uuid.Nil {
		return Match{}, fmt.Errorf("get current match: empty user id")
	}

	match, err := s.repo.GetCurrentMatch(ctx, userID)
	if err != nil {
		return Match{}, err
	}

	return s.cancelExpiredPendingMatchOnRead(ctx, match)
}

func (s *Service) cancelExpiredPendingMatchOnRead(ctx context.Context, match Match) (Match, error) {
	if match.ID == uuid.Nil || match.Status != MatchStatusPending {
		return match, nil
	}

	if _, err := s.repo.CancelExpiredPendingMatches(ctx, time.Now().UTC(), s.pendingTimeout, 100); err != nil {
		return Match{}, err
	}

	updated, err := s.repo.GetMatch(ctx, match.ID)
	if err == nil && updated.Status == MatchStatusCancelled {
		s.publishMatch("match.cancelled", updated)
	}
	return updated, err
}

func (s *Service) ListMatchSubmissions(ctx context.Context, matchID uuid.UUID, viewerID uuid.UUID) ([]MatchSubmission, error) {
	if matchID == uuid.Nil {
		return nil, fmt.Errorf("list match submissions: empty match id")
	}
	if viewerID == uuid.Nil {
		return nil, fmt.Errorf("list match submissions: empty viewer id")
	}

	match, err := s.repo.GetMatch(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if viewerID != match.Player1ID && viewerID != match.Player2ID {
		return nil, ErrSubmissionUserNotInMatch
	}

	return s.repo.ListMatchSubmissions(ctx, matchID)
}

func (s *Service) FinishExpiredMatches(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 50
	}

	return s.repo.FinishExpiredMatches(ctx, now, limit)
}

func (s *Service) CancelExpiredPendingMatches(ctx context.Context, now time.Time, pendingTimeout time.Duration, limit int) (int, error) {
	if pendingTimeout <= 0 {
		pendingTimeout = defaultPendingMatchTimeout
	}
	if limit <= 0 {
		limit = 50
	}

	return s.repo.CancelExpiredPendingMatches(ctx, now, pendingTimeout, limit)
}

func (s *Service) publishMatch(eventType string, match Match) {
	if s.notifier == nil || match.ID == uuid.Nil {
		return
	}
	s.notifier.PublishMany([]uuid.UUID{match.Player1ID, match.Player2ID}, eventType, match)
}
