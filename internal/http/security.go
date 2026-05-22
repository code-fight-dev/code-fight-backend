package http

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

func requireAllowedOrigin(allowedOrigins []string) fiber.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
			return c.Next()
		}

		origin := c.Get(fiber.HeaderOrigin)
		if origin == "" {
			return c.Next()
		}

		if _, ok := allowed[origin]; ok {
			return c.Next()
		}

		return fiber.NewError(fiber.StatusForbidden, "origin is not allowed")
	}
}

func newSubmissionLimiter(max int, window time.Duration) fiber.Handler {
	type bucket struct {
		count   int
		resetAt time.Time
	}

	if max <= 0 {
		max = 10
	}
	if window <= 0 {
		window = time.Minute
	}

	var (
		mu      sync.Mutex
		buckets = make(map[string]bucket)
	)

	return func(c *fiber.Ctx) error {
		if c.Method() != fiber.MethodPost {
			return c.Next()
		}

		scope, ok := submissionLimiterScope(c.Path())
		if !ok {
			return c.Next()
		}

		var userID string
		switch id := c.Locals("user_id").(type) {
		case string:
			userID = id
		default:
			if id != nil {
				userID = strings.TrimSpace(fmt.Sprint(id))
			}
		}
		if userID == "" {
			userID = c.IP()
		}

		key := userID + ":" + scope
		now := time.Now()

		mu.Lock()
		defer mu.Unlock()

		current := buckets[key]
		if current.resetAt.IsZero() || now.After(current.resetAt) {
			buckets[key] = bucket{
				count:   1,
				resetAt: now.Add(window),
			}
			return c.Next()
		}

		if current.count >= max {
			return fiber.NewError(fiber.StatusTooManyRequests, "too many submissions")
		}

		current.count++
		buckets[key] = current

		return c.Next()
	}
}

func submissionLimiterScope(path string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "tasks" && parts[3] == "submissions" {
		return "task:" + parts[2], true
	}
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "tasks" && parts[3] == "runs" {
		return "task-run:" + parts[2], true
	}
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "matches" && parts[3] == "submissions" {
		return "match:" + parts[2], true
	}

	return "", false
}

func newReplayCaptureLimiter(max int, window time.Duration) fiber.Handler {
	type bucket struct {
		count   int
		resetAt time.Time
	}

	if max <= 0 {
		max = 120
	}
	if window <= 0 {
		window = time.Minute
	}

	var (
		mu      sync.Mutex
		buckets = make(map[string]bucket)
	)

	return func(c *fiber.Ctx) error {
		if c.Method() != fiber.MethodPost {
			return c.Next()
		}

		scope, ok := replayCaptureLimiterScope(c.Path())
		if !ok {
			return c.Next()
		}

		var userID string
		switch id := c.Locals("user_id").(type) {
		case string:
			userID = id
		default:
			if id != nil {
				userID = strings.TrimSpace(fmt.Sprint(id))
			}
		}
		if userID == "" {
			userID = c.IP()
		}

		key := userID + ":" + scope
		now := time.Now()

		mu.Lock()
		defer mu.Unlock()

		current := buckets[key]
		if current.resetAt.IsZero() || now.After(current.resetAt) {
			buckets[key] = bucket{
				count:   1,
				resetAt: now.Add(window),
			}
			return c.Next()
		}

		if current.count >= max {
			return fiber.NewError(fiber.StatusTooManyRequests, "too many replay ingest requests")
		}

		current.count++
		buckets[key] = current

		return c.Next()
	}
}

func replayCaptureLimiterScope(path string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "api" || parts[1] != "matches" || parts[3] != "replay" {
		return "", false
	}

	switch parts[4] {
	case "events", "snapshots":
		return "replay:" + parts[2], true
	default:
		return "", false
	}
}

func newAuthLimiter() fiber.Handler {
	type bucket struct {
		count   int
		resetAt time.Time
	}

	var (
		mu      sync.Mutex
		buckets = make(map[string]bucket)
		window  = time.Minute
		max     = 15
	)

	return func(c *fiber.Ctx) error {
		if c.Method() == fiber.MethodOptions {
			return c.Next()
		}

		key := c.IP() + ":" + c.Path()
		now := time.Now()

		mu.Lock()
		defer mu.Unlock()

		current := buckets[key]
		if current.resetAt.IsZero() || now.After(current.resetAt) {
			current = bucket{
				count:   1,
				resetAt: now.Add(window),
			}
			buckets[key] = current
			return c.Next()
		}

		if current.count >= max {
			return fiber.NewError(fiber.StatusTooManyRequests, "too many auth attempts")
		}

		current.count++
		buckets[key] = current

		return c.Next()
	}
}
