package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/judge"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestHandlerListChallenges(t *testing.T) {
	repo := &fakeTaskHandlerRepo{
		items: []ChallengeListItem{{
			ID:                 "payload-id",
			TaskID:             uuid.NewString(),
			Slug:               "two-sum",
			Title:              "Two Sum",
			SupportedLanguages: []string{"go", "rust"},
		}},
	}
	app := fiber.New()
	NewHandler(NewService(repo, fakeTaskHandlerJudge{}), nil, "").RegisterRoutes(app)

	resp := taskHandlerRequest(t, app, http.MethodGet, "/tasks", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body struct {
		Items []ChallengeListItem `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].SupportedLanguages[0] != "go" {
		t.Fatalf("body = %#v", body)
	}
}

func TestHandlerListChallengesForAuthenticatedUserIncludesStats(t *testing.T) {
	repo := &fakeTaskHandlerRepo{
		items: []ChallengeListItem{{
			ID:                 "payload-id",
			TaskID:             uuid.NewString(),
			Slug:               "two-sum",
			Title:              "Two Sum",
			SupportedLanguages: []string{"go"},
		}},
		stats: UserTaskStats{SolvedTasks: 1, InProgressTasks: 2},
	}
	authService := auth.NewService(newFakeTaskAuthRepo(), time.Hour, nil)
	result, err := authService.SignUp(context.Background(), auth.SignUpInput{
		Username: "player",
		Email:    "player@example.com",
		Password: "password123",
	}, auth.SessionMeta{})
	if err != nil {
		t.Fatalf("SignUp returned error: %v", err)
	}
	app := fiber.New()
	NewHandler(NewService(repo, fakeTaskHandlerJudge{}), authService, "sid").RegisterRoutes(app)

	resp := taskHandlerRequestWithCookie(t, app, http.MethodGet, "/tasks", "", "sid="+result.SessionToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body struct {
		UserTaskStats UserTaskStats `json:"userTaskStats"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.UserTaskStats.TotalTasks != 1 || body.UserTaskStats.SolvedTasks != 1 {
		t.Fatalf("stats = %#v", body.UserTaskStats)
	}
}

func TestHandlerGetChallengeBySlug(t *testing.T) {
	taskID := uuid.NewString()
	repo := &fakeTaskHandlerRepo{
		challenge: &Challenge{
			ID:                    "payload-id",
			TaskID:                taskID,
			Slug:                  "two-sum",
			Title:                 "Two Sum",
			SupportedLanguages:    []string{"go", "rust"},
			LanguageVersions:      map[string]string{"go": "1.24", "rust": "1.80"},
			StarterCodeByLanguage: map[string]string{"go": "package main", "rust": "fn main() {}"},
		},
	}
	app := fiber.New()
	NewHandler(NewService(repo, fakeTaskHandlerJudge{}), nil, "").RegisterRoutes(app)

	resp := taskHandlerRequest(t, app, http.MethodGet, "/tasks/two-sum", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body struct {
		Task Challenge `json:"task"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Task.TaskID != taskID || len(body.Task.SupportedLanguages) != 1 || body.Task.LanguageVersions["go"] != "1.24" {
		t.Fatalf("body = %#v", body)
	}
}

func TestHandlerGetChallengeBySlugForAuthenticatedUserIncludesHistory(t *testing.T) {
	taskID := uuid.NewString()
	authService := auth.NewService(newFakeTaskAuthRepo(), time.Hour, nil)
	result, err := authService.SignUp(context.Background(), auth.SignUpInput{
		Username: "player",
		Email:    "player@example.com",
		Password: "password123",
	}, auth.SessionMeta{})
	if err != nil {
		t.Fatalf("SignUp returned error: %v", err)
	}
	repo := &fakeTaskHandlerRepo{
		challenge: &Challenge{
			ID:                    "payload-id",
			TaskID:                taskID,
			Slug:                  "two-sum",
			Title:                 "Two Sum",
			SupportedLanguages:    []string{"go"},
			LanguageVersions:      map[string]string{"go": "1.24"},
			StarterCodeByLanguage: map[string]string{"go": "package main"},
		},
		history: []TaskSubmission{{
			ID:              uuid.NewString(),
			TaskID:          taskID,
			UserID:          result.User.ID.String(),
			Language:        "go",
			LanguageVersion: "1.24",
			Status:          "finished",
		}},
	}
	app := fiber.New()
	NewHandler(NewService(repo, fakeTaskHandlerJudge{}), authService, "sid").RegisterRoutes(app)

	resp := taskHandlerRequestWithCookie(t, app, http.MethodGet, "/tasks/two-sum", "", "sid="+result.SessionToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body struct {
		Task Challenge `json:"task"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Task.SubmissionHistory) != 1 {
		t.Fatalf("history = %#v", body.Task.SubmissionHistory)
	}
}

func TestHandlerGetChallengeByIDForAuthenticatedUserIncludesHistory(t *testing.T) {
	taskID := uuid.NewString()
	authService := auth.NewService(newFakeTaskAuthRepo(), time.Hour, nil)
	result, err := authService.SignUp(context.Background(), auth.SignUpInput{
		Username: "player",
		Email:    "player@example.com",
		Password: "password123",
	}, auth.SessionMeta{})
	if err != nil {
		t.Fatalf("SignUp returned error: %v", err)
	}
	repo := &fakeTaskHandlerRepo{
		challenge: &Challenge{
			ID:                    "payload-id",
			TaskID:                taskID,
			Slug:                  "two-sum",
			Title:                 "Two Sum",
			SupportedLanguages:    []string{"go"},
			LanguageVersions:      map[string]string{"go": "1.24"},
			StarterCodeByLanguage: map[string]string{"go": "package main"},
		},
		history: []TaskSubmission{{
			ID:              uuid.NewString(),
			TaskID:          taskID,
			UserID:          result.User.ID.String(),
			Language:        "go",
			LanguageVersion: "1.24",
			Status:          "finished",
		}},
	}
	app := fiber.New()
	NewHandler(NewService(repo, fakeTaskHandlerJudge{}), authService, "sid").RegisterRoutes(app)

	resp := taskHandlerRequestWithCookie(t, app, http.MethodGet, "/tasks/id/"+taskID, "", "sid="+result.SessionToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body struct {
		Task Challenge `json:"task"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Task.TaskID != taskID || len(body.Task.SubmissionHistory) != 1 {
		t.Fatalf("task = %#v", body.Task)
	}
}

func TestHandlerGetChallengeByIDRejectsInvalidID(t *testing.T) {
	app := fiber.New()
	NewHandler(NewService(&fakeTaskHandlerRepo{}, fakeTaskHandlerJudge{}), nil, "").RegisterRoutes(app)

	resp := taskHandlerRequest(t, app, http.MethodGet, "/tasks/id/not-a-uuid", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestHandlerGetChallengeBySlugReturnsNotFound(t *testing.T) {
	app := fiber.New()
	NewHandler(NewService(&fakeTaskHandlerRepo{}, fakeTaskHandlerJudge{}), nil, "").RegisterRoutes(app)

	resp := taskHandlerRequest(t, app, http.MethodGet, "/tasks/missing", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestHandlerServiceErrors(t *testing.T) {
	repoErr := errors.New("repo failed")
	taskID := uuid.NewString()

	tests := []struct {
		name string
		repo *fakeTaskHandlerRepo
		path string
	}{
		{name: "list", repo: &fakeTaskHandlerRepo{listErr: repoErr}, path: "/tasks"},
		{name: "detail by slug", repo: &fakeTaskHandlerRepo{challengeErr: repoErr}, path: "/tasks/two-sum"},
		{name: "detail by id", repo: &fakeTaskHandlerRepo{challengeErr: repoErr}, path: "/tasks/id/" + taskID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			NewHandler(NewService(tt.repo, fakeTaskHandlerJudge{}), nil, "").RegisterRoutes(app)

			resp := taskHandlerRequest(t, app, http.MethodGet, tt.path, "")
			if resp.StatusCode != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", resp.StatusCode)
			}
		})
	}
}

func TestHandlerAuthenticatedStatsAndHistoryErrors(t *testing.T) {
	taskID := uuid.NewString()
	authService := auth.NewService(newFakeTaskAuthRepo(), time.Hour, nil)
	result, err := authService.SignUp(context.Background(), auth.SignUpInput{
		Username: "player",
		Email:    "player@example.com",
		Password: "password123",
	}, auth.SessionMeta{})
	if err != nil {
		t.Fatalf("SignUp returned error: %v", err)
	}

	app := fiber.New()
	NewHandler(NewService(&fakeTaskHandlerRepo{
		items:    []ChallengeListItem{{TaskID: taskID}},
		statsErr: errors.New("stats failed"),
	}, fakeTaskHandlerJudge{}), authService, "sid").RegisterRoutes(app)
	resp := taskHandlerRequestWithCookie(t, app, http.MethodGet, "/tasks", "", "sid="+result.SessionToken)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("stats status = %d, want 500", resp.StatusCode)
	}

	challenge := &Challenge{TaskID: taskID, Slug: "two-sum", SupportedLanguages: []string{"go"}, LanguageVersions: map[string]string{"go": "1.24"}}
	app = fiber.New()
	NewHandler(NewService(&fakeTaskHandlerRepo{
		challenge:  challenge,
		historyErr: errors.New("history failed"),
	}, fakeTaskHandlerJudge{}), authService, "sid").RegisterRoutes(app)
	resp = taskHandlerRequestWithCookie(t, app, http.MethodGet, "/tasks/two-sum", "", "sid="+result.SessionToken)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("slug history status = %d, want 500", resp.StatusCode)
	}

	app = fiber.New()
	NewHandler(NewService(&fakeTaskHandlerRepo{
		challenge:  challenge,
		historyErr: errors.New("history failed"),
	}, fakeTaskHandlerJudge{}), authService, "sid").RegisterRoutes(app)
	resp = taskHandlerRequestWithCookie(t, app, http.MethodGet, "/tasks/id/"+taskID, "", "sid="+result.SessionToken)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("id history status = %d, want 500", resp.StatusCode)
	}
}

func taskHandlerRequest(t *testing.T, app *fiber.App, method, path, body string) *http.Response {
	t.Helper()
	return taskHandlerRequestWithCookie(t, app, method, path, body, "")
}

func taskHandlerRequestWithCookie(t *testing.T, app *fiber.App, method, path, body string, cookie string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	return resp
}

type fakeTaskHandlerRepo struct {
	items        []ChallengeListItem
	challenge    *Challenge
	stats        UserTaskStats
	history      []TaskSubmission
	listErr      error
	statsErr     error
	challengeErr error
	historyErr   error
}

func (r *fakeTaskHandlerRepo) ListChallenges(context.Context) ([]ChallengeListItem, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.items, nil
}

func (r *fakeTaskHandlerRepo) ListChallengesForUser(context.Context, string) ([]ChallengeListItem, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.items, nil
}

func (r *fakeTaskHandlerRepo) GetUserTaskStats(context.Context, string) (UserTaskStats, error) {
	if r.statsErr != nil {
		return UserTaskStats{}, r.statsErr
	}
	return r.stats, nil
}

func (r *fakeTaskHandlerRepo) ListUserTaskSubmissions(context.Context, string, string) ([]TaskSubmission, error) {
	if r.historyErr != nil {
		return nil, r.historyErr
	}
	return r.history, nil
}

func (r *fakeTaskHandlerRepo) GetChallengeBySlug(context.Context, string) (*Challenge, error) {
	if r.challengeErr != nil {
		return nil, r.challengeErr
	}
	return r.challenge, nil
}

func (r *fakeTaskHandlerRepo) GetChallengeByID(context.Context, string) (*Challenge, error) {
	if r.challengeErr != nil {
		return nil, r.challengeErr
	}
	return r.challenge, nil
}

type fakeTaskHandlerJudge struct{}

func (fakeTaskHandlerJudge) SupportedLanguages(context.Context) ([]judge.SupportedLanguage, error) {
	return []judge.SupportedLanguage{{Language: "go", Version: "1.24"}}, nil
}

type fakeTaskAuthRepo struct {
	usersByID      map[uuid.UUID]auth.User
	usersByEmail   map[string]auth.User
	usersByName    map[string]auth.User
	sessionsByHash map[string]auth.Session
}

func newFakeTaskAuthRepo() *fakeTaskAuthRepo {
	return &fakeTaskAuthRepo{
		usersByID:      make(map[uuid.UUID]auth.User),
		usersByEmail:   make(map[string]auth.User),
		usersByName:    make(map[string]auth.User),
		sessionsByHash: make(map[string]auth.Session),
	}
}

func (r *fakeTaskAuthRepo) CreateUser(_ context.Context, user auth.User) (auth.User, error) {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	r.usersByID[user.ID] = user
	r.usersByEmail[user.Email] = user
	r.usersByName[user.Username] = user
	return user, nil
}
func (r *fakeTaskAuthRepo) GetUserByEmail(_ context.Context, email string) (auth.User, error) {
	user, ok := r.usersByEmail[email]
	if !ok {
		return auth.User{}, auth.ErrNotFound
	}
	return user, nil
}
func (r *fakeTaskAuthRepo) GetUserByUsername(_ context.Context, username string) (auth.User, error) {
	user, ok := r.usersByName[username]
	if !ok {
		return auth.User{}, auth.ErrNotFound
	}
	return user, nil
}
func (r *fakeTaskAuthRepo) GetUserByID(_ context.Context, userID uuid.UUID) (auth.User, error) {
	user, ok := r.usersByID[userID]
	if !ok {
		return auth.User{}, auth.ErrNotFound
	}
	return user, nil
}
func (r *fakeTaskAuthRepo) GetOAuthAccount(context.Context, auth.OAuthProvider, string) (auth.OAuthAccount, error) {
	return auth.OAuthAccount{}, auth.ErrNotFound
}
func (r *fakeTaskAuthRepo) GetOAuthAccountByUserProvider(context.Context, uuid.UUID, auth.OAuthProvider) (auth.OAuthAccount, error) {
	return auth.OAuthAccount{}, auth.ErrNotFound
}
func (r *fakeTaskAuthRepo) CreateOAuthAccount(context.Context, auth.OAuthAccount) (auth.OAuthAccount, error) {
	return auth.OAuthAccount{}, auth.ErrInvalidInput
}
func (r *fakeTaskAuthRepo) UpdateProviderAvatar(context.Context, uuid.UUID, string) error { return nil }
func (r *fakeTaskAuthRepo) CreateSession(_ context.Context, session auth.Session) (auth.Session, error) {
	r.sessionsByHash[session.TokenHash] = session
	return session, nil
}
func (r *fakeTaskAuthRepo) GetSessionByTokenHash(_ context.Context, tokenHash string) (auth.Session, error) {
	session, ok := r.sessionsByHash[tokenHash]
	if !ok {
		return auth.Session{}, auth.ErrNotFound
	}
	return session, nil
}
func (r *fakeTaskAuthRepo) DeleteSessionByTokenHash(_ context.Context, tokenHash string) error {
	delete(r.sessionsByHash, tokenHash)
	return nil
}
func (r *fakeTaskAuthRepo) DeleteExpiredSessions(context.Context, time.Time) error  { return nil }
func (r *fakeTaskAuthRepo) DeleteSessionsByUserID(context.Context, uuid.UUID) error { return nil }
func (r *fakeTaskAuthRepo) CreatePasswordResetCode(context.Context, auth.PasswordResetCode) (auth.PasswordResetCode, error) {
	return auth.PasswordResetCode{}, auth.ErrInvalidInput
}
func (r *fakeTaskAuthRepo) GetActivePasswordResetCode(context.Context, uuid.UUID, time.Time) (auth.PasswordResetCode, error) {
	return auth.PasswordResetCode{}, auth.ErrNotFound
}
func (r *fakeTaskAuthRepo) IncrementPasswordResetAttempts(context.Context, uuid.UUID) error {
	return nil
}
func (r *fakeTaskAuthRepo) MarkPasswordResetCodeUsed(context.Context, uuid.UUID) error { return nil }
func (r *fakeTaskAuthRepo) UpdateUserPasswordHash(context.Context, uuid.UUID, string) error {
	return nil
}
