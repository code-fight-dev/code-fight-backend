package managementhttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"code-fight-backend/internal/access"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
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

func TestUnexpectedErrorsAreLoggedOnceWithRequestID(t *testing.T) {
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	for _, test := range []struct {
		name    string
		err     error
		handled bool
		status  int
	}{
		{"returned error", errors.New("audit storage unavailable"), false, 500},
		{"handled error", errors.New("audit storage unavailable"), true, 500},
		{"database error", fmt.Errorf("sensitive-wrapper: %w", &pgconn.PgError{
			Code: "23514", Message: "sensitive-message", Detail: "sensitive-detail", Where: "sensitive-sql",
			TableName: "tournaments", ConstraintName: "tournaments_status_check",
		}), true, 500},
		{"connection error", &pgconn.ConnectError{}, true, 500},
		{"expected rejection", access.ErrInvalidInput, true, 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			logs.Reset()
			app := fiber.New()
			app.Use(ErrorBoundary)
			app.Patch("/tournaments/:id", func(c *fiber.Ctx) error {
				if test.handled {
					return Error(c, test.err)
				}
				return test.err
			})
			req := httptest.NewRequest("PATCH", "/tournaments/private-id?token=sensitive-query", strings.NewReader(`{"title":"sensitive-body"}`))
			req.Header.Set("Authorization", "Bearer sensitive-token")
			req.Header.Set("Cookie", "sid=sensitive-cookie")
			req.Header.Set("X-Request-ID", "sensitive-client-id")
			response, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != test.status {
				t.Fatalf("status = %d, body = %s", response.StatusCode, body)
			}
			if test.status < 500 {
				if logs.Len() != 0 {
					t.Fatalf("expected client rejection produced an error log: %s", logs.String())
				}
				return
			}
			var record map[string]any
			decoder := json.NewDecoder(bytes.NewReader(logs.Bytes()))
			if err := decoder.Decode(&record); err != nil {
				t.Fatalf("missing structured error log: %s: %v", logs.String(), err)
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				t.Fatalf("expected exactly one error log: %s", logs.String())
			}
			if record["level"] != "ERROR" || record["request_id"] != response.Header.Get("X-Request-ID") ||
				record["route_template"] != "/tournaments/:id" || record["method"] != "PATCH" || record["status"] != float64(500) {
				t.Fatalf("uncorrelated error log: %s", logs.String())
			}
			if strings.Contains(logs.String(), "sensitive-") || strings.Contains(logs.String(), "private-id") {
				t.Fatalf("error log exposed request data: %s", logs.String())
			}
			if test.name == "database error" {
				detail, ok := record["error"].(map[string]any)
				if !ok || detail["sqlstate"] != "23514" || detail["constraint"] != "tournaments_status_check" {
					t.Fatalf("missing database diagnostics: %s", logs.String())
				}
			} else if test.name == "connection error" {
				if record["error"] != "PostgreSQL connection failed" {
					t.Fatalf("missing connection failure: %s", logs.String())
				}
			} else if record["error"] != "audit storage unavailable" {
				t.Fatalf("underlying error was lost: %s", logs.String())
			}
			if strings.Contains(string(body), "sensitive-") || strings.Contains(string(body), "audit storage") || !strings.Contains(string(body), "internal_error") {
				t.Fatalf("unsafe public error response: %s", body)
			}
		})
	}
}
