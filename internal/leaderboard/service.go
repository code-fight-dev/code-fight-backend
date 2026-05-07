package leaderboard

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetPage(ctx context.Context, mode string, limit int, offset int, viewerID *uuid.UUID) (Page, error) {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = DefaultMode
	}
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	if offset < 0 {
		offset = DefaultOffset
	}

	items, total, err := s.repo.List(ctx, mode, limit, offset)
	if err != nil {
		return Page{}, err
	}

	page := Page{
		Mode:   mode,
		Limit:  limit,
		Offset: offset,
		Total:  total,
		Items:  items,
	}

	if viewerID != nil && *viewerID != uuid.Nil {
		viewerRank, err := s.repo.GetViewerRank(ctx, mode, *viewerID)
		if err != nil {
			return Page{}, err
		}
		page.ViewerRank = viewerRank
	}

	return page, nil
}
