package tasks

import (
	"context"
	"testing"
	"time"

	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/testsupport"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryIntegrationChallengeCatalogAndUserProgress(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	repo := NewRepository(db)

	userID := createTasksTestUser(t, db, ctx)
	taskID, slug := createTasksTestTask(t, db, ctx)
	insertTasksTestSubmission(t, db, ctx, taskID, userID, true)

	items, err := repo.ListChallenges(ctx)
	if err != nil {
		t.Fatalf("ListChallenges returned error: %v", err)
	}
	item := findChallengeItem(t, items, slug)
	if item.TaskID != taskID.String() || item.Difficulty != "Medium" || item.AcceptanceRate != 100 {
		t.Fatalf("challenge item = %#v", item)
	}

	userItems, err := repo.ListChallengesForUser(ctx, userID.String())
	if err != nil {
		t.Fatalf("ListChallengesForUser returned error: %v", err)
	}
	userItem := findChallengeItem(t, userItems, slug)
	if userItem.Progress != "solved" {
		t.Fatalf("user progress = %q, want solved", userItem.Progress)
	}

	stats, err := repo.GetUserTaskStats(ctx, userID.String())
	if err != nil {
		t.Fatalf("GetUserTaskStats returned error: %v", err)
	}
	if stats.SolvedTasks == 0 {
		t.Fatalf("stats = %#v, want solved task", stats)
	}

	submissions, err := repo.ListUserTaskSubmissions(ctx, taskID.String(), userID.String())
	if err != nil {
		t.Fatalf("ListUserTaskSubmissions returned error: %v", err)
	}
	if len(submissions) != 1 || submissions[0].TaskID != taskID.String() {
		t.Fatalf("submissions = %#v", submissions)
	}

	bySlug, err := repo.GetChallengeBySlug(ctx, slug)
	if err != nil {
		t.Fatalf("GetChallengeBySlug returned error: %v", err)
	}
	if bySlug == nil || bySlug.TaskID != taskID.String() || len(bySlug.TestCases) == 0 {
		t.Fatalf("challenge by slug = %#v", bySlug)
	}
	byID, err := repo.GetChallengeByID(ctx, taskID.String())
	if err != nil {
		t.Fatalf("GetChallengeByID returned error: %v", err)
	}
	if byID == nil || byID.Slug != slug {
		t.Fatalf("challenge by id = %#v", byID)
	}
}

func createTasksTestUser(t *testing.T, db *pgxpool.Pool, ctx context.Context) uuid.UUID {
	t.Helper()
	suffix := uuid.NewString()
	user, err := auth.NewRepository(db).CreateUser(ctx, auth.User{
		ID:           uuid.New(),
		Email:        "tasks-" + suffix + "@example.com",
		Username:     "task_" + suffix[:8],
		DisplayName:  "Task " + suffix[:8],
		PasswordHash: "hash",
		AvatarSource: auth.AvatarSourceNone,
	})
	if err != nil {
		t.Fatalf("create fixture user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "delete from users where id = $1", user.ID)
	})
	return user.ID
}

func createTasksTestTask(t *testing.T, db *pgxpool.Pool, ctx context.Context) (uuid.UUID, string) {
	t.Helper()
	taskID := uuid.New()
	sampleID := uuid.New()
	hiddenID := uuid.New()
	slug := "task-it-" + taskID.String()
	payload := `{
		"id":"` + slug + `",
		"title":"Integration Challenge",
		"summary":"Integration summary",
		"tags":["integration"],
		"category":"Testing",
		"kind":"algorithmic",
		"supportedLanguages":["go","python"],
		"starterCodeByLanguage":{"go":"package main","python":"print()"},
		"estimatedMinutes":15,
		"popularity":7
	}`

	_, err := db.Exec(ctx, `
		insert into tasks (id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, frontend_payload)
		values ($1, $2, 'Integration Challenge', 'Solve it', 'medium', 1500, 131072, $3::jsonb)
	`, taskID, slug, payload)
	if err != nil {
		t.Fatalf("insert fixture task: %v", err)
	}
	_, err = db.Exec(ctx, `
		insert into task_languages (id, task_id, language, language_version, is_enabled)
		values
			($1, $3, 'go', '1.24', true),
			($2, $3, 'python', '3.12', true)
	`, uuid.New(), uuid.New(), taskID)
	if err != nil {
		t.Fatalf("insert fixture languages: %v", err)
	}
	_, err = db.Exec(ctx, `
		insert into task_test_cases (id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight)
		values
			($1, $3, 'sample', 1, '1 2', '3', 1),
			($2, $3, 'hidden', 2, '2 3', '5', 1)
	`, sampleID, hiddenID, taskID)
	if err != nil {
		t.Fatalf("insert fixture tests: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, `
			delete from submission_test_results
			where submission_id in (select id from submissions where task_id = $1)
		`, taskID)
		_, _ = db.Exec(ctx, "delete from submissions where task_id = $1", taskID)
		_, _ = db.Exec(ctx, "delete from task_test_cases where task_id = $1", taskID)
		_, _ = db.Exec(ctx, "delete from task_languages where task_id = $1", taskID)
		_, _ = db.Exec(ctx, "delete from tasks where id = $1", taskID)
	})
	return taskID, slug
}

func insertTasksTestSubmission(t *testing.T, db *pgxpool.Pool, ctx context.Context, taskID uuid.UUID, userID uuid.UUID, accepted bool) {
	t.Helper()
	verdict := "wrong_answer"
	if accepted {
		verdict = "accepted"
	}
	now := time.Now().UTC()
	_, err := db.Exec(ctx, `
		insert into submissions (
			id, task_id, user_id, language, language_version, source_code,
			status, verdict, judge_status, passed_tests, total_tests, score,
			created_at, updated_at
		)
		values ($1, $2, $3, 'go', '1.24', 'package main', 'finished', $4, 'finished', 1, 1, 100, $5, $5)
	`, uuid.New(), taskID, userID, verdict, now)
	if err != nil {
		t.Fatalf("insert fixture submission: %v", err)
	}
}

func findChallengeItem(t *testing.T, items []ChallengeListItem, slug string) ChallengeListItem {
	t.Helper()
	for _, item := range items {
		if item.Slug == slug {
			return item
		}
	}
	t.Fatalf("challenge %q not found in %#v", slug, items)
	return ChallengeListItem{}
}
