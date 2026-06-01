package realtime

import (
	"bufio"
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

func TestHandlerRegisterRoutesAndStreamRequiresAuth(t *testing.T) {
	app := fiber.New()
	NewHandler(NewBroker()).RegisterRoutes(app)

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestRealtimeUserIDFromCtx(t *testing.T) {
	userID := uuid.New()
	app := fiber.New()
	app.Get("/uuid", func(c *fiber.Ctx) error {
		c.Locals("user_id", userID)
		got, err := userIDFromCtx(c)
		if err != nil {
			return err
		}
		return c.SendString(got.String())
	})
	app.Get("/string", func(c *fiber.Ctx) error {
		c.Locals("user_id", userID.String())
		got, err := userIDFromCtx(c)
		if err != nil {
			return err
		}
		return c.SendString(got.String())
	})
	app.Get("/bad", func(c *fiber.Ctx) error {
		c.Locals("user_id", "bad")
		_, err := userIDFromCtx(c)
		if err == nil {
			return c.SendStatus(http.StatusOK)
		}
		return c.SendStatus(http.StatusBadRequest)
	})
	app.Get("/nil", func(c *fiber.Ctx) error {
		c.Locals("user_id", uuid.Nil)
		_, err := userIDFromCtx(c)
		if err == nil {
			return c.SendStatus(http.StatusOK)
		}
		return c.SendStatus(http.StatusBadRequest)
	})
	app.Get("/nil-string", func(c *fiber.Ctx) error {
		c.Locals("user_id", uuid.Nil.String())
		_, err := userIDFromCtx(c)
		if err == nil {
			return c.SendStatus(http.StatusOK)
		}
		return c.SendStatus(http.StatusBadRequest)
	})

	for _, path := range []string{"/uuid", "/string"} {
		resp := realtimeHandlerRequest(t, app, path)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, resp.StatusCode)
		}
	}
	for _, path := range []string{"/bad", "/nil", "/nil-string"} {
		resp := realtimeHandlerRequest(t, app, path)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want 400", path, resp.StatusCode)
		}
	}
}

func TestHandlerStreamSetsSSEHeaders(t *testing.T) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	ctx.Locals("user_id", uuid.New())
	handler := NewHandler(NewBroker())

	if err := handler.Stream(ctx); err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	if got := string(ctx.Response().Header.ContentType()); got != "text/event-stream" {
		t.Fatalf("content-type = %q, want text/event-stream", got)
	}
	if got := string(ctx.Response().Header.Peek(fiber.HeaderCacheControl)); got != "no-cache, no-transform" {
		t.Fatalf("cache-control = %q", got)
	}
}

func TestWriteEvent(t *testing.T) {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	writeEvent(writer, Event{ID: "event-1", Type: "match.started", Data: []byte(`{"ok":true}`)})

	got := buf.String()
	for _, want := range []string{"id: event-1", "event: match.started", `data: {"ok":true}`} {
		if !strings.Contains(got, want) {
			t.Fatalf("event output %q missing %q", got, want)
		}
	}
}

func realtimeHandlerRequest(t *testing.T, app *fiber.App, path string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	return resp
}
