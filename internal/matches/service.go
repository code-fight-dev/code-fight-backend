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
	repo RepositoryInterface
}

func NewService(repo RepositoryInterface) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateMatch(ctx context.Context, input CreateMatchInput) (Match, error) {
	if input.Player1ID == uuid.Nil || input.Player2ID == uuid.Nil {
		return Match{}, fmt.Errorf("create match: empty player id")
	}
	if input.Player1ID == input.Player2ID {
		return Match{}, ErrSamePlayer
	}
	if input.Mode == "" {
		input.Mode = "global"
	}

	match := Match{
		ID:            uuid.New(),
		Mode:          input.Mode,
		Player1ID:     input.Player1ID,
		Player2ID:     input.Player2ID,
		Status:        MatchStatusPending,
		IsRated:       input.IsRated,
		RatingApplied: false,
	}

	return s.repo.CreateMatch(ctx, match)
}

func (s *Service) StartMatch(ctx context.Context, input StartMatchInput) (Match, error) {
	if input.MatchID == uuid.Nil {
		return Match{}, fmt.Errorf("start match: empty match id")
	}

	return s.repo.StartMatch(ctx, input.MatchID)
}

func (s *Service) FinishMatch(ctx context.Context, input FinishMatchInput) error {
	if input.MatchID == uuid.Nil {
		return fmt.Errorf("finish match: empty match id")
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