package http

import (
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"code-fight-backend/internal/administration"
	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/config"
	"code-fight-backend/internal/leaderboard"
	"code-fight-backend/internal/matches"
	"code-fight-backend/internal/realtime"
	"code-fight-backend/internal/submissions"
	"code-fight-backend/internal/tasks"
	"code-fight-backend/internal/tournament"
	"code-fight-backend/internal/users"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestNewRegistersHealthRoutes(t *testing.T) {
	app := newServerTestApp()
	for _, path := range []string{"/health", "/health/live"} {
		req, err := nethttp.NewRequest(nethttp.MethodGet, path, nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app test: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != nethttp.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, resp.StatusCode)
		}
	}
}

func TestNewScopesManagementErrorBoundary(t *testing.T) {
	app := newServerTestApp()
	for _, test := range []struct {
		name, method, path, origin string
		status                     int
		code                       string
	}{
		{"admin auth", "GET", "/api/admin/users/any", "", 401, "unauthenticated"},
		{"organizer auth", "POST", "/api/organizer/tournaments", "", 401, "unauthenticated"},
		{"admin origin", "PATCH", "/api/admin/users/any/role", "https://untrusted.test", 403, "forbidden"},
		{"organizer origin", "POST", "/api/organizer/tournaments", "https://untrusted.test", 403, "forbidden"},
		{"legacy origin", "POST", "/api/tasks/any/submissions", "https://untrusted.test", 403, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, nil)
			req.Header.Set("Origin", test.origin)
			response, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			var body struct {
				Error json.RawMessage `json:"error"`
			}
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != test.status {
				t.Fatalf("status = %d, want %d: %s", response.StatusCode, test.status, body.Error)
			}
			if test.code == "" {
				var legacyMessage string
				if err := json.Unmarshal(body.Error, &legacyMessage); err != nil || legacyMessage != "origin is not allowed" {
					t.Fatalf("legacy error contract changed: %s", body.Error)
				}
				if response.Header.Get("X-Request-ID") != "" {
					t.Fatal("management boundary ran for legacy API")
				}
				return
			}
			var detail struct {
				Code, Message string
				RequestID     string `json:"requestId"`
			}
			if err := json.Unmarshal(body.Error, &detail); err != nil || detail.Code != test.code || detail.Message == "" {
				t.Fatalf("invalid management error: %s", body.Error)
			}
			if _, err := uuid.Parse(detail.RequestID); err != nil || detail.RequestID != response.Header.Get("X-Request-ID") {
				t.Fatalf("missing/mismatched request ID: %s", body.Error)
			}
			if response.Header.Get("Cache-Control") != "private, no-store" {
				t.Fatal("management error may be cached")
			}
		})
	}
}

func newServerTestApp() *fiber.App {
	return New(
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
		administration.NewHandler(nil),
		tournament.NewHandler(nil),
	)
}
