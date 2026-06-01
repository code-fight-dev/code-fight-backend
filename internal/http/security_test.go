package http

import (
	nethttp "net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestRequireAllowedOrigin(t *testing.T) {
	app := fiber.New()
	app.Use(requireAllowedOrigin([]string{"https://app.example.com"}))
	app.Post("/api/write", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	app.Get("/api/read", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })

	if status := testRequestStatus(t, app, nethttp.MethodPost, "/api/write", "https://app.example.com"); status != fiber.StatusNoContent {
		t.Fatalf("allowed status = %d", status)
	}
	if status := testRequestStatus(t, app, nethttp.MethodPost, "/api/write", "https://evil.example.com"); status != fiber.StatusForbidden {
		t.Fatalf("forbidden status = %d", status)
	}
	if status := testRequestStatus(t, app, nethttp.MethodGet, "/api/read", "https://evil.example.com"); status != fiber.StatusNoContent {
		t.Fatalf("GET status = %d", status)
	}
	if status := testRequestStatus(t, app, nethttp.MethodPost, "/api/write", ""); status != fiber.StatusNoContent {
		t.Fatalf("empty origin status = %d", status)
	}
}

func TestSubmissionLimiterScope(t *testing.T) {
	tests := []struct {
		path string
		want string
		ok   bool
	}{
		{path: "/api/tasks/task-1/submissions", want: "task:task-1", ok: true},
		{path: "/api/tasks/task-1/runs", want: "task-run:task-1", ok: true},
		{path: "/api/matches/match-1/submissions", want: "match:match-1", ok: true},
		{path: "/api/tasks/task-1", ok: false},
	}

	for _, tt := range tests {
		got, ok := submissionLimiterScope(tt.path)
		if got != tt.want || ok != tt.ok {
			t.Fatalf("submissionLimiterScope(%q) = %q/%v", tt.path, got, ok)
		}
	}
}

func TestSubmissionLimiter(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return c.Next()
	})
	app.Use(newSubmissionLimiter(1, time.Minute))
	app.Post("/api/tasks/task-1/submissions", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })

	if status := testRequestStatus(t, app, nethttp.MethodPost, "/api/tasks/task-1/submissions", ""); status != fiber.StatusNoContent {
		t.Fatalf("first status = %d", status)
	}
	if status := testRequestStatus(t, app, nethttp.MethodPost, "/api/tasks/task-1/submissions", ""); status != fiber.StatusTooManyRequests {
		t.Fatalf("second status = %d", status)
	}

	defaultApp := fiber.New()
	defaultApp.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", uuidStringer("user-2"))
		return c.Next()
	})
	defaultApp.Use(newSubmissionLimiter(0, 0))
	defaultApp.Post("/api/tasks/task-2/runs", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	defaultApp.Get("/api/tasks/task-2/runs", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	if status := testRequestStatus(t, defaultApp, nethttp.MethodGet, "/api/tasks/task-2/runs", ""); status != fiber.StatusNoContent {
		t.Fatalf("GET limiter status = %d", status)
	}
	if status := testRequestStatus(t, defaultApp, nethttp.MethodPost, "/api/other", ""); status != fiber.StatusNotFound {
		t.Fatalf("unscoped limiter status = %d", status)
	}

	incrementApp := fiber.New()
	incrementApp.Use(newSubmissionLimiter(2, time.Minute))
	incrementApp.Post("/api/matches/match-1/submissions", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	for i := 0; i < 2; i++ {
		if status := testRequestStatus(t, incrementApp, nethttp.MethodPost, "/api/matches/match-1/submissions", ""); status != fiber.StatusNoContent {
			t.Fatalf("increment limiter status %d = %d", i, status)
		}
	}
}

func TestReplayCaptureLimiterScope(t *testing.T) {
	tests := []struct {
		path string
		want string
		ok   bool
	}{
		{path: "/api/matches/match-1/replay/events", want: "replay:match-1", ok: true},
		{path: "/api/matches/match-1/replay/snapshots", want: "replay:match-1", ok: true},
		{path: "/api/matches/match-1/replay/other", ok: false},
	}

	for _, tt := range tests {
		got, ok := replayCaptureLimiterScope(tt.path)
		if got != tt.want || ok != tt.ok {
			t.Fatalf("replayCaptureLimiterScope(%q) = %q/%v", tt.path, got, ok)
		}
	}
}

func TestReplayCaptureLimiterAndAuthLimiter(t *testing.T) {
	app := fiber.New()
	app.Use(newReplayCaptureLimiter(1, time.Minute))
	app.Post("/api/matches/match-1/replay/events", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })

	if status := testRequestStatus(t, app, nethttp.MethodPost, "/api/matches/match-1/replay/events", ""); status != fiber.StatusNoContent {
		t.Fatalf("first replay status = %d", status)
	}
	if status := testRequestStatus(t, app, nethttp.MethodPost, "/api/matches/match-1/replay/events", ""); status != fiber.StatusTooManyRequests {
		t.Fatalf("second replay status = %d", status)
	}

	defaultReplayApp := fiber.New()
	defaultReplayApp.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", uuidStringer("user-3"))
		return c.Next()
	})
	defaultReplayApp.Use(newReplayCaptureLimiter(0, 0))
	defaultReplayApp.Get("/api/matches/match-1/replay/events", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	defaultReplayApp.Post("/api/matches/match-1/replay/other", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	if status := testRequestStatus(t, defaultReplayApp, nethttp.MethodGet, "/api/matches/match-1/replay/events", ""); status != fiber.StatusNoContent {
		t.Fatalf("GET replay limiter status = %d", status)
	}
	if status := testRequestStatus(t, defaultReplayApp, nethttp.MethodPost, "/api/matches/match-1/replay/other", ""); status != fiber.StatusNoContent {
		t.Fatalf("unscoped replay limiter status = %d", status)
	}

	replayIncrementApp := fiber.New()
	replayIncrementApp.Use(newReplayCaptureLimiter(2, time.Minute))
	replayIncrementApp.Post("/api/matches/match-2/replay/snapshots", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	for i := 0; i < 2; i++ {
		if status := testRequestStatus(t, replayIncrementApp, nethttp.MethodPost, "/api/matches/match-2/replay/snapshots", ""); status != fiber.StatusNoContent {
			t.Fatalf("replay increment limiter status %d = %d", i, status)
		}
	}

	authApp := fiber.New()
	authApp.Use(newAuthLimiter())
	authApp.Post("/api/auth/signin", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	authApp.Options("/api/auth/signin", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	if status := testRequestStatus(t, authApp, nethttp.MethodOptions, "/api/auth/signin", ""); status != fiber.StatusNoContent {
		t.Fatalf("auth options status = %d", status)
	}
	for i := 0; i < 15; i++ {
		if status := testRequestStatus(t, authApp, nethttp.MethodPost, "/api/auth/signin", ""); status != fiber.StatusNoContent {
			t.Fatalf("auth status %d = %d", i, status)
		}
	}
	if status := testRequestStatus(t, authApp, nethttp.MethodPost, "/api/auth/signin", ""); status != fiber.StatusTooManyRequests {
		t.Fatalf("limited auth status = %d", status)
	}
}

func testRequestStatus(t *testing.T, app *fiber.App, method string, path string, origin string) int {
	t.Helper()

	req, err := nethttp.NewRequest(method, path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if origin != "" {
		req.Header.Set(fiber.HeaderOrigin, origin)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode
}

type uuidStringer string

func (s uuidStringer) String() string {
	return string(s)
}
