package users

import (
	"context"
	"errors"
	"io"
	stdhttp "net/http"
	"strings"

	"code-fight-backend/internal/auth"

	"github.com/gofiber/fiber/v2"
)

type currentUserReader interface {
	CurrentUser(ctx context.Context, sessionToken string) (auth.PublicUser, error)
}

type Handler struct {
	service           *Service
	authReader        currentUserReader
	sessionCookieName string
}

type profileResponse struct {
	Profile Profile `json:"profile"`
}

func NewHandler(service *Service, authReader currentUserReader, sessionCookieName string) *Handler {
	return &Handler{
		service:           service,
		authReader:        authReader,
		sessionCookieName: sessionCookieName,
	}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/:username", h.profileByUsername)
	router.Patch("/:username", h.updateProfile)
	router.Post("/:username/avatar", h.updateAvatar)
}

func (h *Handler) profileByUsername(c *fiber.Ctx) error {
	profile, err := h.service.GetProfileByUsername(c.UserContext(), c.Params("username"))
	if err != nil {
		return mapServiceError(err)
	}

	return c.JSON(profileResponse{Profile: profile})
}

func (h *Handler) updateProfile(c *fiber.Ctx) error {
	currentUser, err := h.authReader.CurrentUser(c.UserContext(), c.Cookies(h.sessionCookieName))
	if err != nil {
		return mapServiceError(err)
	}

	username := strings.TrimSpace(c.Params("username"))
	if !strings.EqualFold(currentUser.Username, username) {
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	}

	var input UpdateProfileInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	profile, err := h.service.UpdateProfile(c.UserContext(), currentUser.ID, username, input)
	if err != nil {
		return mapServiceError(err)
	}

	return c.JSON(profileResponse{Profile: profile})
}

func (h *Handler) updateAvatar(c *fiber.Ctx) error {
	currentUser, err := h.authReader.CurrentUser(c.UserContext(), c.Cookies(h.sessionCookieName))
	if err != nil {
		return mapServiceError(err)
	}

	username := strings.TrimSpace(c.Params("username"))
	if !strings.EqualFold(currentUser.Username, username) {
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	}

	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "avatar file is required")
	}
	if fileHeader.Size > maxAvatarFileSize {
		return fiber.NewError(fiber.StatusBadRequest, "avatar file must be 2 MB or smaller")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "avatar file is invalid")
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxAvatarFileSize+1))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "failed to read avatar file")
	}
	if len(content) > maxAvatarFileSize {
		return fiber.NewError(fiber.StatusBadRequest, "avatar file must be 2 MB or smaller")
	}

	contentType := stdhttp.DetectContentType(content)
	profile, err := h.service.UpdateAvatar(c.UserContext(), currentUser.ID, username, UpdateAvatarInput{
		Content:     content,
		ContentType: contentType,
	})
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
	case errors.Is(err, ErrForbidden):
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	case errors.Is(err, ErrUnavailable):
		return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
	case errors.Is(err, auth.ErrUnauthorized):
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "internal server error")
	}
}
