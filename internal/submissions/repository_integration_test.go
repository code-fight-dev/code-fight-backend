package submissions

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/judge"
	"code-fight-backend/internal/testsupport"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPGJudgeAdmissionAfterLostAcknowledgement(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	repo := NewPGRepository(db)
	userID := createSubmissionTestUser(t, db, ctx)
	taskID, _, _ := createSubmissionTestTask(t, db, ctx)
	type acceptedJob struct{ submissionID, judgeID string }
	accepted := make(chan acceptedJob, maxActiveJudgeJobsPerUser+1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/languages" {
			_ = json.NewEncoder(w).Encode([]judge.SupportedLanguage{{Language: "go", Version: "1.24"}})
			return
		}
		var req judge.SubmitRequest
		if r.Method != http.MethodPost || r.URL.Path != "/submissions" || json.NewDecoder(r.Body).Decode(&req) != nil {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		accepted <- acceptedJob{req.SubmissionID, uuid.NewString()}
		conn, _, err := http.NewResponseController(w).Hijack()
		if err != nil {
			t.Errorf("drop Judge acknowledgement: %v", err)
			return
		}
		_ = conn.Close()
	}))
	defer server.Close()
	service := NewService(repo, NewPGTaskRepository(db), nil, judge.NewHTTPClient(server.URL, 5*time.Second, "token"),
		nil, "http://backend/internal/judge/callback", "token", 10_000, nil)

	for i := range maxActiveJudgeJobsPerUser {
		var err error
		if i%2 == 0 {
			_, err = service.CreateTaskSubmission(ctx, userID, taskID, validSubmissionRequest())
		} else {
			_, err = service.CreateTaskCodeRun(ctx, userID, taskID, validCodeRunRequest())
		}
		if !errors.Is(err, ErrJudgeSubmitFailed) {
			t.Fatalf("job %d should lose the acknowledgement, got %v", i, err)
		}
	}
	for _, table := range []string{"submissions", "code_runs"} {
		result, err := db.Exec(ctx, `update `+table+` set updated_at = now() - interval '1 hour'
			where user_id = $1 and status = 'queued' and judge_submission_id is null`, userID)
		if err != nil || result.RowsAffected() != 2 {
			t.Fatalf("backdate unacknowledged %s: %v (%v)", table, err, result)
		}
	}
	if _, err := service.CreateTaskSubmission(ctx, userID, taskID, validSubmissionRequest()); !errors.Is(err, ErrJudgeActiveLimit) {
		t.Fatalf("fifth job must be blocked while all four are active: %v", err)
	}
	if len(accepted) != maxActiveJudgeJobsPerUser {
		t.Fatalf("Judge received %d jobs", len(accepted))
	}
	// Recover both a submission and a code run from the authenticated callback.
	verdict := VerdictAccepted
	for i, table := range []string{"submissions", "code_runs"} {
		job := <-accepted
		startedAt := time.Now().UTC()
		if err := service.HandleJudgeCallback(ctx, judge.CallbackPayload{
			SubmissionID: job.submissionID, JudgeSubmissionID: job.judgeID,
			JudgeStatus: JudgeStatusFinished, Verdict: &verdict, StartedAt: &startedAt, FinishedAt: &startedAt,
		}); err != nil {
			t.Fatalf("finish unacknowledged job %s: %v", job.submissionID, err)
		}
		acknowledge := repo.MarkSentToJudge
		if i == 1 {
			acknowledge = repo.MarkCodeRunSentToJudge
		}
		if err := acknowledge(ctx, job.submissionID, job.judgeID, JudgeStatusQueued); err != nil {
			t.Fatalf("record late acknowledgement: %v", err)
		}
		var status string
		if err := db.QueryRow(ctx, `select status from `+table+` where id=$1`, job.submissionID).Scan(&status); err != nil || status != StatusFinished {
			t.Fatalf("late acknowledgement must preserve finished state: %q (%v)", status, err)
		}
	}
	for range 2 {
		if _, err := service.CreateTaskSubmission(ctx, userID, taskID, validSubmissionRequest()); !errors.Is(err, ErrJudgeSubmitFailed) {
			t.Fatalf("job should reach Judge after a slot is released: %v", err)
		}
	}
	if _, err := service.CreateTaskCodeRun(ctx, userID, taskID, validCodeRunRequest()); !errors.Is(err, ErrJudgeActiveLimit) {
		t.Fatalf("job limit must still apply after both slots are filled: %v", err)
	}
}

func TestPGRepositoryConcurrentJudgeAdmission(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	repo := NewPGRepository(db)
	userID := createSubmissionTestUser(t, db, ctx)
	taskID, _, _ := createSubmissionTestTask(t, db, ctx)
	now := time.Now().UTC()

	const requests = 8
	start := make(chan struct{})
	results := make(chan error, requests)
	for i := range requests {
		go func() {
			<-start
			if i%2 == 0 {
				results <- repo.Create(ctx, &Submission{
					ID: uuid.NewString(), TaskID: taskID.String(), UserID: userID.String(),
					Language: "go", LanguageVersion: "1.24", SourceCode: "package main",
					Status: StatusQueued, CreatedAt: now, UpdatedAt: now,
				})
				return
			}
			results <- repo.CreateCodeRun(ctx, &CodeRun{
				ID: uuid.NewString(), TaskID: taskID.String(), UserID: userID.String(),
				Language: "go", LanguageVersion: "1.24", SourceCode: "package main",
				Status: StatusQueued, CreatedAt: now, UpdatedAt: now,
			})
		}()
	}
	close(start)

	admitted, rejected := 0, 0
	for range requests {
		switch err := <-results; {
		case err == nil:
			admitted++
		case errors.Is(err, ErrJudgeActiveLimit):
			rejected++
		default:
			t.Errorf("admit judge job: %v", err)
		}
	}
	if admitted != maxActiveJudgeJobsPerUser || rejected != requests-maxActiveJudgeJobsPerUser {
		t.Fatalf("admitted %d, rejected %d", admitted, rejected)
	}
}

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
		StartedAt:   &startedAt,
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

func TestPGRepositoryIntegrationCallbackAttemptOrdering(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	repo := NewPGRepository(db)
	userID := createSubmissionTestUser(t, db, ctx)
	taskID, testID, _ := createSubmissionTestTask(t, db, ctx)
	now := time.Now().UTC().Truncate(time.Microsecond)
	old, newer := now.Add(-time.Second), now.Add(time.Second)
	for _, table := range []string{"submissions", "code_runs"} {
		t.Run(table, func(t *testing.T) {
			id := uuid.NewString()
			_, err := db.Exec(ctx, `insert into `+table+` (id, task_id, user_id, language, language_version, source_code, status, created_at, updated_at)
				values ($1, $2, $3, 'go', '1.24', 'package main', 'sent_to_judge', now(), now())`, id, taskID, userID)
			if err != nil {
				t.Fatal(err)
			}
			mark := repo.MarkRunning
			resultsTable, foreignKey := "submission_test_results", "submission_id"
			if table == "code_runs" {
				mark = repo.MarkCodeRunRunning
				resultsTable, foreignKey = "code_run_test_results", "code_run_id"
			}
			finish := func(at *time.Time) error {
				verdict := VerdictAccepted
				input := CompleteInput{Status: StatusFinished, JudgeStatus: JudgeStatusFinished, Verdict: &verdict, StartedAt: at, Score: 100}
				if table == "submissions" {
					return repo.Complete(ctx, id, input, []SubmissionTestResult{{SubmissionID: id, TestCaseID: testID.String(), Status: VerdictAccepted}})
				}
				testIDString := testID.String()
				return repo.CompleteCodeRun(ctx, id, input, []CodeRunTestResult{{CodeRunID: id, TestCaseID: &testIDString, Status: VerdictAccepted}})
			}
			if err := mark(ctx, id, JudgeStatusRunning, &old); err != nil {
				t.Fatal(err)
			}
			if err := mark(ctx, id, JudgeStatusRunning, &now); err != nil {
				t.Fatal(err)
			}
			for _, at := range []*time.Time{&old, nil} {
				if err := mark(ctx, id, JudgeStatusRunning, at); !errors.Is(err, ErrStaleJudgeCallback) {
					t.Fatalf("stale running: %v", err)
				}
				if err := finish(at); !errors.Is(err, ErrStaleJudgeCallback) {
					t.Fatalf("stale completion: %v", err)
				}
			}
			if err := finish(&now); err != nil {
				t.Fatal(err)
			}
			var resultID string
			if err := db.QueryRow(ctx, `select id from `+resultsTable+` where `+foreignKey+`=$1`, id).Scan(&resultID); err != nil {
				t.Fatal(err)
			}
			for _, at := range []*time.Time{&old, &now, nil} {
				if err := finish(at); !errors.Is(err, ErrStaleJudgeCallback) {
					t.Fatalf("duplicate/stale completion: %v", err)
				}
			}
			var preserved bool
			if err := db.QueryRow(ctx, `select exists(select 1 from `+resultsTable+` where id=$1)`, resultID).Scan(&preserved); err != nil || !preserved {
				t.Fatalf("stale callback replaced test results: preserved=%v err=%v", preserved, err)
			}
			if err := mark(ctx, id, JudgeStatusRunning, &newer); err != nil {
				t.Fatal(err)
			}
			if err := finish(&newer); err != nil {
				t.Fatal(err)
			}
			var stored time.Time
			if err := db.QueryRow(ctx, `select started_at from `+table+` where id=$1`, id).Scan(&stored); err != nil || !stored.Equal(newer) {
				t.Fatalf("latest attempt=%v want=%v err=%v", stored, newer, err)
			}
		})
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
