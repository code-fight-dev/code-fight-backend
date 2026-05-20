package leaderboard

import (
	"errors"
	"strconv"
	"strings"

	"code-fight-backend/internal/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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
	router.Get("/leaderboard", h.GetLeaderboard)
}

func (h *Handler) GetLeaderboard(c *fiber.Ctx) error {
	limitFallback := DefaultLimit
	if strings.TrimSpace(c.Query("pageSize")) != "" {
		limitFallback = 0
	}

	limit, err := parseIntQuery(c, "limit", limitFallback)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid limit")
	}
	if limit == 0 {
		limit, err = parseIntQuery(c, "pageSize", DefaultLimit)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid pageSize")
		}
	}

	offset, err := parseIntQuery(c, "offset", DefaultOffset)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid offset")
	}

	if strings.TrimSpace(c.Query("page")) != "" {
		page, err := parseIntQuery(c, "page", 1)
		if err != nil || page <= 0 {
			return fiber.NewError(fiber.StatusBadRequest, "invalid page")
		}
		offset = (page - 1) * limit
	}

	viewerID, err := h.viewerID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	page, err := h.service.GetPage(c.Context(), c.Query("mode", DefaultMode), limit, offset, viewerID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(page)
}

func (h *Handler) viewerID(c *fiber.Ctx) (*uuid.UUID, error) {
	if h.authService == nil || strings.TrimSpace(h.cookieName) == "" {
		return nil, nil
	}

	token := c.Cookies(h.cookieName)
	if strings.TrimSpace(token) == "" {
		return nil, nil
	}

	user, err := h.authService.CurrentUser(c.UserContext(), token)
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			return nil, nil
		}
		return nil, err
	}

	return &user.ID, nil
}

func parseIntQuery(c *fiber.Ctx, key string, fallback int) (int, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}

	return value, nil
}
