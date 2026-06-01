package leaderboard

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestServiceGetPageNormalizesInputsAndCaches(t *testing.T) {
	repo := &fakeLeaderboardRepo{
		items: []Entry{{Rank: 1, UserID: uuid.New(), Username: "leader"}},
		total: 1,
		stats: Stats{RankedPlayers: 10, CachedAt: time.Now()},
	}
	service := NewService(repo)

	page, err := service.GetPage(context.Background(), " ", -1, -10, nil)
	if err != nil {
		t.Fatalf("GetPage returned error: %v", err)
	}
	if page.Mode != DefaultMode || page.Limit != DefaultLimit || page.Offset != DefaultOffset {
		t.Fatalf("page defaults = %#v", page)
	}
	if repo.listCalls != 1 || repo.statsCalls != 1 {
		t.Fatalf("repo calls = list %d stats %d", repo.listCalls, repo.statsCalls)
	}

	page.Items[0].Username = "mutated"
	cached, err := service.GetPage(context.Background(), DefaultMode, DefaultLimit, DefaultOffset, nil)
	if err != nil {
		t.Fatalf("cached GetPage returned error: %v", err)
	}
	if cached.Items[0].Username != "leader" {
		t.Fatalf("cached page was mutated: %#v", cached.Items)
	}
	if repo.listCalls != 1 || repo.statsCalls != 1 {
		t.Fatalf("cache miss: list %d stats %d", repo.listCalls, repo.statsCalls)
	}
}

func TestServiceGetPageClampsLimitAndLoadsViewerRank(t *testing.T) {
	viewerID := uuid.New()
	repo := &fakeLeaderboardRepo{
		items:      []Entry{},
		stats:      Stats{RankedPlayers: 10},
		viewerRank: &Entry{Rank: 5, UserID: viewerID, Username: "viewer"},
	}
	service := NewService(repo)

	page, err := service.GetPage(context.Background(), "regional", MaxLimit+50, 0, &viewerID)
	if err != nil {
		t.Fatalf("GetPage returned error: %v", err)
	}
	if page.Limit != MaxLimit {
		t.Fatalf("limit = %d, want %d", page.Limit, MaxLimit)
	}
	if page.ViewerRank == nil || page.ViewerRank.UserID != viewerID {
		t.Fatalf("viewer rank = %#v", page.ViewerRank)
	}
	if repo.viewerRankCalls != 1 {
		t.Fatalf("viewer rank calls = %d", repo.viewerRankCalls)
	}
}

type fakeLeaderboardRepo struct {
	items           []Entry
	total           int
	stats           Stats
	viewerRank      *Entry
	listCalls       int
	statsCalls      int
	viewerRankCalls int
}

func (r *fakeLeaderboardRepo) List(_ context.Context, _ string, _ int, _ int) ([]Entry, int, error) {
	r.listCalls++
	return append([]Entry(nil), r.items...), r.total, nil
}

func (r *fakeLeaderboardRepo) GetViewerRank(_ context.Context, _ string, _ uuid.UUID) (*Entry, error) {
	r.viewerRankCalls++
	return r.viewerRank, nil
}

func (r *fakeLeaderboardRepo) GetStats(context.Context, string) (Stats, error) {
	r.statsCalls++
	return r.stats, nil
}
