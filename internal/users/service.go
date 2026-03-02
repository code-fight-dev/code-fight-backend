package users

import (
	"context"
	"fmt"
	"strings"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) GetProfileByUsername(ctx context.Context, username string) (Profile, error) {
	normalizedUsername := strings.TrimSpace(username)
	if normalizedUsername == "" {
		return Profile{}, fmt.Errorf("%w: username is required", ErrInvalidInput)
	}

	profile, err := s.repository.GetProfileByUsername(ctx, normalizedUsername)
	if err != nil {
		return Profile{}, err
	}

	return profile, nil
}
