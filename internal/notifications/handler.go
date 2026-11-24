package notifications

import (
	"code-fight-backend/internal/managementhttp"
	"code-fight-backend/internal/realtime"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func RegisterRoutes(router fiber.Router, store *Store, broker *realtime.Broker) {
	router.Get("/", func(c *fiber.Ctx) error {
		userID, err := managementhttp.ActorID(c)
		if err != nil {
			return fiber.ErrUnauthorized
		}
		var before *uuid.UUID
		if cursor := c.Query("before"); cursor != "" {
			id, err := uuid.Parse(cursor)
			if err != nil {
				return fiber.ErrBadRequest
			}
			before = &id
		}
		page, err := store.List(c.UserContext(), userID, before)
		if err != nil {
			return err
		}
		c.Set(fiber.HeaderCacheControl, "private, no-store")
		return c.JSON(page)
	})
	markRead := func(c *fiber.Ctx) error {
		userID, err := managementhttp.ActorID(c)
		if err != nil {
			return fiber.ErrUnauthorized
		}
		var target *uuid.UUID
		if c.Params("id") != "" {
			id, err := managementhttp.ParamID(c, "id")
			if err != nil {
				return fiber.ErrBadRequest
			}
			target = &id
		}
		if err := store.MarkRead(c.UserContext(), userID, target); err != nil {
			return err
		}
		broker.Publish(userID, ChangedEvent, nil)
		return c.SendStatus(fiber.StatusNoContent)
	}
	router.Post("/read", markRead)
	router.Post("/:id/read", markRead)
}
