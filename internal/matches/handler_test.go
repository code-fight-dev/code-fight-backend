package matches

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

var errUnknownMatchTest = errors.New("unknown")

func TestHandlerJoinQueue(t *testing.T) {
	userID := uuid.New()
	opponentID := uuid.New()
	taskID := uuid.New()
	repo := &fakeMatchServiceRepo{opponentID: &opponentID}
	app := matchesHandlerApp(NewService(repo, fakeTaskSelector{taskID: taskID}, nil), userID)

	resp := matchesHandlerRequest(t, app, http.MethodPost, "/matches/queue", `{"task_mode":"hard","is_rated":false}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body QueueResult
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "matched" || body.Match == nil {
		t.Fatalf("body = %#v", body)
	}
}

func TestHandlerCreateMatch(t *testing.T) {
	userID := uuid.New()
	player2ID := uuid.New()
	taskID := uuid.New()
	repo := &fakeMatchServiceRepo{}
	app := matchesHandlerApp(NewService(repo, fakeTaskSelector{taskID: taskID}, nil), userID)

	resp := matchesHandlerRequest(
		t,
		app,
		http.MethodPost,
		"/matches/",
		`{"player2_id":"`+player2ID.String()+`","task_mode":"normal","is_rated":true}`,
	)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	if repo.match.Player1ID != userID || repo.match.Player2ID != player2ID {
		t.Fatalf("created match = %#v", repo.match)
	}
}

func TestHandlerStartCancelAndSurrenderMatch(t *testing.T) {
	userID := uuid.New()
	matchID := uuid.New()
	taskID := uuid.New()
	difficulty := TaskDifficultyEasy
	repo := &fakeMatchServiceRepo{match: Match{
		ID:                     matchID,
		TaskID:                 &taskID,
		TaskDifficultySnapshot: &difficulty,
		Player1ID:              userID,
		Player2ID:              uuid.New(),
		Status:                 MatchStatusPending,
	}}
	app := matchesHandlerApp(NewService(repo, nil, nil), userID)

	startResp := matchesHandlerRequest(t, app, http.MethodPost, "/matches/"+matchID.String()+"/start", "")
	if startResp.StatusCode != http.StatusOK {
		t.Fatalf("start status = %d, want 200", startResp.StatusCode)
	}

	cancelResp := matchesHandlerRequest(t, app, http.MethodPost, "/matches/"+matchID.String()+"/cancel", "")
	if cancelResp.StatusCode != http.StatusNoContent {
		t.Fatalf("cancel status = %d, want 204", cancelResp.StatusCode)
	}

	repo.match.Status = MatchStatusRunning
	surrenderResp := matchesHandlerRequest(t, app, http.MethodPost, "/matches/"+matchID.String()+"/surrender", "")
	if surrenderResp.StatusCode != http.StatusNoContent {
		t.Fatalf("surrender status = %d, want 204", surrenderResp.StatusCode)
	}
}

func TestHandlerLeaveQueue(t *testing.T) {
	userID := uuid.New()
	repo := &fakeMatchServiceRepo{}
	app := matchesHandlerApp(NewService(repo, nil, nil), userID)

	resp := matchesHandlerRequest(t, app, http.MethodDelete, "/matches/queue", "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if len(repo.removedQueued) != 1 || repo.removedQueued[0] != userID {
		t.Fatalf("removed queued = %#v", repo.removedQueued)
	}
}

func TestHandlerGetMatchAndCurrent(t *testing.T) {
	userID := uuid.New()
	matchID := uuid.New()
	repo := &fakeMatchServiceRepo{
		match: Match{
			ID:        matchID,
			Player1ID: userID,
			Player2ID: uuid.New(),
			Status:    MatchStatusRunning,
		},
		current: Match{
			ID:        matchID,
			Player1ID: userID,
			Player2ID: uuid.New(),
			Status:    MatchStatusRunning,
		},
	}
	app := matchesHandlerApp(NewService(repo, nil, nil), userID)

	getResp := matchesHandlerRequest(t, app, http.MethodGet, "/matches/"+matchID.String(), "")
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200", getResp.StatusCode)
	}

	currentResp := matchesHandlerRequest(t, app, http.MethodGet, "/matches/current", "")
	if currentResp.StatusCode != http.StatusOK {
		t.Fatalf("current status = %d, want 200", currentResp.StatusCode)
	}
}

func TestHandlerListMatchSubmissions(t *testing.T) {
	userID := uuid.New()
	matchID := uuid.New()
	repo := &fakeMatchServiceRepo{
		match: Match{
			ID:        matchID,
			Player1ID: userID,
			Player2ID: uuid.New(),
			Status:    MatchStatusRunning,
		},
		submissions: []MatchSubmission{{
			ID:        uuid.New(),
			MatchID:   matchID,
			UserID:    userID,
			Status:    "finished",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}},
	}
	app := matchesHandlerApp(NewService(repo, nil, nil), userID)

	resp := matchesHandlerRequest(t, app, http.MethodGet, "/matches/"+matchID.String()+"/submissions", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestHandlerIngestReplayEventsAndSnapshot(t *testing.T) {
	userID := uuid.New()
	matchID := uuid.New()
	repo := &fakeReplayMatchRepo{fakeMatchServiceRepo: fakeMatchServiceRepo{match: Match{
		ID:        matchID,
		Player1ID: userID,
		Player2ID: uuid.New(),
		Status:    MatchStatusRunning,
	}}}
	app := matchesHandlerApp(NewService(repo, nil, nil), userID)

	eventsResp := matchesHandlerRequest(
		t,
		app,
		http.MethodPost,
		"/matches/"+matchID.String()+"/replay/events",
		`{"events":[{"seq":1,"t_ms":0,"event_type":"insert","payload":{"text":"a"}}]}`,
	)
	if eventsResp.StatusCode != http.StatusAccepted {
		t.Fatalf("events status = %d, want 202", eventsResp.StatusCode)
	}
	if repo.eventsInput == nil || len(repo.eventsInput.Events) != 1 {
		t.Fatalf("events input = %#v", repo.eventsInput)
	}

	snapshotResp := matchesHandlerRequest(
		t,
		app,
		http.MethodPost,
		"/matches/"+matchID.String()+"/replay/snapshots",
		`{"seq":1,"t_ms":10,"language":"go","source_code":"package main"}`,
	)
	if snapshotResp.StatusCode != http.StatusAccepted {
		t.Fatalf("snapshot status = %d, want 202", snapshotResp.StatusCode)
	}
	if repo.snapshotInput == nil || repo.snapshotInput.Language != "go" {
		t.Fatalf("snapshot input = %#v", repo.snapshotInput)
	}
}

func TestHandlerGetMatchReplay(t *testing.T) {
	userID := uuid.New()
	matchID := uuid.New()
	finishedAt := time.Now().UTC()
	repo := &fakeReplayMatchRepo{fakeMatchServiceRepo: fakeMatchServiceRepo{match: Match{
		ID:         matchID,
		Player1ID:  userID,
		Player2ID:  uuid.New(),
		Status:     MatchStatusFinished,
		FinishedAt: &finishedAt,
	}}}
	app := matchesHandlerApp(NewService(repo, nil, nil), userID)

	resp := matchesHandlerRequest(t, app, http.MethodGet, "/matches/"+matchID.String()+"/replay", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestHandlerGetPublicStats(t *testing.T) {
	repo := &fakeMatchServiceRepo{publicStats: PublicStats{
		QueuedPlayers:      2,
		ActiveMatchPlayers: 4,
		CachedAt:           time.Now().UTC(),
	}}
	app := fiber.New()
	NewHandler(NewService(repo, nil, nil), time.Minute).RegisterPublicRoutes(app)

	resp := matchesHandlerRequest(t, app, http.MethodGet, "/matches/stats", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if repo.publicStatsCalls != 1 {
		t.Fatalf("public stats calls = %d, want 1", repo.publicStatsCalls)
	}
}

func TestHandlerFinishMatch(t *testing.T) {
	matchID := uuid.New()
	winnerID := uuid.New()
	repo := &fakeMatchServiceRepo{match: Match{ID: matchID, Status: MatchStatusRunning}}
	service := NewService(repo, nil, nil)
	app := fiber.New()
	NewHandler(service, time.Minute).RegisterInternalRoutes(app)

	resp := matchesHandlerRequest(
		t,
		app,
		http.MethodPost,
		"/matches/"+matchID.String()+"/finish",
		`{"winner_id":"`+winnerID.String()+`","result_type":"player1_win","winning_reason":"accepted_faster"}`,
	)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if repo.finishInput == nil || repo.finishInput.WinnerID == nil || *repo.finishInput.WinnerID != winnerID {
		t.Fatalf("finish input = %#v", repo.finishInput)
	}
}

func TestHandlerListUserMatchesValidatesPagination(t *testing.T) {
	app := matchesHandlerApp(NewService(&fakeMatchServiceRepo{}, nil, nil), uuid.New())

	resp := matchesHandlerRequest(t, app, http.MethodGet, "/matches/user/"+uuid.NewString()+"?limit=101", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestHandlerValidationBranches(t *testing.T) {
	userID := uuid.New()
	matchID := uuid.NewString()
	app := matchesHandlerApp(NewService(&fakeMatchServiceRepo{match: Match{
		ID:        uuid.MustParse(matchID),
		Player1ID: uuid.New(),
		Player2ID: uuid.New(),
		Status:    MatchStatusRunning,
	}}, fakeTaskSelector{}, nil), userID)

	tests := []struct {
		method string
		path   string
		body   string
		want   int
	}{
		{http.MethodPost, "/matches/queue", `{"task_mode":`, http.StatusBadRequest},
		{http.MethodPost, "/matches/queue", `{"task_mode":"wild"}`, http.StatusBadRequest},
		{http.MethodPost, "/matches/", `{`, http.StatusBadRequest},
		{http.MethodPost, "/matches/", `{"player2_id":"bad"}`, http.StatusBadRequest},
		{http.MethodPost, "/matches/", `{"player2_id":"` + uuid.NewString() + `","task_mode":"wild"}`, http.StatusBadRequest},
		{http.MethodPost, "/matches/not-uuid/start", "", http.StatusBadRequest},
		{http.MethodPost, "/matches/not-uuid/cancel", "", http.StatusBadRequest},
		{http.MethodPost, "/matches/not-uuid/surrender", "", http.StatusBadRequest},
		{http.MethodGet, "/matches/not-uuid", "", http.StatusBadRequest},
		{http.MethodGet, "/matches/" + matchID, "", http.StatusForbidden},
		{http.MethodGet, "/matches/not-uuid/submissions", "", http.StatusBadRequest},
		{http.MethodGet, "/matches/not-uuid/replay", "", http.StatusBadRequest},
		{http.MethodGet, "/matches/user/not-uuid", "", http.StatusBadRequest},
		{http.MethodGet, "/matches/user/" + uuid.NewString() + "?limit=x", "", http.StatusBadRequest},
		{http.MethodGet, "/matches/user/" + uuid.NewString() + "?offset=x", "", http.StatusBadRequest},
		{http.MethodGet, "/matches/user/" + uuid.NewString() + "?offset=-1", "", http.StatusBadRequest},
	}
	for _, tt := range tests {
		resp := matchesHandlerRequest(t, app, tt.method, tt.path, tt.body)
		if resp.StatusCode != tt.want {
			t.Fatalf("%s %s status = %d, want %d", tt.method, tt.path, resp.StatusCode, tt.want)
		}
	}
}

func TestHandlerServiceErrorBranches(t *testing.T) {
	userID := uuid.New()
	matchID := uuid.New()
	taskID := uuid.New()
	baseMatch := Match{
		ID:        matchID,
		TaskID:    &taskID,
		Player1ID: userID,
		Player2ID: uuid.New(),
		Status:    MatchStatusRunning,
	}
	difficulty := TaskDifficultyEasy
	startableMatch := baseMatch
	startableMatch.Status = MatchStatusPending
	startableMatch.TaskDifficultySnapshot = &difficulty
	replayableMatch := baseMatch
	replayableMatch.Status = MatchStatusFinished
	finishedAt := time.Now().UTC()
	replayableMatch.FinishedAt = &finishedAt

	tests := []struct {
		name string
		repo *fakeMatchServiceRepo
		make func(*fiber.App) *http.Response
		want int
	}{
		{
			name: "join queue",
			repo: &fakeMatchServiceRepo{enqueueErr: errUnknownMatchTest},
			make: func(app *fiber.App) *http.Response {
				return matchesHandlerRequest(t, app, http.MethodPost, "/matches/queue", "")
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "leave queue",
			repo: &fakeMatchServiceRepo{removeErr: errUnknownMatchTest},
			make: func(app *fiber.App) *http.Response {
				return matchesHandlerRequest(t, app, http.MethodDelete, "/matches/queue", "")
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "public stats",
			repo: &fakeMatchServiceRepo{publicStatsErr: errUnknownMatchTest},
			make: func(app *fiber.App) *http.Response {
				return matchesHandlerRequest(t, app, http.MethodGet, "/matches/stats", "")
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "create match",
			repo: &fakeMatchServiceRepo{createErr: errUnknownMatchTest},
			make: func(app *fiber.App) *http.Response {
				return matchesHandlerRequest(t, app, http.MethodPost, "/matches/", `{"player2_id":"`+uuid.NewString()+`"}`)
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "start match",
			repo: &fakeMatchServiceRepo{match: startableMatch, startErr: errUnknownMatchTest},
			make: func(app *fiber.App) *http.Response {
				return matchesHandlerRequest(t, app, http.MethodPost, "/matches/"+matchID.String()+"/start", "")
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "cancel match",
			repo: &fakeMatchServiceRepo{match: startableMatch, cancelErr: errUnknownMatchTest},
			make: func(app *fiber.App) *http.Response {
				return matchesHandlerRequest(t, app, http.MethodPost, "/matches/"+matchID.String()+"/cancel", "")
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "surrender match",
			repo: &fakeMatchServiceRepo{match: baseMatch, finishErr: errUnknownMatchTest},
			make: func(app *fiber.App) *http.Response {
				return matchesHandlerRequest(t, app, http.MethodPost, "/matches/"+matchID.String()+"/surrender", "")
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "get match",
			repo: &fakeMatchServiceRepo{matchErr: errUnknownMatchTest},
			make: func(app *fiber.App) *http.Response {
				return matchesHandlerRequest(t, app, http.MethodGet, "/matches/"+matchID.String(), "")
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "current match",
			repo: &fakeMatchServiceRepo{currentErr: errUnknownMatchTest},
			make: func(app *fiber.App) *http.Response {
				return matchesHandlerRequest(t, app, http.MethodGet, "/matches/current", "")
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "submissions",
			repo: &fakeMatchServiceRepo{match: baseMatch, submissionsErr: errUnknownMatchTest},
			make: func(app *fiber.App) *http.Response {
				return matchesHandlerRequest(t, app, http.MethodGet, "/matches/"+matchID.String()+"/submissions", "")
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "replay",
			repo: &fakeMatchServiceRepo{match: replayableMatch, replayPlayersErr: errUnknownMatchTest},
			make: func(app *fiber.App) *http.Response {
				return matchesHandlerRequest(t, app, http.MethodGet, "/matches/"+matchID.String()+"/replay", "")
			},
			want: http.StatusInternalServerError,
		},
		{
			name: "user matches",
			repo: &fakeMatchServiceRepo{listErr: errUnknownMatchTest},
			make: func(app *fiber.App) *http.Response {
				return matchesHandlerRequest(t, app, http.MethodGet, "/matches/user/"+uuid.NewString(), "")
			},
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := matchesHandlerApp(NewService(tt.repo, fakeTaskSelector{taskID: taskID}, nil), userID)
			if tt.name == "public stats" {
				app = fiber.New()
				NewHandler(NewService(tt.repo, fakeTaskSelector{taskID: taskID}, nil), time.Minute).RegisterPublicRoutes(app)
			}
			resp := tt.make(app)
			if resp.StatusCode != tt.want {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}
}

func TestHandlerCurrentMatchEmptyAndBadStringUserID(t *testing.T) {
	app := matchesHandlerApp(NewService(&fakeMatchServiceRepo{}, nil, nil), uuid.New())

	resp := matchesHandlerRequest(t, app, http.MethodGet, "/matches/current", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("empty current status = %d, want 200", resp.StatusCode)
	}

	badApp := fiber.New()
	badApp.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "not-a-uuid")
		return c.Next()
	})
	NewHandler(NewService(&fakeMatchServiceRepo{}, nil, nil), time.Minute).RegisterRoutes(badApp.Group("/matches"))

	badResp := matchesHandlerRequest(t, badApp, http.MethodDelete, "/matches/queue", "")
	if badResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad user status = %d, want 401", badResp.StatusCode)
	}
}

func TestHandlerListUserMatches(t *testing.T) {
	viewerID := uuid.New()
	userID := uuid.New()
	matchID := uuid.New()
	finishedAt := time.Now().UTC()
	repo := &fakeMatchServiceRepo{match: Match{
		ID:         matchID,
		Player1ID:  userID,
		Player2ID:  viewerID,
		Status:     MatchStatusFinished,
		FinishedAt: &finishedAt,
	}}
	app := matchesHandlerApp(NewService(repo, nil, nil), viewerID)

	resp := matchesHandlerRequest(t, app, http.MethodGet, "/matches/user/"+userID.String()+"?limit=5&offset=2", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if repo.listLimit != 5 {
		t.Fatalf("list limit = %d, want 5", repo.listLimit)
	}

	var body struct {
		Items []Match `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].ID != matchID {
		t.Fatalf("body = %#v", body)
	}
}

func TestHandlerAcceptsStringUserID(t *testing.T) {
	userID := uuid.New()
	repo := &fakeMatchServiceRepo{}
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", userID.String())
		return c.Next()
	})
	NewHandler(NewService(repo, nil, nil), time.Minute).RegisterRoutes(app.Group("/matches"))

	resp := matchesHandlerRequest(t, app, http.MethodDelete, "/matches/queue", "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if len(repo.removedQueued) != 1 || repo.removedQueued[0] != userID {
		t.Fatalf("removed queued = %#v", repo.removedQueued)
	}
}

func TestHandlerRequiresAuth(t *testing.T) {
	app := matchesHandlerApp(NewService(&fakeMatchServiceRepo{}, nil, nil), uuid.Nil)

	resp := matchesHandlerRequest(t, app, http.MethodPost, "/matches/queue", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestWriteMatchErrorMapsStatuses(t *testing.T) {
	app := fiber.New()
	app.Get("/not-found", func(c *fiber.Ctx) error { return writeMatchError(c, ErrMatchNotFound) })
	app.Get("/bad", func(c *fiber.Ctx) error { return writeMatchError(c, ErrInvalidTaskMode) })
	app.Get("/conflict", func(c *fiber.Ctx) error { return writeMatchError(c, ErrMatchAlreadyStarted) })
	app.Get("/unprocessable", func(c *fiber.Ctx) error { return writeMatchError(c, ErrNoTasksAvailable) })
	app.Get("/forbidden", func(c *fiber.Ctx) error { return writeMatchError(c, ErrMatchReplayForbidden) })
	app.Get("/unknown", func(c *fiber.Ctx) error { return writeMatchError(c, errUnknownMatchTest) })

	tests := map[string]int{
		"/not-found":     http.StatusNotFound,
		"/bad":           http.StatusBadRequest,
		"/conflict":      http.StatusConflict,
		"/unprocessable": http.StatusUnprocessableEntity,
		"/forbidden":     http.StatusForbidden,
		"/unknown":       http.StatusInternalServerError,
	}
	for path, want := range tests {
		resp := matchesHandlerRequest(t, app, http.MethodGet, path, "")
		if resp.StatusCode != want {
			t.Fatalf("%s status = %d, want %d", path, resp.StatusCode, want)
		}
	}
}

func matchesHandlerApp(service *Service, userID uuid.UUID) *fiber.App {
	app := fiber.New()
	if userID != uuid.Nil {
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("user_id", userID)
			return c.Next()
		})
	}
	NewHandler(service, time.Minute).RegisterRoutes(app.Group("/matches"))
	return app
}

func matchesHandlerRequest(t *testing.T, app *fiber.App, method, path, body string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	return resp
}

type fakeReplayMatchRepo struct {
	fakeMatchServiceRepo
	eventsInput   *MatchReplayEventsIngestInput
	snapshotInput *MatchReplaySnapshotIngestInput
}

func (r *fakeReplayMatchRepo) IngestMatchReplayEvents(_ context.Context, input MatchReplayEventsIngestInput) (MatchReplayEventsIngestResult, error) {
	copied := input
	r.eventsInput = &copied
	return MatchReplayEventsIngestResult{Ingested: len(input.Events), LastSeq: input.Events[len(input.Events)-1].Seq}, nil
}

func (r *fakeReplayMatchRepo) IngestMatchReplaySnapshot(_ context.Context, input MatchReplaySnapshotIngestInput) (MatchReplaySnapshotIngestResult, error) {
	copied := input
	r.snapshotInput = &copied
	return MatchReplaySnapshotIngestResult{Inserted: true, LastSeq: input.Seq}, nil
}

func (r *fakeReplayMatchRepo) ListMatchReplayTimelineEvents(context.Context, uuid.UUID) ([]MatchReplayCapturedEvent, error) {
	return []MatchReplayCapturedEvent{{
		UserID:    r.match.Player1ID,
		Seq:       1,
		TMs:       0,
		EventType: "insert",
		Payload:   []byte(`{"text":"a"}`),
	}}, nil
}

func (r *fakeReplayMatchRepo) ListMatchReplayTimelineSnapshots(context.Context, uuid.UUID) ([]MatchReplayCapturedSnapshot, error) {
	return []MatchReplayCapturedSnapshot{{
		UserID:     r.match.Player1ID,
		Seq:        1,
		TMs:        0,
		Language:   "go",
		SourceCode: "package main",
	}}, nil
}
