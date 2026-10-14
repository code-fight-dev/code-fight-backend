package submissions

import (
	"context"
	"testing"

	"code-fight-backend/internal/judge"
)

func TestCodeRunCallbackAcceptsSupervisorOutputLimits(t *testing.T) {
	for _, verdict := range []string{VerdictOutputLimitExceeded, VerdictOutputDirectoryLimitExceeded} {
		t.Run(verdict, func(t *testing.T) {
			repo, run := codeRunCallbackRepo()
			service := NewService(repo, nil, nil, nil, nil, "", "", 1024, nil)
			payload := judge.CallbackPayload{JudgeSubmissionID: "judge-run-1", SubmissionID: run.ID, JudgeStatus: JudgeStatusFinished, Verdict: &verdict, TotalTests: 1}
			if verdict == VerdictOutputLimitExceeded {
				payload.Tests = []judge.CallbackTestResult{{TestCaseID: "sample-1", Status: VerdictOutputLimitExceeded}}
			}
			if err := service.HandleCodeRunJudgeCallback(context.Background(), payload); err != nil {
				t.Fatal(err)
			}
			if got := repo.runs[run.ID]; got.Status != StatusFinished || got.Verdict == nil || *got.Verdict != verdict {
				t.Fatalf("run=%+v", got)
			}
		})
	}
}
