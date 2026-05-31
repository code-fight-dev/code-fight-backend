package leaderboard

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestHandlerGetLeaderboard(t *testing.T) {
	repo := &fakeLeaderboardHandlerRepo{
		items: []Entry{{Rank: 1, UserID: uuid.New(), Username: "ada", Rating: 1800}},
		total: 1,
		stats: Stats{RankedPlayers: 1},
	}
	app := fiber.New()
	NewHandler(NewService(repo), nil, "").RegisterRoutes(app)

	resp := leaderboardHandlerRequest(t, app, http.MethodGet, "/leaderboard?page=2&pageSize=25&mode=regional")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if repo.mode != "regional" || repo.limit != 25 || repo.offset != 25 {
		t.Fatalf("query mapping = mode %q, limit %d, offset %d", repo.mode, repo.limit, repo.offset)
	}

	var body Page
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Mode != "regional" || len(body.Items) != 1 || body.Stats.RankedPlayers != 1 {
		t.Fatalf("body = %#v", body)
	}
}

func TestHandlerGetLeaderboardRejectsInvalidQueries(t *testing.T) {
	app := fiber.New()
	NewHandler(NewService(&fakeLeaderboardHandlerRepo{}), nil, "").RegisterRoutes(app)

	tests := []string{
		"/leaderboard?limit=nope",
		"/leaderboard?pageSize=nope",
		"/leaderboard?offset=nope",
		"/leaderboard?page=0",
	}
	for _, path := range tests {
		resp := leaderboardHandlerRequest(t, app, http.MethodGet, path)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want 400", path, resp.StatusCode)
		}
	}
}

func leaderboardHandlerRequest(t *testing.T, app *fiber.App, method, path string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, path, strings.NewReader(""))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	return resp
}

type fakeLeaderboardHandlerRepo struct {
	items  []Entry
	total  int
	stats  Stats
	mode   string
	limit  int
	offset int
}

func (r *fakeLeaderboardHandlerRepo) List(_ context.Context, mode string, limit int, offset int) ([]Entry, int, error) {
	r.mode = mode
	r.limit = limit
	r.offset = offset
	return r.items, r.total, nil
}

func (r *fakeLeaderboardHandlerRepo) GetViewerRank(context.Context, string, uuid.UUID) (*Entry, error) {
	return nil, nil
}

func (r *fakeLeaderboardHandlerRepo) GetStats(context.Context, string) (Stats, error) {
	return r.stats, nil
}
