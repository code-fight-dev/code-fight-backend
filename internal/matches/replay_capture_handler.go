package matches

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type replayEventPayloadRequest struct {
	Seq       int64  `json:"seq"`
	TMs       int64  `json:"t_ms"`
	EventType string `json:"event_type"`
	Payload   any    `json:"payload"`
}

type ingestReplayEventsRequest struct {
	Events []replayEventPayloadRequest `json:"events"`
}

type ingestReplaySnapshotRequest struct {
	Seq        int64  `json:"seq"`
	TMs        int64  `json:"t_ms"`
	Language   string `json:"language"`
	SourceCode string `json:"source_code"`
}

func (h *Handler) IngestMatchReplayEvents(c *fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	matchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid match id"})
	}

	var req ingestReplayEventsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	events := make([]MatchReplayEventIngestItem, 0, len(req.Events))
	for _, item := range req.Events {
		payload := item.Payload
		if payload == nil {
			payload = map[string]any{}
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid event payload"})
		}

		events = append(events, MatchReplayEventIngestItem{
			Seq:       item.Seq,
			TMs:       item.TMs,
			EventType: item.EventType,
			Payload:   payloadBytes,
		})
	}

	result, err := h.service.IngestMatchReplayEvents(c.UserContext(), MatchReplayEventsIngestInput{
		MatchID: matchID,
		UserID:  userID,
		Events:  events,
	})
	if err != nil {
		return writeMatchError(c, err)
	}

	return c.Status(fiber.StatusAccepted).JSON(result)
}

func (h *Handler) IngestMatchReplaySnapshot(c *fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	matchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid match id"})
	}

	var req ingestReplaySnapshotRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	result, err := h.service.IngestMatchReplaySnapshot(c.UserContext(), MatchReplaySnapshotIngestInput{
		MatchID:    matchID,
		UserID:     userID,
		Seq:        req.Seq,
		TMs:        req.TMs,
		Language:   req.Language,
		SourceCode: req.SourceCode,
	})
	if err != nil {
		return writeMatchError(c, err)
	}

	return c.Status(fiber.StatusAccepted).JSON(result)
}
