package managementhttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"code-fight-backend/internal/access"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func ErrorBoundary(c *fiber.Ctx) error {
	c.Set("Cache-Control", "private, no-store")
	RequestID(c)
	if err := c.Next(); err != nil {
		return Error(c, err)
	}
	return nil
}

func Decode(c *fiber.Ctx, target any) error {
	mediaType, _, err := mime.ParseMediaType(c.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return access.ErrInvalidInput
	}
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return access.ErrInvalidInput
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return access.ErrInvalidInput
	}
	return nil
}

func ActorID(c *fiber.Ctx) (uuid.UUID, error) {
	id, ok := c.Locals("user_id").(uuid.UUID)
	if !ok || id == uuid.Nil {
		return uuid.Nil, fiber.ErrUnauthorized
	}
	return id, nil
}

func ParamID(c *fiber.Ctx, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(name))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, access.ErrInvalidInput
	}
	return id, nil
}

func RequestID(c *fiber.Ctx) string {
	if id, ok := c.Locals("management_request_id").(string); ok {
		return id
	}
	id := uuid.NewString()
	c.Locals("management_request_id", id)
	c.Set("X-Request-ID", id)
	return id
}

func Error(c *fiber.Ctx, err error) error {
	status, code, message := 500, "internal_error", "Internal server error"
	var httpError *fiber.Error
	httpStatus := 0
	if errors.As(err, &httpError) {
		httpStatus = httpError.Code
	}
	switch {
	case httpStatus == 401:
		status, code, message = 401, "unauthenticated", "Sign in to continue"
	case errors.Is(err, access.ErrForbidden) || httpStatus == 403:
		status, code, message = 403, "forbidden", "You do not have permission to perform this action"
	case errors.Is(err, access.ErrNotFound) || httpStatus == 404:
		status, code, message = 404, "not_found", "Resource not found"
	case errors.Is(err, access.ErrInvalidInput) || httpStatus == 400:
		status, code, message = 400, "invalid_request", "Invalid request"
	case errors.Is(err, access.ErrConflict) || httpStatus == 409:
		status, code, message = 409, "version_conflict", "The resource changed; reload it before saving"
	case httpStatus >= 400 && httpStatus < 500:
		status, code, message = httpStatus, "request_failed", http.StatusText(httpStatus)
		if message == "" {
			message = "Request failed"
		}
	case httpStatus >= 500 && httpStatus < 600:
		status = httpStatus
	}
	if status >= 500 {
		slog.ErrorContext(c.UserContext(), "management.request.failed",
			"request_id", RequestID(c), "method", c.Method(),
			"route_template", c.Route().Path, "status", status,
			"error", logError(err),
		)
	}
	return c.Status(status).JSON(fiber.Map{"error": fiber.Map{"code": code, "message": message, "requestId": RequestID(c)}})
}

func logError(err error) any {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return slog.GroupValue(
			slog.String("type", "postgres"), slog.String("sqlstate", pgErr.Code),
			slog.String("table", pgErr.TableName), slog.String("constraint", pgErr.ConstraintName),
		)
	}
	var connectErr *pgconn.ConnectError
	if errors.As(err, &connectErr) {
		return "PostgreSQL connection failed"
	}
	return err
}
