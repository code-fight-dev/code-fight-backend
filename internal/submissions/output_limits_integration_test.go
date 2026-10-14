package submissions

import (
	"testing"
	"time"

	"code-fight-backend/internal/judge"
	"code-fight-backend/internal/testsupport"
	"github.com/google/uuid"
)

func TestSupervisorOutputLimitsPersistThroughCallback(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	repo := NewPGRepository(db)
	userID := createSubmissionTestUser(t, db, ctx)
	taskID, sampleID, _ := createSubmissionTestTask(t, db, ctx)
	service := NewService(repo, nil, nil, nil, nil, "", "", 1024, nil)
	for _, verdict := range []string{VerdictOutputLimitExceeded, VerdictOutputDirectoryLimitExceeded} {
		for _, codeRun := range []bool{false, true} {
			t.Run(verdict+map[bool]string{false: "_submission", true: "_run"}[codeRun], func(t *testing.T) {
				id := uuid.NewString()
				judgeID := "judge-" + id
				now := time.Now().UTC()
				if codeRun {
					if err := repo.CreateCodeRun(ctx, &CodeRun{ID: id, TaskID: taskID.String(), UserID: userID.String(), Language: "javascript", LanguageVersion: "node24", SourceCode: "function Solve() { return 0; }", Status: StatusQueued, CreatedAt: now, UpdatedAt: now}); err != nil {
						t.Fatal(err)
					}
					if err := repo.MarkCodeRunSentToJudge(ctx, id, judgeID, JudgeStatusQueued); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := repo.Create(ctx, &Submission{ID: id, TaskID: taskID.String(), UserID: userID.String(), Language: "javascript", LanguageVersion: "node24", SourceCode: "function Solve() { return 0; }", Status: StatusQueued, CreatedAt: now, UpdatedAt: now}); err != nil {
						t.Fatal(err)
					}
					if err := repo.MarkSentToJudge(ctx, id, judgeID, JudgeStatusQueued); err != nil {
						t.Fatal(err)
					}
				}
				payload := judge.CallbackPayload{SubmissionID: id, JudgeSubmissionID: judgeID, JudgeStatus: JudgeStatusFinished, Verdict: &verdict, TotalTests: 1}
				if verdict == VerdictOutputLimitExceeded {
					payload.Tests = []judge.CallbackTestResult{{TestCaseID: sampleID.String(), Status: VerdictOutputLimitExceeded}}
				}
				if codeRun {
					if err := service.HandleCodeRunJudgeCallback(ctx, payload); err != nil {
						t.Fatal(err)
					}
					got, tests, err := repo.GetCodeRunByID(ctx, id)
					if err != nil || got.Verdict == nil || *got.Verdict != verdict || got.Status != StatusFinished {
						t.Fatalf("run=%+v err=%v", got, err)
					}
					if verdict == VerdictOutputLimitExceeded && (len(tests) != 1 || tests[0].Status != verdict) {
						t.Fatalf("tests=%+v", tests)
					}
				} else {
					if err := service.HandleJudgeCallback(ctx, payload); err != nil {
						t.Fatal(err)
					}
					got, tests, err := repo.GetByID(ctx, id)
					if err != nil || got.Verdict == nil || *got.Verdict != verdict || got.Status != StatusFinished {
						t.Fatalf("submission=%+v err=%v", got, err)
					}
					if verdict == VerdictOutputLimitExceeded && (len(tests) != 1 || tests[0].Status != verdict) {
						t.Fatalf("tests=%+v", tests)
					}
				}
			})
		}
	}
}
