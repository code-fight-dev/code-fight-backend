package matches

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Post("/", h.CreateMatch)
	router.Post("/:id/start", h.StartMatch)
	router.Post("/:id/finish", h.FinishMatch)
	router.Post("/:id/cancel", h.CancelMatch)
	router.Get("/:id", h.GetMatch)
	router.Get("/user/:id", h.ListUserMatches)
}

type createMatchRequest struct {
	Player1ID string `json:"player1_id"`
	Player2ID string `json:"player2_id"`
	Mode      string `json:"mode"`
	IsRated   bool   `json:"is_rated"`
}

func (h *Handler) CreateMatch(c *fiber.Ctx) error {
	var req createMatchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	player1ID, err := uuid.Parse(req.Player1ID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid player1_id"})
	}
	player2ID, err := uuid.Parse(req.Player2ID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid player2_id"})
	}

	match, err := h.service.CreateMatch(c.UserContext(), CreateMatchInput{
		Player1ID: player1ID,
		Player2ID: player2ID,
		Mode:      req.Mode,
		IsRated:   req.IsRated,
	})
	if err != nil {
		return writeMatchError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(match)
}

func (h *Handler) StartMatch(c *fiber.Ctx) error {
	matchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid match id"})
	}

	match, err := h.service.StartMatch(c.UserContext(), StartMatchInput{
		MatchID: matchID,
	})
	if err != nil {
		return writeMatchError(c, err)
	}

	return c.JSON(match)
}

type finishMatchRequest struct {
	WinnerID *string `json:"winner_id"`
}

func (h *Handler) FinishMatch(c *fiber.Ctx) error {
	matchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid match id"})
	}

	var req finishMatchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	var winnerID *uuid.UUID
	if req.WinnerID != nil && *req.WinnerID != "" {
		parsed, err := uuid.Parse(*req.WinnerID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid winner_id"})
		}
		winnerID = &parsed
	}

	err = h.service.FinishMatch(c.UserContext(), FinishMatchInput{
		MatchID:  matchID,
		WinnerID: winnerID,
	})
	if err != nil {
		return writeMatchError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) CancelMatch(c *fiber.Ctx) error {
	matchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid match id"})
	}

	err = h.service.CancelMatch(c.UserContext(), CancelMatchInput{
		MatchID: matchID,
	})
	if err != nil {
		return writeMatchError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) GetMatch(c *fiber.Ctx) error {
	matchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid match id"})
	}

	match, err := h.service.GetMatch(c.UserContext(), matchID)
	if err != nil {
		return writeMatchError(c, err)
	}

	return c.JSON(match)
}

func (h *Handler) ListUserMatches(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	limit := 20
	offset := 0

	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid limit"})
		}
		limit = v
	}
	if raw := c.Query("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid offset"})
		}
		offset = v
	}

	items, err := h.service.ListUserMatches(c.UserContext(), ListUserMatchesInput{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return writeMatchError(c, err)
	}

	return c.JSON(fiber.Map{
		"items": items,
		"limit": limit,
		"offset": offset,
	})
}

func writeMatchError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrMatchNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrSamePlayer),
		errors.Is(err, ErrInvalidWinner),
		errors.Is(err, ErrInvalidPagination):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrMatchAlreadyStarted),
		errors.Is(err, ErrMatchAlreadyFinished),
		errors.Is(err, ErrMatchCancelled),
		errors.Is(err, ErrRatingAlreadyApplied),
		errors.Is(err, ErrInvalidMatchStatus):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrRatingNotFound):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
}