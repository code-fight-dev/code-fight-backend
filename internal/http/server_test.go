package http

import (
	nethttp "net/http"
	"testing"
	"time"

	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/config"
	"code-fight-backend/internal/leaderboard"
	"code-fight-backend/internal/matches"
	"code-fight-backend/internal/realtime"
	"code-fight-backend/internal/submissions"
	"code-fight-backend/internal/tasks"
	"code-fight-backend/internal/users"
)

func TestNewRegistersHealthRoutes(t *testing.T) {
	app := New(
		config.Config{
			AllowedOrigins:       []string{"http://localhost:3000"},
			MaxBodyBytes:         1024,
			SubmissionRateLimit:  10,
			SubmissionRateWindow: time.Minute,
		},
		auth.NewHandler(nil, nil, nil, "http://frontend.test", "sid", "", "Lax", "oauth_state", time.Hour, time.Minute, false),
		users.NewHandler(nil, nil, "sid"),
		tasks.NewHandler(nil, nil, ""),
		leaderboard.NewHandler(nil, nil, ""),
		matches.NewHandler(nil, time.Minute),
		realtime.NewHandler(realtime.NewBroker()),
		submissions.NewHandler(nil, "token"),
		nil,
		nil,
	)

	for _, path := range []string{"/health", "/health/live"} {
		req, err := nethttp.NewRequest(nethttp.MethodGet, path, nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app test: %v", err)
		}
		if resp.StatusCode != nethttp.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, resp.StatusCode)
		}
	}
}
