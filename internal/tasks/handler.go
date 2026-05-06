package tasks

import (
	"errors"
	"strings"

	"code-fight-backend/internal/auth"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service     *Service
	authService *auth.Service
	cookieName  string
}

func NewHandler(service *Service, authService *auth.Service, cookieName string) *Handler {
	return &Handler{
		service:     service,
		authService: authService,
		cookieName:  cookieName,
	}
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

	response := fiber.Map{
		"items": items,
	}

	user, ok, err := h.currentUser(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if ok {
		stats, err := h.service.GetUserTaskStats(c.Context(), user.ID.String())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		response["userTaskStats"] = stats
	}

	return c.JSON(response)
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

	user, ok, err := h.currentUser(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if ok {
		history, err := h.service.ListUserTaskSubmissions(
			c.Context(),
			challenge.TaskID,
			user.ID.String(),
		)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		challenge.SubmissionHistory = history
	}

	return c.JSON(fiber.Map{
		"task": challenge,
	})
}

func (h *Handler) currentUser(c *fiber.Ctx) (auth.PublicUser, bool, error) {
	if h.authService == nil || strings.TrimSpace(h.cookieName) == "" {
		return auth.PublicUser{}, false, nil
	}

	token := c.Cookies(h.cookieName)
	if strings.TrimSpace(token) == "" {
		return auth.PublicUser{}, false, nil
	}

	user, err := h.authService.CurrentUser(c.UserContext(), token)
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			return auth.PublicUser{}, false, nil
		}

		return auth.PublicUser{}, false, err
	}

	return user, true, nil
}

