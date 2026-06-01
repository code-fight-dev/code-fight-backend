package leaderboard

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"code-fight-backend/internal/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestHandlerGetLeaderboard(t *testing.T) {
	repo := &fakeLeaderboardHandlerRepo{
		items: []Entry{{Rank: 1, UserID: uuid.New(), Username: "ada", Rating: 1800}},
		total: 1,
		stats: Stats{RankedPlayers: 1},
	}
	app := fiber.New()
	NewHandler(NewService(repo), nil, "").RegisterRoutes(app)

	resp := leaderboardHandlerRequest(t, app, http.MethodGet, "/leaderboard?page=2&pageSize=25&mode=regional")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if repo.mode != "regional" || repo.limit != 25 || repo.offset != 25 {
		t.Fatalf("query mapping = mode %q, limit %d, offset %d", repo.mode, repo.limit, repo.offset)
	}

	var body Page
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Mode != "regional" || len(body.Items) != 1 || body.Stats.RankedPlayers != 1 {
		t.Fatalf("body = %#v", body)
	}
}

func TestHandlerGetLeaderboardRejectsInvalidQueries(t *testing.T) {
	app := fiber.New()
	NewHandler(NewService(&fakeLeaderboardHandlerRepo{}), nil, "").RegisterRoutes(app)

	tests := []string{
		"/leaderboard?limit=nope",
		"/leaderboard?pageSize=nope",
		"/leaderboard?offset=nope",
		"/leaderboard?page=0",
	}
	for _, path := range tests {
		resp := leaderboardHandlerRequest(t, app, http.MethodGet, path)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want 400", path, resp.StatusCode)
		}
	}
}

func TestHandlerGetLeaderboardIncludesViewerRankForAuthenticatedUser(t *testing.T) {
	authRepo := newFakeLeaderboardAuthRepo()
	authService := auth.NewService(authRepo, time.Hour, nil)
	result, err := authService.SignUp(context.Background(), auth.SignUpInput{
		Username: "player",
		Email:    "player@example.com",
		Password: "password123",
	}, auth.SessionMeta{})
	if err != nil {
		t.Fatalf("SignUp returned error: %v", err)
	}

	viewerRank := &Entry{Rank: 7, UserID: result.User.ID, Username: result.User.Username, Rating: 1510}
	repo := &fakeLeaderboardHandlerRepo{
		items:      []Entry{{Rank: 1, UserID: uuid.New(), Username: "ada", Rating: 1800}},
		total:      10,
		stats:      Stats{RankedPlayers: 10},
		viewerRank: viewerRank,
	}
	app := fiber.New()
	NewHandler(NewService(repo), authService, "sid").RegisterRoutes(app)

	resp := leaderboardHandlerRequestWithCookie(t, app, http.MethodGet, "/leaderboard?mode=global", "sid="+result.SessionToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if repo.viewerID != result.User.ID {
		t.Fatalf("viewerID = %s, want %s", repo.viewerID, result.User.ID)
	}

	var body Page
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ViewerRank == nil || body.ViewerRank.Rank != viewerRank.Rank || body.ViewerRank.UserID != result.User.ID {
		t.Fatalf("viewerRank = %#v", body.ViewerRank)
	}
}

func TestHandlerGetLeaderboardTreatsUnauthorizedViewerAsAnonymous(t *testing.T) {
	authRepo := newFakeLeaderboardAuthRepo()
	authService := auth.NewService(authRepo, time.Hour, nil)
	repo := &fakeLeaderboardHandlerRepo{stats: Stats{}}
	app := fiber.New()
	NewHandler(NewService(repo), authService, "sid").RegisterRoutes(app)

	resp := leaderboardHandlerRequestWithCookie(t, app, http.MethodGet, "/leaderboard", "sid=missing")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if repo.viewerRankCalls != 0 {
		t.Fatalf("viewer rank calls = %d, want 0", repo.viewerRankCalls)
	}
}

func TestHandlerGetLeaderboardReturnsAuthErrors(t *testing.T) {
	authRepo := newFakeLeaderboardAuthRepo()
	authRepo.getSessionErr = errors.New("session store down")
	authService := auth.NewService(authRepo, time.Hour, nil)
	app := fiber.New()
	NewHandler(NewService(&fakeLeaderboardHandlerRepo{}), authService, "sid").RegisterRoutes(app)

	resp := leaderboardHandlerRequestWithCookie(t, app, http.MethodGet, "/leaderboard", "sid=token")
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}

func leaderboardHandlerRequest(t *testing.T, app *fiber.App, method, path string) *http.Response {
	t.Helper()
	return leaderboardHandlerRequestWithCookie(t, app, method, path, "")
}

func leaderboardHandlerRequestWithCookie(t *testing.T, app *fiber.App, method, path string, cookie string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, path, strings.NewReader(""))
	if err != nil {
		t.Fatalf("new request: %v", err)
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

type fakeLeaderboardHandlerRepo struct {
	items  []Entry
	total  int
	stats  Stats
	mode   string
	limit  int
	offset int

	viewerID        uuid.UUID
	viewerRank      *Entry
	viewerRankCalls int
}

func (r *fakeLeaderboardHandlerRepo) List(_ context.Context, mode string, limit int, offset int) ([]Entry, int, error) {
	r.mode = mode
	r.limit = limit
	r.offset = offset
	return r.items, r.total, nil
}

func (r *fakeLeaderboardHandlerRepo) GetViewerRank(_ context.Context, _ string, userID uuid.UUID) (*Entry, error) {
	r.viewerID = userID
	r.viewerRankCalls++
	return r.viewerRank, nil
}

func (r *fakeLeaderboardHandlerRepo) GetStats(context.Context, string) (Stats, error) {
	return r.stats, nil
}

type fakeLeaderboardAuthRepo struct {
	usersByID      map[uuid.UUID]auth.User
	usersByEmail   map[string]auth.User
	usersByName    map[string]auth.User
	sessionsByHash map[string]auth.Session
	getSessionErr  error
}

func newFakeLeaderboardAuthRepo() *fakeLeaderboardAuthRepo {
	return &fakeLeaderboardAuthRepo{
		usersByID:      make(map[uuid.UUID]auth.User),
		usersByEmail:   make(map[string]auth.User),
		usersByName:    make(map[string]auth.User),
		sessionsByHash: make(map[string]auth.Session),
	}
}

func (r *fakeLeaderboardAuthRepo) CreateUser(_ context.Context, user auth.User) (auth.User, error) {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	r.usersByID[user.ID] = user
	r.usersByEmail[user.Email] = user
	r.usersByName[user.Username] = user
	return user, nil
}

func (r *fakeLeaderboardAuthRepo) GetUserByEmail(_ context.Context, email string) (auth.User, error) {
	user, ok := r.usersByEmail[email]
	if !ok {
		return auth.User{}, auth.ErrNotFound
	}
	return user, nil
}

func (r *fakeLeaderboardAuthRepo) GetUserByUsername(_ context.Context, username string) (auth.User, error) {
	user, ok := r.usersByName[username]
	if !ok {
		return auth.User{}, auth.ErrNotFound
	}
	return user, nil
}

func (r *fakeLeaderboardAuthRepo) GetUserByID(_ context.Context, userID uuid.UUID) (auth.User, error) {
	user, ok := r.usersByID[userID]
	if !ok {
		return auth.User{}, auth.ErrNotFound
	}
	return user, nil
}

func (r *fakeLeaderboardAuthRepo) GetOAuthAccount(context.Context, auth.OAuthProvider, string) (auth.OAuthAccount, error) {
	return auth.OAuthAccount{}, auth.ErrNotFound
}

func (r *fakeLeaderboardAuthRepo) GetOAuthAccountByUserProvider(context.Context, uuid.UUID, auth.OAuthProvider) (auth.OAuthAccount, error) {
	return auth.OAuthAccount{}, auth.ErrNotFound
}

func (r *fakeLeaderboardAuthRepo) CreateOAuthAccount(context.Context, auth.OAuthAccount) (auth.OAuthAccount, error) {
	return auth.OAuthAccount{}, auth.ErrInvalidInput
}

func (r *fakeLeaderboardAuthRepo) UpdateProviderAvatar(context.Context, uuid.UUID, string) error {
	return nil
}

func (r *fakeLeaderboardAuthRepo) CreateSession(_ context.Context, session auth.Session) (auth.Session, error) {
	r.sessionsByHash[session.TokenHash] = session
	return session, nil
}

func (r *fakeLeaderboardAuthRepo) GetSessionByTokenHash(_ context.Context, tokenHash string) (auth.Session, error) {
	if r.getSessionErr != nil {
		return auth.Session{}, r.getSessionErr
	}
	session, ok := r.sessionsByHash[tokenHash]
	if !ok {
		return auth.Session{}, auth.ErrNotFound
	}
	return session, nil
}

func (r *fakeLeaderboardAuthRepo) DeleteSessionByTokenHash(_ context.Context, tokenHash string) error {
	delete(r.sessionsByHash, tokenHash)
	return nil
}

func (r *fakeLeaderboardAuthRepo) DeleteExpiredSessions(context.Context, time.Time) error {
	return nil
}

func (r *fakeLeaderboardAuthRepo) DeleteSessionsByUserID(context.Context, uuid.UUID) error {
	return nil
}

func (r *fakeLeaderboardAuthRepo) CreatePasswordResetCode(context.Context, auth.PasswordResetCode) (auth.PasswordResetCode, error) {
	return auth.PasswordResetCode{}, auth.ErrInvalidInput
}

func (r *fakeLeaderboardAuthRepo) GetActivePasswordResetCode(context.Context, uuid.UUID, time.Time) (auth.PasswordResetCode, error) {
	return auth.PasswordResetCode{}, auth.ErrNotFound
}

func (r *fakeLeaderboardAuthRepo) IncrementPasswordResetAttempts(context.Context, uuid.UUID) error {
	return nil
}

func (r *fakeLeaderboardAuthRepo) MarkPasswordResetCodeUsed(context.Context, uuid.UUID) error {
	return nil
}

func (r *fakeLeaderboardAuthRepo) UpdateUserPasswordHash(context.Context, uuid.UUID, string) error {
	return nil
}
