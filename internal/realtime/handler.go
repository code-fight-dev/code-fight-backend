package realtime

import (
	"bufio"
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

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("", h.Stream)
	router.Get("/", h.Stream)
}

func (h *Handler) Stream(c *fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	events, unsubscribe := h.broker.Subscribe(userID)

	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache, no-transform")
	c.Set(fiber.HeaderConnection, "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		defer unsubscribe()

		writeEvent(w, Event{
			ID:   uuid.NewString(),
			Type: "connected",
			Data: json.RawMessage(`{"ok":true}`),
		})

		heartbeat := time.NewTicker(20 * time.Second)
		defer heartbeat.Stop()

		ctx := c.UserContext()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				writeEvent(w, event)
			case <-heartbeat.C:
				if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	}))

	return nil
}

func writeEvent(w *bufio.Writer, event Event) {
	if event.ID != "" {
		fmt.Fprintf(w, "id: %s\n", event.ID)
	}
	fmt.Fprintf(w, "event: %s\n", event.Type)
	fmt.Fprintf(w, "data: %s\n\n", string(event.Data))
	_ = w.Flush()
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
