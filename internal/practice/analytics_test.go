package practice

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/testsupport"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestPracticeAPI(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	newUser := func() uuid.UUID {
		id := uuid.New()
		name := "practice_" + id.String()[:8]
		if _, err := auth.NewRepository(db).CreateUser(ctx, auth.User{
			ID: id, Email: name + "@example.test", Username: name, DisplayName: name,
			PasswordHash: "hash", AvatarSource: auth.AvatarSourceNone,
		}); err != nil {
			t.Fatal(err)
		}
		return id
	}
	user, other := newUser(), newUser()
	var task uuid.UUID
	if err := db.QueryRow(ctx, `select id from tasks where slug='mirror-log'`).Scan(&task); err != nil {
		t.Fatal(err)
	}
	match := uuid.New()
	if _, err := db.Exec(ctx, `insert into matches(id, task_id, player1_id, player2_id) values($1,$2,$3,$4)`, match, task, user, other); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, `delete from code_runs where user_id=$1`, user)
		_, _ = db.Exec(ctx, `delete from submissions where user_id in ($1,$2)`, user, other)
		_, _ = db.Exec(ctx, `delete from matches where id=$1`, match)
		_, _ = db.Exec(ctx, `delete from users where id in ($1,$2)`, user, other)
	})
	start := time.Date(2025, 2, 1, 23, 30, 0, 0, time.UTC)
	insert := func(owner uuid.UUID, matchID *uuid.UUID, at time.Time, language, verdict string) uuid.UUID {
		id := uuid.New()
		if _, err := db.Exec(ctx, `insert into submissions(id, match_id, task_id, user_id, language, language_version,
			source_code, status, verdict, created_at, finished_at) values($1,$2,$3,$4,$5,'21','private source','finished',$6,$7,$8)`,
			id, matchID, task, owner, language, verdict, at, at.Add(12*time.Hour)); err != nil {
			t.Fatal(err)
		}
		return id
	}
	insert(user, nil, start, "java", "accepted")
	first := insert(user, nil, start.Add(24*time.Hour), "java", "accepted")
	second := insert(user, nil, start.Add(24*time.Hour), "go", "wrong_answer")
	insert(user, &match, start.Add(time.Hour), "java", "accepted")
	insert(other, nil, start, "java", "accepted")
	if _, err := db.Exec(ctx, `insert into code_runs(id,task_id,user_id,language,language_version,source_code,status,verdict,created_at)
		values($1,$2,$3,'java','21','private trial','finished','accepted',$4)`, uuid.New(), task, user, start); err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		if c.Get("Test-Session") == "present" {
			c.Locals("user_id", user)
		}
		return c.Next()
	})
	RegisterRoutes(app.Group("/api/practice"), db)
	get := func(path string, session bool, status int, out any) {
		t.Helper()
		req := httptest.NewRequest("GET", "/api/practice"+path, nil)
		if session {
			req.Header.Set("Test-Session", "present")
		}
		res, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != status {
			body, _ := io.ReadAll(res.Body)
			t.Fatalf("GET %s status=%d, want %d: %s", path, res.StatusCode, status, body)
		}
		if out != nil {
			if err := json.NewDecoder(res.Body).Decode(out); err != nil {
				t.Fatal(err)
			}
		}
	}
	get("/summary", false, 401, nil)
	get("/summary?timeZone=invalid", true, 400, nil)
	get("/submissions?cursor=invalid", true, 400, nil)
	var summary Summary
	get("/summary?from=2025-02-02&to=2025-02-04&timeZone=Europe%2FParis", true, 200, &summary)
	if summary.Totals.Submissions != 3 || summary.Totals.AttemptedTasks != 1 || summary.Totals.SolvedTasks != 1 ||
		summary.Streaks.LongestDays != 2 || summary.Streaks.CurrentDays != 0 || len(summary.Activity.Days) != 3 ||
		summary.Activity.Days[0] != (DailyCounts{Date: "2025-02-02", Submissions: 1, FirstSolves: 1}) ||
		summary.Activity.Days[1] != (DailyCounts{Date: "2025-02-03", Submissions: 2}) ||
		summary.Activity.Days[2] != (DailyCounts{Date: "2025-02-04"}) {
		t.Fatalf("unexpected practice summary: %+v", summary)
	}
	for _, dimension := range []struct {
		key   string
		items []TaskCounts
	}{{"easy", summary.ByDifficulty}, {"Strings", summary.ByCategory}, {"Two Pointers", summary.ByTag}} {
		if !slices.ContainsFunc(dimension.items, func(item TaskCounts) bool {
			return item.Key == dimension.key && item.AttemptedTasks == 1 && item.SolvedTasks == 1 && item.AvailableTasks >= 1
		}) {
			t.Fatalf("missing progress for %s: %+v", dimension.key, dimension.items)
		}
	}
	if !slices.Contains(summary.ByLanguage, LanguageCounts{Language: "java", Submissions: 2, AcceptedSubmissions: 2, AttemptedTasks: 1, SolvedTasks: 1}) {
		t.Fatalf("unexpected language counts: %+v", summary.ByLanguage)
	}
	get("/summary?from=2025-02-03&to=2025-02-03&timeZone=Europe%2FParis", true, 200, &summary)
	if len(summary.Activity.Days) != 1 || summary.Activity.Days[0].FirstSolves != 0 {
		t.Fatalf("repeat accepted counted as a first solve: %+v", summary.Activity.Days)
	}
	type page struct {
		Items      []Submission
		NextCursor string
	}
	var a, b, filtered page
	get("/submissions?limit=1", true, 200, &a)
	if len(a.Items) != 1 || a.NextCursor == "" {
		t.Fatalf("first page: %+v", a)
	}
	get("/submissions?limit=1&cursor="+a.NextCursor, true, 200, &b)
	if len(b.Items) != 1 || a.Items[0].SubmissionID == b.Items[0].SubmissionID ||
		(a.Items[0].SubmissionID != first && a.Items[0].SubmissionID != second) ||
		(b.Items[0].SubmissionID != first && b.Items[0].SubmissionID != second) {
		t.Fatalf("pages: %+v %+v", a, b)
	}
	get("/submissions?language=JAVA&verdict=accepted", true, 200, &filtered)
	if len(filtered.Items) != 2 || filtered.NextCursor != "" {
		t.Fatalf("filtered result: %+v", filtered)
	}
}
