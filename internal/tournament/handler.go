package tournament

import (
	"context"
	"errors"
	"strconv"

	"code-fight-backend/internal/access"
	"code-fight-backend/internal/managementhttp"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type draftService interface {
	CreateDraft(context.Context, uuid.UUID, CreateInput, string) (Draft, error)
	GetDraft(context.Context, uuid.UUID, uuid.UUID) (Draft, error)
	ListDrafts(context.Context, uuid.UUID, int, string) (Page, error)
	UpdateDraft(context.Context, uuid.UUID, uuid.UUID, UpdateInput, string) (Draft, error)
}
type Handler struct{ service draftService }

func NewHandler(service draftService) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/tournaments", h.list)
	router.Post("/tournaments", h.create)
	router.Get("/tournaments/:id", h.get)
	router.Patch("/tournaments/:id", h.update)
}

func respondError(c *fiber.Ctx, err error) error {
	if errors.Is(err, ErrSlugTaken) {
		return c.Status(409).JSON(fiber.Map{"error": fiber.Map{"code": "slug_taken", "message": "This tournament slug is already in use", "requestId": managementhttp.RequestID(c)}})
	}
	return managementhttp.Error(c, err)
}

func (h *Handler) create(c *fiber.Ctx) error {
	c.Set("Cache-Control", "private, no-store")
	actorID, err := managementhttp.ActorID(c)
	if err != nil {
		return respondError(c, err)
	}
	var input CreateInput
	if err := managementhttp.Decode(c, &input); err != nil {
		return respondError(c, err)
	}
	draft, err := h.service.CreateDraft(c.UserContext(), actorID, input, managementhttp.RequestID(c))
	if err != nil {
		return respondError(c, err)
	}
	return c.Status(201).JSON(fiber.Map{"tournament": draft})
}
func (h *Handler) get(c *fiber.Ctx) error {
	c.Set("Cache-Control", "private, no-store")
	actorID, err := managementhttp.ActorID(c)
	if err != nil {
		return respondError(c, err)
	}
	id, err := managementhttp.ParamID(c, "id")
	if err != nil {
		return respondError(c, err)
	}
	draft, err := h.service.GetDraft(c.UserContext(), actorID, id)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(fiber.Map{"tournament": draft})
}
func (h *Handler) list(c *fiber.Ctx) error {
	c.Set("Cache-Control", "private, no-store")
	actorID, err := managementhttp.ActorID(c)
	if err != nil {
		return respondError(c, err)
	}
	limit, err := strconv.Atoi(c.Query("limit", "50"))
	if err != nil {
		return respondError(c, access.ErrInvalidInput)
	}
	page, err := h.service.ListDrafts(c.UserContext(), actorID, limit, c.Query("cursor"))
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(page)
}
func (h *Handler) update(c *fiber.Ctx) error {
	c.Set("Cache-Control", "private, no-store")
	actorID, err := managementhttp.ActorID(c)
	if err != nil {
		return respondError(c, err)
	}
	id, err := managementhttp.ParamID(c, "id")
	if err != nil {
		return respondError(c, err)
	}
	var input UpdateInput
	if err := managementhttp.Decode(c, &input); err != nil {
		return respondError(c, err)
	}
	draft, err := h.service.UpdateDraft(c.UserContext(), actorID, id, input, managementhttp.RequestID(c))
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(fiber.Map{"tournament": draft})
}
