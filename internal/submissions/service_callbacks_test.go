package submissions

import (
	"context"
	"errors"
	"testing"
	"time"

	"code-fight-backend/internal/judge"

	"github.com/google/uuid"
)

func TestHandleCodeRunJudgeCallbackMarksRunning(t *testing.T) {
	repo, run := codeRunCallbackRepo()
	service := NewService(repo, nil, nil, nil, nil, "", "", 1024, nil)
	startedAt := time.Now().UTC()

	err := service.HandleCodeRunJudgeCallback(context.Background(), judge.CallbackPayload{
		JudgeSubmissionID: "judge-run-1",
		SubmissionID:      run.ID,
		JudgeStatus:       JudgeStatusRunning,
		StartedAt:         &startedAt,
	})
	if err != nil {
		t.Fatalf("HandleCodeRunJudgeCallback returned error: %v", err)
	}

	updated := repo.runs[run.ID]
	if updated.Status != StatusRunning {
		t.Fatalf("run status = %q, want running", updated.Status)
	}
	if updated.StartedAt == nil || !updated.StartedAt.Equal(startedAt) {
		t.Fatalf("startedAt = %v, want %v", updated.StartedAt, startedAt)
	}
}

func TestHandleCodeRunJudgeCallbackCompletesRun(t *testing.T) {
	repo, run := codeRunCallbackRepo()
	service := NewService(repo, nil, nil, nil, nil, "", "", 1024, nil)
	verdict := VerdictAccepted

	err := service.HandleCodeRunJudgeCallback(context.Background(), judge.CallbackPayload{
		JudgeSubmissionID: "judge-run-1",
		SubmissionID:      run.ID,
		JudgeStatus:       JudgeStatusFinished,
		Verdict:           &verdict,
		PassedTests:       2,
		TotalTests:        2,
		Score:             100,
		Tests: []judge.CallbackTestResult{
			{TestCaseID: "sample-1", Status: VerdictAccepted},
			{TestCaseID: customCodeRunTestID, Status: VerdictAccepted},
		},
	})
	if err != nil {
		t.Fatalf("HandleCodeRunJudgeCallback returned error: %v", err)
	}

	updated := repo.runs[run.ID]
	if updated.Status != StatusFinished || updated.Verdict == nil || *updated.Verdict != VerdictAccepted {
		t.Fatalf("run after callback = %#v", updated)
	}
	if updated.PassedTests != 2 || updated.TotalTests != 2 || updated.Score != 100 {
		t.Fatalf("run score = %d/%d/%d", updated.PassedTests, updated.TotalTests, updated.Score)
	}
}

func TestHandleCodeRunJudgeCallbackRejectsInvalidPayloads(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*judge.CallbackPayload)
	}{
		{name: "wrong submission id", mutate: func(p *judge.CallbackPayload) { p.SubmissionID = "other" }},
		{name: "bad status", mutate: func(p *judge.CallbackPayload) { p.JudgeStatus = "strange" }},
		{name: "missing verdict", mutate: func(p *judge.CallbackPayload) { p.Verdict = nil }},
		{name: "bad verdict", mutate: func(p *judge.CallbackPayload) { v := "bad"; p.Verdict = &v }},
		{name: "negative score", mutate: func(p *judge.CallbackPayload) { p.Score = -1 }},
		{name: "passed exceeds total", mutate: func(p *judge.CallbackPayload) { p.PassedTests = 2; p.TotalTests = 1 }},
		{name: "unknown test", mutate: func(p *judge.CallbackPayload) { p.Tests[0].TestCaseID = "hidden-1" }},
		{name: "bad test status", mutate: func(p *judge.CallbackPayload) { p.Tests[0].Status = "weird" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, run := codeRunCallbackRepo()
			service := NewService(repo, nil, nil, nil, nil, "", "", 1024, nil)
			verdict := VerdictAccepted
			payload := judge.CallbackPayload{
				JudgeSubmissionID: "judge-run-1",
				SubmissionID:      run.ID,
				JudgeStatus:       JudgeStatusFinished,
				Verdict:           &verdict,
				PassedTests:       1,
				TotalTests:        1,
				Score:             100,
				Tests:             []judge.CallbackTestResult{{TestCaseID: "sample-1", Status: VerdictAccepted}},
			}
			tt.mutate(&payload)

			err := service.HandleCodeRunJudgeCallback(context.Background(), payload)
			if !errors.Is(err, ErrJudgeCallbackUnauthorized) {
				t.Fatalf("HandleCodeRunJudgeCallback error = %v, want ErrJudgeCallbackUnauthorized", err)
			}
		})
	}
}

func TestHandleJudgeCallbackRejectsInvalidSubmissionPayloads(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*judge.CallbackPayload)
	}{
		{name: "wrong submission id", mutate: func(p *judge.CallbackPayload) { p.SubmissionID = "other" }},
		{name: "bad status", mutate: func(p *judge.CallbackPayload) { p.JudgeStatus = "odd" }},
		{name: "missing verdict", mutate: func(p *judge.CallbackPayload) { p.Verdict = nil }},
		{name: "bad verdict", mutate: func(p *judge.CallbackPayload) { v := "bad"; p.Verdict = &v }},
		{name: "negative tests", mutate: func(p *judge.CallbackPayload) { p.PassedTests = -1 }},
		{name: "unknown test", mutate: func(p *judge.CallbackPayload) { p.Tests[0].TestCaseID = "hidden" }},
		{name: "bad test status", mutate: func(p *judge.CallbackPayload) { p.Tests[0].Status = "bad" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := submissionRepoFixture()
			sub := &Submission{
				ID:                uuid.NewString(),
				TaskID:            "task-1",
				UserID:            uuid.NewString(),
				Language:          "go",
				Status:            StatusSentToJudge,
				JudgeSubmissionID: ptrString("judge-sub-1"),
				JudgeStatus:       ptrString(JudgeStatusQueued),
			}
			repo.submissions = map[string]*Submission{sub.ID: sub}
			verdict := VerdictAccepted
			payload := judge.CallbackPayload{
				JudgeSubmissionID: "judge-sub-1",
				SubmissionID:      sub.ID,
				JudgeStatus:       JudgeStatusFinished,
				Verdict:           &verdict,
				PassedTests:       1,
				TotalTests:        1,
				Score:             100,
				Tests:             []judge.CallbackTestResult{{TestCaseID: "case-1", Status: VerdictAccepted}},
			}
			tt.mutate(&payload)
			service := NewService(repo, nil, nil, nil, nil, "", "", 1024, nil)

			err := service.HandleJudgeCallback(context.Background(), payload)
			if !errors.Is(err, ErrJudgeCallbackUnauthorized) {
				t.Fatalf("HandleJudgeCallback error = %v, want ErrJudgeCallbackUnauthorized", err)
			}
		})
	}
}

func TestCodeRunResultTestIdentity(t *testing.T) {
	testCaseID, testName, isCustom := codeRunResultTestIdentity(" custom ")
	if testCaseID != nil || testName == nil || *testName != "Custom test" || !isCustom {
		t.Fatalf("custom identity = %v/%v/%v", testCaseID, testName, isCustom)
	}

	testCaseID, testName, isCustom = codeRunResultTestIdentity(" sample-1 ")
	if testCaseID == nil || *testCaseID != "sample-1" || testName != nil || isCustom {
		t.Fatalf("sample identity = %v/%v/%v", testCaseID, testName, isCustom)
	}

	testCaseID, testName, isCustom = codeRunResultTestIdentity(" ")
	if testCaseID != nil || testName != nil || isCustom {
		t.Fatalf("empty identity = %v/%v/%v", testCaseID, testName, isCustom)
	}
}

func TestSubmissionScopeAndNullableVerdict(t *testing.T) {
	if got := submissionScope(&Submission{}); got != "task" {
		t.Fatalf("submissionScope task = %q", got)
	}
	matchID := uuid.NewString()
	if got := submissionScope(&Submission{MatchID: &matchID}); got != "match" {
		t.Fatalf("submissionScope match = %q", got)
	}
	if got := nullableVerdict(nil); got != "none" {
		t.Fatalf("nil verdict = %q", got)
	}
	verdict := " accepted "
	if got := nullableVerdict(&verdict); got != "accepted" {
		t.Fatalf("verdict = %q", got)
	}
}

func TestHandleJudgeCallbackRunningFailedAndAlreadyFinal(t *testing.T) {
	repo := submissionRepoFixture()
	sub := &Submission{
		ID:                uuid.NewString(),
		TaskID:            "task-1",
		UserID:            uuid.NewString(),
		Language:          "go",
		Status:            StatusSentToJudge,
		JudgeSubmissionID: ptrString("judge-sub-state"),
		JudgeStatus:       ptrString(JudgeStatusQueued),
	}
	repo.submissions = map[string]*Submission{sub.ID: sub}
	service := NewService(repo, nil, nil, nil, nil, "", "", 1024, nil)

	startedAt := time.Now().UTC()
	if err := service.HandleJudgeCallback(context.Background(), judge.CallbackPayload{
		JudgeSubmissionID: "judge-sub-state",
		SubmissionID:      sub.ID,
		JudgeStatus:       JudgeStatusRunning,
		StartedAt:         &startedAt,
	}); err != nil {
		t.Fatalf("running callback returned error: %v", err)
	}
	if repo.submissions[sub.ID].Status != StatusRunning || repo.submissions[sub.ID].StartedAt == nil {
		t.Fatalf("running submission = %#v", repo.submissions[sub.ID])
	}

	repo.submissions[sub.ID].Status = StatusSentToJudge
	if err := service.HandleJudgeCallback(context.Background(), judge.CallbackPayload{
		JudgeSubmissionID: "judge-sub-state",
		SubmissionID:      sub.ID,
		JudgeStatus:       JudgeStatusFailed,
		ErrorMessage:      ptrString("boom"),
		Tests:             []judge.CallbackTestResult{{TestCaseID: "case-1", Status: "skipped"}},
	}); err != nil {
		t.Fatalf("failed callback returned error: %v", err)
	}
	if repo.submissions[sub.ID].Status != StatusFailed {
		t.Fatalf("failed submission = %#v", repo.submissions[sub.ID])
	}

	if err := service.HandleJudgeCallback(context.Background(), judge.CallbackPayload{
		JudgeSubmissionID: "judge-sub-state",
		SubmissionID:      sub.ID,
		JudgeStatus:       JudgeStatusFinished,
		Verdict:           ptrString(VerdictAccepted),
	}); err != nil {
		t.Fatalf("already final callback returned error: %v", err)
	}
}

func TestHandleJudgeCallbackMatchApplierError(t *testing.T) {
	matchID := uuid.New()
	subID := uuid.NewString()
	repo := submissionRepoFixture()
	repo.submissions = map[string]*Submission{subID: {
		ID:                subID,
		TaskID:            "task-1",
		UserID:            uuid.NewString(),
		Language:          "go",
		MatchID:           ptrString(matchID.String()),
		Status:            StatusSentToJudge,
		JudgeSubmissionID: ptrString("judge-match-sub"),
	}}
	applierErr := errors.New("apply failed")
	service := NewService(repo, nil, nil, nil, fakeMatchApplier{err: applierErr}, "", "", 1024, nil)

	err := service.HandleJudgeCallback(context.Background(), judge.CallbackPayload{
		JudgeSubmissionID: "judge-match-sub",
		SubmissionID:      subID,
		JudgeStatus:       JudgeStatusFailed,
		Tests:             []judge.CallbackTestResult{{TestCaseID: "case-1", Status: VerdictRuntimeError}},
	})
	if err == nil || !errors.Is(err, applierErr) {
		t.Fatalf("match applier error = %v, want %v", err, applierErr)
	}
}

func TestHandleCodeRunJudgeCallbackFailedAndAlreadyFinal(t *testing.T) {
	repo, run := codeRunCallbackRepo()
	service := NewService(repo, nil, nil, nil, nil, "", "", 1024, nil)

	if err := service.HandleCodeRunJudgeCallback(context.Background(), judge.CallbackPayload{
		JudgeSubmissionID: "judge-run-1",
		SubmissionID:      run.ID,
		JudgeStatus:       JudgeStatusFailed,
		Tests: []judge.CallbackTestResult{
			{TestCaseID: customCodeRunTestID, Status: VerdictRuntimeError},
			{TestCaseID: "sample-1", Status: VerdictAccepted},
		},
	}); err != nil {
		t.Fatalf("failed code run callback returned error: %v", err)
	}
	if repo.runs[run.ID].Status != StatusFailed {
		t.Fatalf("failed code run = %#v", repo.runs[run.ID])
	}

	if err := service.HandleCodeRunJudgeCallback(context.Background(), judge.CallbackPayload{
		JudgeSubmissionID: "judge-run-1",
		SubmissionID:      run.ID,
		JudgeStatus:       JudgeStatusFinished,
		Verdict:           ptrString(VerdictAccepted),
	}); err != nil {
		t.Fatalf("already final code run callback returned error: %v", err)
	}
}

type fakeMatchApplier struct {
	err error
}

func (a fakeMatchApplier) ApplySubmissionResult(context.Context, uuid.UUID) error {
	return a.err
}

func codeRunCallbackRepo() (*fakeCodeRunRepo, *CodeRun) {
	run := &CodeRun{
		ID:                uuid.NewString(),
		TaskID:            uuid.NewString(),
		UserID:            uuid.NewString(),
		Language:          "go",
		LanguageVersion:   "1.24",
		Status:            StatusSentToJudge,
		JudgeSubmissionID: ptrString("judge-run-1"),
		JudgeStatus:       ptrString(JudgeStatusQueued),
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	repo := &fakeCodeRunRepo{
		runs: map[string]*CodeRun{run.ID: run},
		sampleTests: []JudgeTaskTestCase{
			{ID: "sample-1", OrderNo: 1, InputBlobRef: "in", ExpectedBlobRef: "out", Weight: 1},
		},
	}
	return repo, run
}

func ptrString(value string) *string {
	return &value
}
