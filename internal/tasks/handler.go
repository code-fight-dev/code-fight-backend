package tasks

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/tasks", h.ListChallenges)
	router.Get("/tasks/:slug", h.GetChallengeBySlug)
}

func (h *Handler) ListChallenges(c *fiber.Ctx) error {
	items, err := h.service.ListChallenges(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"items": items,
	})
}

func (h *Handler) GetChallengeBySlug(c *fiber.Ctx) error {
	slug := strings.TrimSpace(c.Params("slug"))
	if slug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "slug is required")
	}

	challenge, err := h.service.GetChallengeBySlug(c.Context(), slug)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if challenge == nil {
		return fiber.NewError(fiber.StatusNotFound, "task not found")
	}

	return c.JSON(fiber.Map{
		"task": challenge,
	})
}

