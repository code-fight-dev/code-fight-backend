package matches

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type RepositoryInterface interface {
	CreateMatch(ctx context.Context, match Match) (Match, error)
	GetMatch(ctx context.Context, matchID uuid.UUID) (Match, error)
	StartMatch(ctx context.Context, matchID uuid.UUID, duration time.Duration) (Match, error)
	FinishMatch(ctx context.Context, input FinishMatchInput) error
	CancelMatch(ctx context.Context, matchID uuid.UUID) error
	ListUserMatches(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Match, error)
	GetCurrentMatch(ctx context.Context, userID uuid.UUID) (Match, error)
	ListMatchSubmissions(ctx context.Context, matchID uuid.UUID) ([]MatchSubmission, error)
	FinishExpiredMatches(ctx context.Context, now time.Time, limit int) (int, error)
	FindQueuedOpponent(ctx context.Context, input JoinQueueInput) (*uuid.UUID, error)
	EnqueuePlayer(ctx context.Context, input JoinQueueInput) error
	RemoveQueuedPlayers(ctx context.Context, userIDs ...uuid.UUID) error

	ApplySubmissionProgress(ctx context.Context, input ApplySubmissionProgressInput) error
}

type Service struct {
	repo             RepositoryInterface
	taskSelector     TaskSelector
	submissionReader SubmissionReader
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
	}
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

	return created, nil
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

	return s.repo.StartMatch(ctx, input.MatchID, input.Duration)
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

	return QueueResult{Status: "matched", Match: &match}, nil
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

	return s.repo.FinishMatch(ctx, input)
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

	return s.repo.CancelMatch(ctx, input.MatchID)
}

func (s *Service) GetMatch(ctx context.Context, matchID uuid.UUID) (Match, error) {
	if matchID == uuid.Nil {
		return Match{}, fmt.Errorf("get match: empty match id")
	}

	return s.repo.GetMatch(ctx, matchID)
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

	return nil
}

func (s *Service) GetCurrentMatch(ctx context.Context, userID uuid.UUID) (Match, error) {
	if userID == uuid.Nil {
		return Match{}, fmt.Errorf("get current match: empty user id")
	}

	return s.repo.GetCurrentMatch(ctx, userID)
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
