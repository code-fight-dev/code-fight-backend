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
	ListPublished(context.Context) ([]Summary, error)
	History(context.Context, uuid.UUID, int, string) (HistoryPage, error)
	Board(context.Context, uuid.UUID) (Board, error)
	Join(context.Context, uuid.UUID, uuid.UUID) error
	Transition(context.Context, uuid.UUID, uuid.UUID, TransitionInput, string) error
	ScheduleFixture(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (uuid.UUID, error)
}
type Handler struct{ service draftService }

func NewHandler(service draftService) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/tournaments", h.list)
	router.Post("/tournaments", h.create)
	router.Get("/tournaments/:id", h.get)
	router.Patch("/tournaments/:id", h.update)
	router.Post("/tournaments/:id/status", h.transition)
	router.Post("/tournaments/:id/fixtures/:fixtureId", h.schedule)
}

func (h *Handler) RegisterPublicRoutes(router fiber.Router, requireAuth fiber.Handler) {
	router.Get("/tournaments", h.published)
	router.Get("/tournaments/history", requireAuth, h.history)
	router.Get("/tournaments/:id", h.board)
	router.Post("/tournaments/:id/join", requireAuth, h.join)
}

func (h *Handler) history(c *fiber.Ctx) error {
	c.Set("Cache-Control", "private, no-store")
	actor, err := managementhttp.ActorID(c)
	if err != nil {
		return respondError(c, err)
	}
	limit, err := strconv.Atoi(c.Query("limit", "50"))
	if err != nil {
		return respondError(c, access.ErrInvalidInput)
	}
	page, err := h.service.History(c.UserContext(), actor, limit, c.Query("cursor"))
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(page)
}

func (h *Handler) published(c *fiber.Ctx) error {
	items, err := h.service.ListPublished(c.UserContext())
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *Handler) board(c *fiber.Ctx) error {
	id, err := managementhttp.ParamID(c, "id")
	if err != nil {
		return respondError(c, err)
	}
	board, err := h.service.Board(c.UserContext(), id)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(board)
}

func (h *Handler) join(c *fiber.Ctx) error {
	actor, err := managementhttp.ActorID(c)
	if err != nil {
		return respondError(c, err)
	}
	id, err := managementhttp.ParamID(c, "id")
	if err != nil {
		return respondError(c, err)
	}
	if err := h.service.Join(c.UserContext(), actor, id); err != nil {
		return respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) transition(c *fiber.Ctx) error {
	actor, err := managementhttp.ActorID(c)
	if err != nil {
		return respondError(c, err)
	}
	id, err := managementhttp.ParamID(c, "id")
	if err != nil {
		return respondError(c, err)
	}
	var input TransitionInput
	if err := managementhttp.Decode(c, &input); err != nil {
		return respondError(c, err)
	}
	if err := h.service.Transition(c.UserContext(), actor, id, input, managementhttp.RequestID(c)); err != nil {
		return respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) schedule(c *fiber.Ctx) error {
	actor, err := managementhttp.ActorID(c)
	if err != nil {
		return respondError(c, err)
	}
	id, err := managementhttp.ParamID(c, "id")
	if err != nil {
		return respondError(c, err)
	}
	fixtureID, err := managementhttp.ParamID(c, "fixtureId")
	if err != nil {
		return respondError(c, err)
	}
	matchID, err := h.service.ScheduleFixture(c.UserContext(), actor, id, fixtureID, managementhttp.RequestID(c))
	if err != nil {
		return respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"matchId": matchID})
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
