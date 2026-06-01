package submissions

import (
	"context"
	"testing"
	"time"

	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/testsupport"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPGRepositoryIntegrationSubmissionAndCodeRunLifecycle(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	repo := NewPGRepository(db)

	userID := createSubmissionTestUser(t, db, ctx)
	taskID, sampleTestID, hiddenTestID := createSubmissionTestTask(t, db, ctx)

	now := time.Now().UTC()
	sub := &Submission{
		ID:              uuid.NewString(),
		TaskID:          taskID.String(),
		UserID:          userID.String(),
		Language:        "go",
		LanguageVersion: "1.24",
		SourceCode:      "package main\nfunc main() {}\n",
		Status:          StatusQueued,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := repo.Create(ctx, sub); err != nil {
		t.Fatalf("Create submission returned error: %v", err)
	}
	if err := repo.MarkSentToJudge(ctx, sub.ID, "judge-"+sub.ID, JudgeStatusQueued); err != nil {
		t.Fatalf("MarkSentToJudge returned error: %v", err)
	}
	startedAt := now.Add(time.Second)
	if err := repo.MarkRunning(ctx, sub.ID, JudgeStatusRunning, &startedAt); err != nil {
		t.Fatalf("MarkRunning returned error: %v", err)
	}
	verdict := VerdictAccepted
	if err := repo.Complete(ctx, sub.ID, CompleteInput{
		Status:      StatusFinished,
		JudgeStatus: JudgeStatusFinished,
		Verdict:     &verdict,
		PassedTests: 2,
		TotalTests:  2,
		Score:       100,
		StartedAt:   &startedAt,
		FinishedAt:  &now,
	}, []SubmissionTestResult{
		{SubmissionID: sub.ID, TestCaseID: sampleTestID.String(), Status: VerdictAccepted},
		{SubmissionID: sub.ID, TestCaseID: hiddenTestID.String(), Status: VerdictAccepted},
	}); err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	loaded, tests, err := repo.GetByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if loaded.Status != StatusFinished || loaded.Verdict == nil || *loaded.Verdict != VerdictAccepted {
		t.Fatalf("loaded submission = %#v", loaded)
	}
	if len(tests) != 2 {
		t.Fatalf("submission tests = %d, want 2", len(tests))
	}
	byJudgeID, err := repo.GetByJudgeSubmissionID(ctx, "judge-"+sub.ID)
	if err != nil || byJudgeID.ID != sub.ID {
		t.Fatalf("GetByJudgeSubmissionID = %#v, %v", byJudgeID, err)
	}

	run := &CodeRun{
		ID:              uuid.NewString(),
		TaskID:          taskID.String(),
		UserID:          userID.String(),
		Language:        "go",
		LanguageVersion: "1.24",
		SourceCode:      "package main\nfunc main() {}\n",
		Status:          StatusQueued,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := repo.CreateCodeRun(ctx, run); err != nil {
		t.Fatalf("CreateCodeRun returned error: %v", err)
	}
	if err := repo.MarkCodeRunSentToJudge(ctx, run.ID, "judge-"+run.ID, JudgeStatusQueued); err != nil {
		t.Fatalf("MarkCodeRunSentToJudge returned error: %v", err)
	}
	if err := repo.MarkCodeRunRunning(ctx, run.ID, JudgeStatusRunning, &startedAt); err != nil {
		t.Fatalf("MarkCodeRunRunning returned error: %v", err)
	}
	testName := "Custom test"
	sampleTestIDString := sampleTestID.String()
	if err := repo.CompleteCodeRun(ctx, run.ID, CompleteInput{
		Status:      StatusFinished,
		JudgeStatus: JudgeStatusFinished,
		Verdict:     &verdict,
		PassedTests: 1,
		TotalTests:  1,
		Score:       100,
	}, []CodeRunTestResult{
		{CodeRunID: run.ID, TestCaseID: &sampleTestIDString, Status: VerdictAccepted},
		{CodeRunID: run.ID, TestName: &testName, IsCustom: true, Status: VerdictAccepted},
	}); err != nil {
		t.Fatalf("CompleteCodeRun returned error: %v", err)
	}
	loadedRun, runTests, err := repo.GetCodeRunByID(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetCodeRunByID returned error: %v", err)
	}
	if loadedRun.Status != StatusFinished || len(runTests) != 2 {
		t.Fatalf("loaded run = %#v tests=%#v", loadedRun, runTests)
	}
}

func TestPGRepositoryIntegrationJudgeTaskReaders(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	repo := NewPGRepository(db)

	taskID, _, _ := createSubmissionTestTask(t, db, ctx)

	limits, err := repo.GetJudgeTaskLimits(ctx, taskID.String())
	if err != nil {
		t.Fatalf("GetJudgeTaskLimits returned error: %v", err)
	}
	if limits.TimeLimitMS != 1500 || limits.MemoryLimitKB != 131072 {
		t.Fatalf("limits = %#v", limits)
	}
	allTests, err := repo.ListJudgeTaskTestCases(ctx, taskID.String())
	if err != nil || len(allTests) != 2 {
		t.Fatalf("ListJudgeTaskTestCases = %#v, %v", allTests, err)
	}
	sampleTests, err := repo.ListJudgeTaskSampleTestCases(ctx, taskID.String())
	if err != nil || len(sampleTests) != 1 {
		t.Fatalf("ListJudgeTaskSampleTestCases = %#v, %v", sampleTests, err)
	}
}

func createSubmissionTestUser(t *testing.T, db *pgxpool.Pool, ctx context.Context) uuid.UUID {
	t.Helper()
	suffix := uuid.NewString()
	user, err := auth.NewRepository(db).CreateUser(ctx, auth.User{
		ID:           uuid.New(),
		Email:        "submissions-" + suffix + "@example.com",
		Username:     "sub_" + suffix[:8],
		DisplayName:  "Sub " + suffix[:8],
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

func createSubmissionTestTask(t *testing.T, db *pgxpool.Pool, ctx context.Context) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	taskID := uuid.New()
	sampleTestID := uuid.New()
	hiddenTestID := uuid.New()
	slug := "it-" + taskID.String()

	_, err := db.Exec(ctx, `
		insert into tasks (id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, frontend_payload)
		values ($1, $2, 'Integration Task', 'Solve it', 'easy', 1500, 131072, $3::jsonb)
	`, taskID, slug, `{"id":"`+slug+`","title":"Integration Task","supportedLanguages":["go"],"starterCodeByLanguage":{"go":"package main"}}`)
	if err != nil {
		t.Fatalf("insert fixture task: %v", err)
	}
	_, err = db.Exec(ctx, `
		insert into task_languages (id, task_id, language, language_version, is_enabled)
		values ($1, $2, 'go', '1.24', true)
	`, uuid.New(), taskID)
	if err != nil {
		t.Fatalf("insert fixture task language: %v", err)
	}
	_, err = db.Exec(ctx, `
		insert into task_test_cases (id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight)
		values
			($1, $3, 'sample', 1, '1 2', '3', 1),
			($2, $3, 'hidden', 2, '2 3', '5', 2)
	`, sampleTestID, hiddenTestID, taskID)
	if err != nil {
		t.Fatalf("insert fixture task tests: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, `
			delete from code_run_test_results
			where code_run_id in (select id from code_runs where task_id = $1)
		`, taskID)
		_, _ = db.Exec(ctx, `
			delete from submission_test_results
			where submission_id in (select id from submissions where task_id = $1)
		`, taskID)
		_, _ = db.Exec(ctx, "delete from code_runs where task_id = $1", taskID)
		_, _ = db.Exec(ctx, "delete from submissions where task_id = $1", taskID)
		_, _ = db.Exec(ctx, "delete from task_test_cases where task_id = $1", taskID)
		_, _ = db.Exec(ctx, "delete from task_languages where task_id = $1", taskID)
		_, _ = db.Exec(ctx, "delete from tasks where id = $1", taskID)
	})

	return taskID, sampleTestID, hiddenTestID
}
