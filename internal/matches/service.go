package matches

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type RepositoryInterface interface {
	CreateMatch(ctx context.Context, match Match) (Match, error)
	GetMatch(ctx context.Context, matchID uuid.UUID) (Match, error)
	StartMatch(ctx context.Context, matchID uuid.UUID) (Match, error)
	FinishMatch(ctx context.Context, input FinishMatchInput) error
	CancelMatch(ctx context.Context, matchID uuid.UUID) error
	ListUserMatches(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Match, error)
}

type Service struct {
	repo         RepositoryInterface
	taskSelector TaskSelector
}

func NewService(repo RepositoryInterface, taskSelector TaskSelector) *Service {
	return &Service{
		repo:         repo,
		taskSelector: taskSelector,
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

	return s.repo.CreateMatch(ctx, match)
}

func (s *Service) StartMatch(ctx context.Context, input StartMatchInput) (Match, error) {
	if input.MatchID == uuid.Nil {
		return Match{}, fmt.Errorf("start match: empty match id")
	}

	match, err := s.repo.GetMatch(ctx, input.MatchID)
	if err != nil {
		return Match{}, err
	}

	if match.TaskID == nil || match.TaskDifficultySnapshot == nil {
		return Match{}, ErrTaskNotAssigned
	}

	return s.repo.StartMatch(ctx, input.MatchID)
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