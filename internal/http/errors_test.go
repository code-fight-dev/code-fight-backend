package http

import (
	"encoding/json"
	"errors"
	nethttp "net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestJSONErrorHandler(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: jsonErrorHandler})
	app.Get("/fiber", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusTeapot, "short and stout")
	})
	app.Get("/plain", func(c *fiber.Ctx) error {
		return errors.New("boom")
	})

	resp, err := app.Test(mustRequest(t, nethttp.MethodGet, "/fiber"))
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	if resp.StatusCode != fiber.StatusTeapot {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	_ = resp.Body.Close()
	if body["error"] != "short and stout" {
		t.Fatalf("body = %#v", body)
	}

	resp, err = app.Test(mustRequest(t, nethttp.MethodGet, "/plain"))
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func mustRequest(t *testing.T, method string, path string) *nethttp.Request {
	t.Helper()
	req, err := nethttp.NewRequest(method, path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	return req
}
