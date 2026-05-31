package submissions

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"code-fight-backend/internal/judge"
	"code-fight-backend/internal/matches"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestHandlerCreateTaskSubmission(t *testing.T) {
	taskID := uuid.New()
	userID := uuid.New()
	repo := submissionRepoFixture()
	service := NewService(
		repo,
		&fakeTaskRepo{task: &Task{ID: taskID, IsActive: true}, allowed: true},
		nil,
		&fakeJudgeClient{
			supported:      []judge.SupportedLanguage{{Language: "go", Version: "1.24"}},
			submitResponse: &judge.SubmitResponse{JudgeSubmissionID: "judge-1", Status: judge.StatusQueued},
		},
		nil,
		"callback",
		"token",
		1024,
		nil,
	)
	app := submissionsHandlerApp(service, "token", userID)

	resp := submissionsHandlerRequest(
		t,
		app,
		http.MethodPost,
		"/tasks/"+taskID.String()+"/submissions",
		`{"language":"go","language_version":"1.24","source_code":"package main"}`,
		nil,
	)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
}

func TestHandlerCreateMatchSubmission(t *testing.T) {
	taskID := uuid.New()
	matchID := uuid.New()
	userID := uuid.New()
	player2ID := uuid.New()
	repo := submissionRepoFixture()
	service := NewService(
		repo,
		&fakeTaskRepo{allowed: true},
		&fakeMatchRepo{match: matches.Match{
			ID:        matchID,
			TaskID:    &taskID,
			Player1ID: userID,
			Player2ID: player2ID,
			Status:    matches.MatchStatusRunning,
		}},
		&fakeJudgeClient{
			supported:      []judge.SupportedLanguage{{Language: "go", Version: "1.24"}},
			submitResponse: &judge.SubmitResponse{JudgeSubmissionID: "judge-match", Status: judge.StatusQueued},
		},
		nil,
		"callback",
		"token",
		1024,
		nil,
	)
	app := submissionsHandlerApp(service, "token", userID)

	resp := submissionsHandlerRequest(
		t,
		app,
		http.MethodPost,
		"/matches/"+matchID.String()+"/submissions",
		`{"language":"go","language_version":"1.24","source_code":"package main"}`,
		nil,
	)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
}

func TestHandlerCreateTaskCodeRun(t *testing.T) {
	taskID := uuid.New()
	userID := uuid.New()
	repo := submissionRepoFixture()
	service := NewService(
		repo,
		&fakeTaskRepo{task: &Task{ID: taskID, IsActive: true}, allowed: true},
		nil,
		&fakeJudgeClient{
			supported:      []judge.SupportedLanguage{{Language: "go", Version: "1.24"}},
			submitResponse: &judge.SubmitResponse{JudgeSubmissionID: "judge-run", Status: judge.StatusQueued},
		},
		nil,
		"callback",
		"token",
		1024,
		nil,
	)
	app := submissionsHandlerApp(service, "token", userID)

	resp := submissionsHandlerRequest(
		t,
		app,
		http.MethodPost,
		"/tasks/"+taskID.String()+"/runs",
		`{"language":"go","language_version":"1.24","source_code":"package main","custom_test":{"input":"1 2\n"}}`,
		nil,
	)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", resp.StatusCode)
	}
}

func TestHandlerCreateTaskSubmissionRequiresAuth(t *testing.T) {
	app := submissionsHandlerApp(NewService(nil, nil, nil, nil, nil, "", "", 1024, nil), "token", uuid.Nil)

	resp := submissionsHandlerRequest(
		t,
		app,
		http.MethodPost,
		"/tasks/"+uuid.NewString()+"/submissions",
		`{"language":"go","language_version":"1.24","source_code":"package main"}`,
		nil,
	)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestHandlerValidationBranches(t *testing.T) {
	userID := uuid.New()
	app := submissionsHandlerApp(NewService(nil, nil, nil, nil, nil, "", "", 1024, nil), "token", userID)
	matchID := uuid.NewString()
	taskID := uuid.NewString()

	tests := []struct {
		method string
		path   string
		body   string
		want   int
	}{
		{http.MethodPost, "/tasks/not-uuid/submissions", `{}`, http.StatusBadRequest},
		{http.MethodPost, "/tasks/" + taskID + "/submissions", `{`, http.StatusBadRequest},
		{http.MethodPost, "/matches/not-uuid/submissions", `{}`, http.StatusBadRequest},
		{http.MethodPost, "/matches/" + matchID + "/submissions", `{`, http.StatusBadRequest},
		{http.MethodPost, "/tasks/not-uuid/runs", `{}`, http.StatusBadRequest},
		{http.MethodPost, "/tasks/" + taskID + "/runs", `{`, http.StatusBadRequest},
		{http.MethodPost, "/judge/callback", `{`, http.StatusUnauthorized},
		{http.MethodPost, "/judge/callback", `{`, http.StatusBadRequest},
		{http.MethodPost, "/judge/callback", `{"judge_status":"running"}`, http.StatusBadRequest},
		{http.MethodPost, "/judge/callback", `{"judge_submission_id":"judge-1"}`, http.StatusBadRequest},
	}
	for i, tt := range tests {
		headers := map[string]string(nil)
		if i >= 7 {
			headers = map[string]string{"X-Judge-Token": "token"}
		}
		resp := submissionsHandlerRequest(t, app, tt.method, tt.path, tt.body, headers)
		if resp.StatusCode != tt.want {
			t.Fatalf("%s %s status = %d, want %d", tt.method, tt.path, resp.StatusCode, tt.want)
		}
	}
}

func TestHandlerGetSubmission(t *testing.T) {
	userID := uuid.New()
	subID := uuid.NewString()
	repo := submissionRepoFixture()
	repo.submissions = map[string]*Submission{}
	repo.submissions[subID] = &Submission{
		ID:              subID,
		TaskID:          uuid.NewString(),
		UserID:          userID.String(),
		Language:        "go",
		LanguageVersion: "1.24",
		Status:          StatusFinished,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	app := submissionsHandlerApp(NewService(repo, nil, nil, nil, nil, "", "", 1024, nil), "token", userID)

	resp := submissionsHandlerRequest(t, app, http.MethodGet, "/submissions/"+subID, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestHandlerGetCodeRun(t *testing.T) {
	userID := uuid.New()
	runID := uuid.NewString()
	repo := submissionRepoFixture()
	repo.runs = map[string]*CodeRun{}
	repo.runs[runID] = &CodeRun{
		ID:              runID,
		TaskID:          uuid.NewString(),
		UserID:          userID.String(),
		Language:        "go",
		LanguageVersion: "1.24",
		Status:          StatusFinished,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	app := submissionsHandlerApp(NewService(repo, nil, nil, nil, nil, "", "", 1024, nil), "token", userID)

	resp := submissionsHandlerRequest(t, app, http.MethodGet, "/code-runs/"+runID, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestHandlerReadErrors(t *testing.T) {
	userID := uuid.New()
	app := submissionsHandlerApp(NewService(submissionRepoFixture(), nil, nil, nil, nil, "", "", 1024, nil), "token", userID)

	subResp := submissionsHandlerRequest(t, app, http.MethodGet, "/submissions/missing", "", nil)
	if subResp.StatusCode != http.StatusNotFound {
		t.Fatalf("submission status = %d, want 404", subResp.StatusCode)
	}

	runResp := submissionsHandlerRequest(t, app, http.MethodGet, "/code-runs/missing", "", nil)
	if runResp.StatusCode != http.StatusNotFound {
		t.Fatalf("code run status = %d, want 404", runResp.StatusCode)
	}

	unauthApp := submissionsHandlerApp(NewService(nil, nil, nil, nil, nil, "", "", 1024, nil), "token", uuid.Nil)
	for _, path := range []string{"/submissions/missing", "/code-runs/missing"} {
		resp := submissionsHandlerRequest(t, unauthApp, http.MethodGet, path, "", nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s status = %d, want 401", path, resp.StatusCode)
		}
	}
}

func TestHandlerMapsServiceErrors(t *testing.T) {
	app := fiber.New()
	app.Get("/task", func(c *fiber.Ctx) error { return mapServiceError(ErrTaskNotFound) })
	app.Get("/bad", func(c *fiber.Ctx) error { return mapServiceError(ErrInvalidSourceCode) })

	notFound := submissionsHandlerRequest(t, app, http.MethodGet, "/task", "", nil)
	if notFound.StatusCode != http.StatusNotFound {
		t.Fatalf("not found status = %d, want 404", notFound.StatusCode)
	}
	bad := submissionsHandlerRequest(t, app, http.MethodGet, "/bad", "", nil)
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad status = %d, want 400", bad.StatusCode)
	}
}

func TestHandlerJudgeCallback(t *testing.T) {
	subID := uuid.NewString()
	judgeID := "judge-submission"
	repo := submissionRepoFixture()
	repo.submissions = map[string]*Submission{}
	repo.submissions[subID] = &Submission{
		ID:                subID,
		TaskID:            uuid.NewString(),
		UserID:            uuid.NewString(),
		Language:          "go",
		LanguageVersion:   "1.24",
		Status:            StatusSentToJudge,
		JudgeSubmissionID: &judgeID,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	app := submissionsHandlerApp(NewService(repo, nil, nil, nil, nil, "", "", 1024, nil), "token", uuid.Nil)

	resp := submissionsHandlerRequest(
		t,
		app,
		http.MethodPost,
		"/judge/callback",
		`{"judge_submission_id":"judge-submission","submission_id":"`+subID+`","judge_status":"running"}`,
		map[string]string{"X-Judge-Token": "token"},
	)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestHandlerJudgeCallbackRejectsBadToken(t *testing.T) {
	app := submissionsHandlerApp(NewService(nil, nil, nil, nil, nil, "", "", 1024, nil), "token", uuid.Nil)

	resp := submissionsHandlerRequest(
		t,
		app,
		http.MethodPost,
		"/judge/callback",
		`{"judge_submission_id":"judge-submission","judge_status":"running"}`,
		map[string]string{"X-Judge-Token": "wrong"},
	)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestMapServiceErrorStatuses(t *testing.T) {
	app := fiber.New()
	app.Get("/not-found", func(c *fiber.Ctx) error { return mapServiceError(ErrSubmissionNotFound) })
	app.Get("/task-not-found", func(c *fiber.Ctx) error { return mapServiceError(ErrTaskNotFound) })
	app.Get("/match-not-found", func(c *fiber.Ctx) error { return mapServiceError(ErrMatchNotFound) })
	app.Get("/bad", func(c *fiber.Ctx) error { return mapServiceError(ErrInvalidSourceCode) })
	app.Get("/inactive", func(c *fiber.Ctx) error { return mapServiceError(ErrTaskInactive) })
	app.Get("/language", func(c *fiber.Ctx) error { return mapServiceError(ErrTaskLanguageNotAllowed) })
	app.Get("/match-status", func(c *fiber.Ctx) error { return mapServiceError(ErrMatchNotStarted) })
	app.Get("/match-task", func(c *fiber.Ctx) error { return mapServiceError(ErrMatchTaskNotAssigned) })
	app.Get("/participant", func(c *fiber.Ctx) error { return mapServiceError(ErrUserNotMatchParticipant) })
	app.Get("/unauthorized", func(c *fiber.Ctx) error { return mapServiceError(ErrJudgeCallbackUnauthorized) })
	app.Get("/judge", func(c *fiber.Ctx) error { return mapServiceError(ErrJudgeSubmitFailed) })
	app.Get("/unknown", func(c *fiber.Ctx) error { return mapServiceError(errors.New("boom")) })

	tests := map[string]int{
		"/not-found":       http.StatusNotFound,
		"/task-not-found":  http.StatusNotFound,
		"/match-not-found": http.StatusNotFound,
		"/bad":             http.StatusBadRequest,
		"/inactive":        http.StatusBadRequest,
		"/language":        http.StatusBadRequest,
		"/match-status":    http.StatusBadRequest,
		"/match-task":      http.StatusBadRequest,
		"/participant":     http.StatusBadRequest,
		"/unauthorized":    http.StatusUnauthorized,
		"/judge":           http.StatusInternalServerError,
		"/unknown":         http.StatusInternalServerError,
	}
	for path, want := range tests {
		resp := submissionsHandlerRequest(t, app, http.MethodGet, path, "", nil)
		if resp.StatusCode != want {
			t.Fatalf("%s status = %d, want %d", path, resp.StatusCode, want)
		}
	}
}

func TestUserIDFromCtxBranches(t *testing.T) {
	userID := uuid.New()
	app := fiber.New()
	app.Get("/string", func(c *fiber.Ctx) error {
		c.Locals("user_id", userID.String())
		got, err := userIDFromCtx(c)
		if err != nil {
			return err
		}
		return c.SendString(got.String())
	})
	app.Get("/bad", func(c *fiber.Ctx) error {
		c.Locals("user_id", "bad")
		if _, err := userIDFromCtx(c); err != nil {
			return c.SendStatus(http.StatusBadRequest)
		}
		return c.SendStatus(http.StatusOK)
	})
	app.Get("/nil", func(c *fiber.Ctx) error {
		c.Locals("user_id", uuid.Nil)
		if _, err := userIDFromCtx(c); err != nil {
			return c.SendStatus(http.StatusUnauthorized)
		}
		return c.SendStatus(http.StatusOK)
	})

	tests := map[string]int{
		"/string": http.StatusOK,
		"/bad":    http.StatusBadRequest,
		"/nil":    http.StatusUnauthorized,
	}
	for path, want := range tests {
		resp := submissionsHandlerRequest(t, app, http.MethodGet, path, "", nil)
		if resp.StatusCode != want {
			t.Fatalf("%s status = %d, want %d", path, resp.StatusCode, want)
		}
	}
}

func submissionsHandlerApp(service *Service, token string, userID uuid.UUID) *fiber.App {
	app := fiber.New()
	if userID != uuid.Nil {
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("user_id", userID)
			return c.Next()
		})
	}
	NewHandler(service, token).RegisterRoutes(app, app)
	return app
}

func submissionsHandlerRequest(t *testing.T, app *fiber.App, method, path, body string, headers map[string]string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	return resp
}
