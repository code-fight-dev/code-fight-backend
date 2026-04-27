package tasks

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListChallenges(ctx context.Context) ([]ChallengeListItem, error) {
	return s.repo.ListChallenges(ctx)
}

func (s *Service) GetChallengeBySlug(ctx context.Context, slug string) (*Challenge, error) {
	return s.repo.GetChallengeBySlug(ctx, slug)
}

