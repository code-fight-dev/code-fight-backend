package administration

import (
	"context"

	"code-fight-backend/internal/managementhttp"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type userService interface {
	GetUser(context.Context, uuid.UUID, uuid.UUID) (User, error)
	ChangeRole(context.Context, uuid.UUID, uuid.UUID, ChangeRoleInput, string) (User, error)
}

type Handler struct{ service userService }

func NewHandler(service userService) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/users/:userId", h.getUser)
	router.Patch("/users/:userId/role", h.changeRole)
}

func (h *Handler) getUser(c *fiber.Ctx) error {
	c.Set("Cache-Control", "private, no-store")
	actorID, err := managementhttp.ActorID(c)
	if err != nil {
		return managementhttp.Error(c, err)
	}
	targetID, err := managementhttp.ParamID(c, "userId")
	if err != nil {
		return managementhttp.Error(c, err)
	}
	user, err := h.service.GetUser(c.UserContext(), actorID, targetID)
	if err != nil {
		return managementhttp.Error(c, err)
	}
	return c.JSON(fiber.Map{"user": user})
}

func (h *Handler) changeRole(c *fiber.Ctx) error {
	c.Set("Cache-Control", "private, no-store")
	actorID, err := managementhttp.ActorID(c)
	if err != nil {
		return managementhttp.Error(c, err)
	}
	targetID, err := managementhttp.ParamID(c, "userId")
	if err != nil {
		return managementhttp.Error(c, err)
	}
	var input ChangeRoleInput
	if err := managementhttp.Decode(c, &input); err != nil {
		return managementhttp.Error(c, err)
	}
	user, err := h.service.ChangeRole(c.UserContext(), actorID, targetID, input, managementhttp.RequestID(c))
	if err != nil {
		return managementhttp.Error(c, err)
	}
	return c.JSON(fiber.Map{"user": user})
}
