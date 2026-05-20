package leaderboard

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	pageCacheTTL  = 3 * time.Minute
	statsCacheTTL = 30 * time.Minute
)

type Service struct {
	repo *Repository

	pageCacheMu sync.RWMutex
	pageCache   map[string]cachedPage

	statsCacheMu sync.RWMutex
	statsCache   map[string]cachedStats
}

type cachedPage struct {
	page      Page
	expiresAt time.Time
}

type cachedStats struct {
	stats     Stats
	expiresAt time.Time
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo:       repo,
		pageCache:  make(map[string]cachedPage),
		statsCache: make(map[string]cachedStats),
	}
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

	page, err := s.getCachedPage(ctx, mode, limit, offset)
	if err != nil {
		return Page{}, err
	}

	stats, err := s.getCachedStats(ctx, mode)
	if err != nil {
		return Page{}, err
	}
	page.Stats = stats

	if viewerID != nil && *viewerID != uuid.Nil {
		viewerRank, err := s.repo.GetViewerRank(ctx, mode, *viewerID)
		if err != nil {
			return Page{}, err
		}
		page.ViewerRank = viewerRank
	}

	return page, nil
}

func (s *Service) getCachedPage(ctx context.Context, mode string, limit int, offset int) (Page, error) {
	now := time.Now().UTC()
	key := fmt.Sprintf("%s:%d:%d", mode, limit, offset)

	s.pageCacheMu.RLock()
	if cached, ok := s.pageCache[key]; ok && now.Before(cached.expiresAt) {
		page := clonePage(cached.page)
		s.pageCacheMu.RUnlock()
		return page, nil
	}
	s.pageCacheMu.RUnlock()

	s.pageCacheMu.Lock()
	defer s.pageCacheMu.Unlock()

	if cached, ok := s.pageCache[key]; ok && now.Before(cached.expiresAt) {
		return clonePage(cached.page), nil
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
	s.pageCache[key] = cachedPage{
		page:      clonePage(page),
		expiresAt: now.Add(pageCacheTTL),
	}

	return page, nil
}

func (s *Service) getCachedStats(ctx context.Context, mode string) (Stats, error) {
	now := time.Now().UTC()

	s.statsCacheMu.RLock()
	if cached, ok := s.statsCache[mode]; ok && now.Before(cached.expiresAt) {
		stats := cached.stats
		s.statsCacheMu.RUnlock()
		return stats, nil
	}
	s.statsCacheMu.RUnlock()

	s.statsCacheMu.Lock()
	defer s.statsCacheMu.Unlock()

	if cached, ok := s.statsCache[mode]; ok && now.Before(cached.expiresAt) {
		return cached.stats, nil
	}

	stats, err := s.repo.GetStats(ctx, mode)
	if err != nil {
		return Stats{}, err
	}
	s.statsCache[mode] = cachedStats{
		stats:     stats,
		expiresAt: now.Add(statsCacheTTL),
	}

	return stats, nil
}

func clonePage(page Page) Page {
	page.Items = append([]Entry(nil), page.Items...)
	if page.ViewerRank != nil {
		viewerRank := *page.ViewerRank
		page.ViewerRank = &viewerRank
	}
	return page
}
