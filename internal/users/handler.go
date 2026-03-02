package users

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
}

type profileResponse struct {
	Profile Profile `json:"profile"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/:username", h.profileByUsername)
}

func (h *Handler) profileByUsername(c *fiber.Ctx) error {
	profile, err := h.service.GetProfileByUsername(c.UserContext(), c.Params("username"))
	if err != nil {
		return mapServiceError(err)
	}

	return c.JSON(profileResponse{Profile: profile})
}

func mapServiceError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	default:
		return err
	}
}
