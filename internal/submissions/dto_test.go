package submissions

import (
	"testing"
	"time"
)

func TestToSubmissionResponse(t *testing.T) {
	startedAt := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(time.Second)
	verdict := VerdictAccepted
	matchID := "match-1"
	judgeID := "judge-1"
	judgeStatus := JudgeStatusFinished
	timeMs := 12

	resp := ToSubmissionResponse(&Submission{
		ID:                "submission-1",
		MatchID:           &matchID,
		TaskID:            "task-1",
		UserID:            "user-1",
		Language:          "go",
		LanguageVersion:   "1.24",
		Status:            StatusFinished,
		Verdict:           &verdict,
		JudgeSubmissionID: &judgeID,
		JudgeStatus:       &judgeStatus,
		PassedTests:       1,
		TotalTests:        1,
		Score:             100,
		CreatedAt:         startedAt,
		StartedAt:         &startedAt,
		FinishedAt:        &finishedAt,
		UpdatedAt:         finishedAt,
	}, []SubmissionTestResult{
		{TestCaseID: "case-1", Status: VerdictAccepted, TimeMs: &timeMs},
	})

	if resp.ID != "submission-1" || resp.MatchID == nil || *resp.MatchID != "match-1" {
		t.Fatalf("response identity = %#v", resp)
	}
	if resp.CreatedAt != "2026-05-30T10:00:00Z" || resp.StartedAt == nil || *resp.StartedAt != "2026-05-30T10:00:00Z" {
		t.Fatalf("response times = %#v", resp)
	}
	if len(resp.TestResults) != 1 || resp.TestResults[0].TestCaseID != "case-1" || resp.TestResults[0].TimeMs == nil {
		t.Fatalf("test results = %#v", resp.TestResults)
	}
}

func TestToCodeRunResponse(t *testing.T) {
	startedAt := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(time.Second)
	verdict := VerdictAccepted
	judgeID := "judge-run-1"
	judgeStatus := JudgeStatusFinished
	testName := "Custom test"

	resp := ToCodeRunResponse(&CodeRun{
		ID:                "run-1",
		TaskID:            "task-1",
		UserID:            "user-1",
		Language:          "go",
		LanguageVersion:   "1.24",
		Status:            StatusFinished,
		Verdict:           &verdict,
		JudgeSubmissionID: &judgeID,
		JudgeStatus:       &judgeStatus,
		PassedTests:       1,
		TotalTests:        1,
		Score:             100,
		CreatedAt:         startedAt,
		StartedAt:         &startedAt,
		FinishedAt:        &finishedAt,
		UpdatedAt:         finishedAt,
	}, []CodeRunTestResult{
		{TestName: &testName, IsCustom: true, Status: VerdictAccepted},
	})

	if resp.ID != "run-1" || resp.TaskID != "task-1" {
		t.Fatalf("response identity = %#v", resp)
	}
	if resp.FinishedAt == nil || *resp.FinishedAt != "2026-05-30T10:00:01Z" {
		t.Fatalf("finished at = %#v", resp.FinishedAt)
	}
	if len(resp.TestResults) != 1 || !resp.TestResults[0].IsCustom || resp.TestResults[0].TestName == nil {
		t.Fatalf("test results = %#v", resp.TestResults)
	}
}
