package realtime

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

type Handler struct {
	broker *Broker
}

func NewHandler(broker *Broker) *Handler {
	return &Handler{broker: broker}
}

func (h *Handler) RegisterRoutes(router fiber.Router, sessionCheck func(*fiber.Ctx) func(context.Context, uuid.UUID) bool) {
	stream := func(c *fiber.Ctx) error { return h.Stream(c, sessionCheck(c)) }
	router.Get("", stream)
	router.Get("/", stream)
}

func (h *Handler) Stream(c *fiber.Ctx, checkSession func(context.Context, uuid.UUID) bool) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	events, unsubscribe := h.broker.Subscribe(userID)

	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "private, no-store, no-transform")
	c.Set(fiber.HeaderConnection, "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		defer unsubscribe()
		heartbeat := time.NewTicker(20 * time.Second)
		defer heartbeat.Stop()
		writeStream(w, events, heartbeat.C, func() bool {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			return checkSession(ctx, userID)
		})
	}))

	return nil
}

func writeStream(w *bufio.Writer, events <-chan Event, ticks <-chan time.Time, validSession func() bool) {
	if writeEvent(w, Event{ID: uuid.NewString(), Type: "connected", Data: json.RawMessage(`{"ok":true}`)}) != nil {
		return
	}
	for {
		select {
		case event, ok := <-events:
			if !ok || writeEvent(w, event) != nil {
				return
			}
		case <-ticks:
			if !validSession() {
				return
			}
			if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				return
			}
			if w.Flush() != nil {
				return
			}
		}
	}
}

func writeEvent(w *bufio.Writer, event Event) error {
	if event.ID != "" {
		fmt.Fprintf(w, "id: %s\n", event.ID)
	}
	fmt.Fprintf(w, "event: %s\n", event.Type)
	fmt.Fprintf(w, "data: %s\n\n", string(event.Data))
	return w.Flush()
}

func userIDFromCtx(c *fiber.Ctx) (uuid.UUID, error) {
	v := c.Locals("user_id")

	switch id := v.(type) {
	case uuid.UUID:
		if id == uuid.Nil {
			return uuid.Nil, errors.New("missing user_id")
		}
		return id, nil
	case string:
		parsed, err := uuid.Parse(id)
		if err != nil {
			return uuid.Nil, err
		}
		if parsed == uuid.Nil {
			return uuid.Nil, errors.New("missing user_id")
		}
		return parsed, nil
	default:
		return uuid.Nil, errors.New("missing user_id")
	}
}
