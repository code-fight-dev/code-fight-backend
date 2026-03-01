package http

import (
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
