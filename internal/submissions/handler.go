package submissions

import (
	"crypto/subtle"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"code-fight-backend/internal/judge"
)

type Handler struct {
	service       *Service
	callbackToken string
}

func NewHandler(service *Service, callbackToken string) *Handler {
	return &Handler{
		service:       service,
		callbackToken: callbackToken,
	}
}

func (h *Handler) RegisterRoutes(api fiber.Router, internal fiber.Router) {
	api.Post("/tasks/:taskId/submissions", h.CreateTaskSubmission)
	api.Post("/tasks/:taskId/runs", h.CreateTaskCodeRun)
	api.Post("/matches/:matchId/submissions", h.CreateMatchSubmission)
	api.Get("/submissions/:submissionId", h.GetSubmission)
	api.Get("/code-runs/:codeRunId", h.GetCodeRun)

	internal.Post("/judge/callback", h.JudgeCallback)
}

func (h *Handler) CreateTaskSubmission(c *fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	taskID, err := uuid.Parse(c.Params("taskId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid task id")
	}

	var req CreateSubmissionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	sub, err := h.service.CreateTaskSubmission(c.Context(), userID, taskID, req)
	if err != nil {
		return mapServiceError(err)
	}

	full, tests, err := h.service.GetByID(c.Context(), sub.ID)
	if err != nil {
		return mapServiceError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(ToSubmissionResponse(full, tests))
}

func (h *Handler) CreateMatchSubmission(c *fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	matchID, err := uuid.Parse(c.Params("matchId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid match id")
	}

	var req CreateSubmissionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	sub, err := h.service.CreateMatchSubmission(c.Context(), userID, matchID, req)
	if err != nil {
		return mapServiceError(err)
	}

	full, tests, err := h.service.GetByID(c.Context(), sub.ID)
	if err != nil {
		return mapServiceError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(ToSubmissionResponse(full, tests))
}

func (h *Handler) CreateTaskCodeRun(c *fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	taskID, err := uuid.Parse(c.Params("taskId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid task id")
	}

	var req CreateCodeRunRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	run, err := h.service.CreateTaskCodeRun(c.Context(), userID, taskID, req)
	if err != nil {
		return mapServiceError(err)
	}

	full, tests, err := h.service.GetCodeRunByIDForUser(c.Context(), run.ID, userID)
	if err != nil {
		return mapServiceError(err)
	}

	return c.Status(fiber.StatusAccepted).JSON(ToCodeRunResponse(full, tests))
}

func (h *Handler) GetSubmission(c *fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	sub, tests, err := h.service.GetByIDForUser(c.Context(), c.Params("submissionId"), userID)
	if err != nil {
		return mapServiceError(err)
	}

	return c.JSON(ToSubmissionResponse(sub, tests))
}

func (h *Handler) GetCodeRun(c *fiber.Ctx) error {
	userID, err := userIDFromCtx(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	run, tests, err := h.service.GetCodeRunByIDForUser(c.Context(), c.Params("codeRunId"), userID)
	if err != nil {
		return mapServiceError(err)
	}

	return c.JSON(ToCodeRunResponse(run, tests))
}

func (h *Handler) JudgeCallback(c *fiber.Ctx) error {
	expected := strings.TrimSpace(h.callbackToken)
	received := strings.TrimSpace(c.Get("X-Judge-Token"))
	if expected == "" || received == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(received)) != 1 {
		return fiber.NewError(fiber.StatusUnauthorized, ErrJudgeCallbackUnauthorized.Error())
	}

	var payload judge.CallbackPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid callback body")
	}

	if payload.JudgeSubmissionID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "judge_submission_id is required")
	}
	if payload.JudgeStatus == "" {
		return fiber.NewError(fiber.StatusBadRequest, "judge_status is required")
	}

	if err := h.service.HandleJudgeCallback(c.Context(), payload); err != nil {
		return mapServiceError(err)
	}

	return c.SendStatus(fiber.StatusOK)
}

func mapServiceError(err error) error {
	switch {
	case errors.Is(err, ErrSubmissionNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, ErrTaskNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, ErrMatchNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, ErrTaskInactive),
		errors.Is(err, ErrTaskLanguageNotAllowed),
		errors.Is(err, ErrMatchNotStarted),
		errors.Is(err, ErrMatchTaskNotAssigned),
		errors.Is(err, ErrUserNotMatchParticipant),
		errors.Is(err, ErrInvalidSourceCode):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, ErrJudgeCallbackUnauthorized):
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
}

func userIDFromCtx(c *fiber.Ctx) (uuid.UUID, error) {
	v := c.Locals("user_id")

	switch id := v.(type) {
	case uuid.UUID:
		if id == uuid.Nil {
			return uuid.Nil, errors.New("missing user_id")
		}
		return id, nil

	case string:
		parsed, err := uuid.Parse(id)
		if err != nil {
			return uuid.Nil, err
		}
		if parsed == uuid.Nil {
			return uuid.Nil, errors.New("missing user_id")
		}
		return parsed, nil

	default:
		return uuid.Nil, errors.New("missing user_id")
	}
}
