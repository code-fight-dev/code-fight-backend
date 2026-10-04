package managementhttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"code-fight-backend/internal/access"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestErrorBoundaryNormalizesErrors(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"auth new error", fiber.NewError(401, "private detail"), 401, "unauthenticated"},
		{"wrapped auth error", fmt.Errorf("wrapped: %w", fiber.ErrUnauthorized), 401, "unauthenticated"},
		{"role error", fiber.ErrForbidden, 403, "forbidden"},
		{"origin error", fiber.NewError(403, "private detail"), 403, "forbidden"},
		{"domain permission", access.ErrForbidden, 403, "forbidden"},
		{"missing route", fiber.NewError(404, "private detail"), 404, "not_found"},
		{"domain not found", access.ErrNotFound, 404, "not_found"},
		{"domain validation", access.ErrInvalidInput, 400, "invalid_request"},
		{"domain conflict", access.ErrConflict, 409, "version_conflict"},
		{"other client error", fiber.ErrMethodNotAllowed, 405, "request_failed"},
		{"nonstandard client error", fiber.NewError(499, "private detail"), 499, "request_failed"},
		{"internal error", errors.New("private detail"), 500, "internal_error"},
		{"http internal error", fiber.NewError(503, "private detail"), 503, "internal_error"},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error {
				t.Errorf("error escaped management boundary: %v", err)
				return c.SendStatus(500)
			}})
			app.Use(ErrorBoundary)
			app.Get("/protected", func(c *fiber.Ctx) error { return test.err })
			req := httptest.NewRequest("GET", "/protected", nil)
			req.Header.Set("X-Request-ID", "client-controlled")
			response, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			var envelope struct {
				Error struct {
					Code      string `json:"code"`
					Message   string `json:"message"`
					RequestID string `json:"requestId"`
				} `json:"error"`
			}
			if err := json.Unmarshal(body, &envelope); err != nil {
				t.Fatalf("invalid error envelope: %s: %v", body, err)
			}
			if response.StatusCode != test.status || envelope.Error.Code != test.code || envelope.Error.Message == "" {
				t.Fatalf("status %d, body %s; want %d/%s", response.StatusCode, body, test.status, test.code)
			}
			if _, err := uuid.Parse(envelope.Error.RequestID); err != nil || envelope.Error.RequestID != response.Header.Get("X-Request-ID") {
				t.Fatalf("missing/mismatched server request ID: %s", body)
			}
			if strings.Contains(string(body), "private detail") || response.Header.Get("Cache-Control") != "private, no-store" {
				t.Fatalf("unsafe error response: %s", body)
			}
		})
	}
}

func TestErrorBoundaryPreservesHandledResponse(t *testing.T) {
	app := fiber.New()
	app.Use(ErrorBoundary)
	app.Get("/handled", func(c *fiber.Ctx) error {
		id := RequestID(c)
		if id != RequestID(c) {
			t.Error("request ID changed within request")
		}
		return c.Status(201).JSON(fiber.Map{"id": id})
	})
	response, err := app.Test(httptest.NewRequest("GET", "/handled", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 201 || len(body) != 1 || body["id"] != response.Header.Get("X-Request-ID") {
		t.Fatalf("handled response was changed: %d %v", response.StatusCode, body)
	}
}
