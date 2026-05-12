package submissions

import (
	"context"
	"testing"
	"time"

	"code-fight-backend/internal/judge"

	"github.com/google/uuid"
)

func TestCreateTaskCodeRunSendsOnlySampleTestsToJudge(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()
	userID := uuid.New()

	repo := &fakeCodeRunRepo{
		limits: &JudgeTaskLimits{
			TimeLimitMS:   1500,
			MemoryLimitKB: 131072,
		},
		sampleTests: []JudgeTaskTestCase{
			{
				ID:              uuid.NewString(),
				OrderNo:         1,
				InputBlobRef:    "1 2\n",
				ExpectedBlobRef: "3\n",
				Weight:          1,
			},
		},
		allTests: []JudgeTaskTestCase{
			{
				ID:              uuid.NewString(),
				OrderNo:         99,
				InputBlobRef:    "hidden\n",
				ExpectedBlobRef: "hidden\n",
				Weight:          10,
			},
		},
	}
	taskRepo := &fakeTaskRepo{
		task:    &Task{ID: taskID, IsActive: true},
		allowed: true,
	}
	judgeClient := &fakeJudgeClient{
		supported: []judge.SupportedLanguage{{Language: "go", Version: "1.24"}},
		submitResponse: &judge.SubmitResponse{
			JudgeSubmissionID: "judge-run-1",
			Status:            judge.StatusQueued,
		},
	}

	service := NewService(
		repo,
		taskRepo,
		nil,
		judgeClient,
		nil,
		"http://backend/internal/judge/callback",
		"callback-token",
		1024*1024,
	)

	run, err := service.CreateTaskCodeRun(ctx, userID, taskID, CreateCodeRunRequest{
		Language:        "go",
		LanguageVersion: "1.24",
		SourceCode:      "package main\nfunc main() {}\n",
	})
	if err != nil {
		t.Fatalf("CreateTaskCodeRun returned error: %v", err)
	}

	if run.Status != StatusSentToJudge {
		t.Fatalf("run status = %q, want %q", run.Status, StatusSentToJudge)
	}
	if run.JudgeSubmissionID == nil || *run.JudgeSubmissionID != "judge-run-1" {
		t.Fatalf("judge submission id = %v, want judge-run-1", run.JudgeSubmissionID)
	}

	if len(judgeClient.submits) != 1 {
		t.Fatalf("judge submits = %d, want 1", len(judgeClient.submits))
	}

	submit := judgeClient.submits[0]
	if submit.SubmissionID != run.ID {
		t.Fatalf("judge submission_id = %q, want run id %q", submit.SubmissionID, run.ID)
	}
	if submit.TaskID != taskID.String() {
		t.Fatalf("judge task_id = %q, want %q", submit.TaskID, taskID.String())
	}
	if got, want := submit.Limits.TimeMS, 1500; got != want {
		t.Fatalf("judge time limit = %d, want %d", got, want)
	}
	if got, want := submit.Limits.MemoryMB, 128; got != want {
		t.Fatalf("judge memory limit = %d, want %d", got, want)
	}
	if len(submit.Tests) != len(repo.sampleTests) {
		t.Fatalf("judge tests = %d, want %d sample test", len(submit.Tests), len(repo.sampleTests))
	}
	if submit.Tests[0].ID != repo.sampleTests[0].ID {
		t.Fatalf("judge test id = %q, want sample id %q", submit.Tests[0].ID, repo.sampleTests[0].ID)
	}
}

func TestCreateTaskCodeRunCombinesSampleAndCustomTest(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()
	userID := uuid.New()
	expectedOutput := "custom out\n"

	repo := &fakeCodeRunRepo{
		limits: &JudgeTaskLimits{
			TimeLimitMS:   1500,
			MemoryLimitKB: 131072,
		},
		sampleTests: []JudgeTaskTestCase{
			{
				ID:              uuid.NewString(),
				OrderNo:         1,
				InputBlobRef:    "sample in\n",
				ExpectedBlobRef: "sample out\n",
				Weight:          1,
			},
		},
	}
	taskRepo := &fakeTaskRepo{
		task:    &Task{ID: taskID, IsActive: true},
		allowed: true,
	}
	judgeClient := &fakeJudgeClient{
		supported: []judge.SupportedLanguage{{Language: "go", Version: "1.24"}},
		submitResponse: &judge.SubmitResponse{
			JudgeSubmissionID: "judge-run-2",
			Status:            judge.StatusQueued,
		},
	}

	service := NewService(
		repo,
		taskRepo,
		nil,
		judgeClient,
		nil,
		"http://backend/internal/judge/callback",
		"callback-token",
		1024*1024,
	)

	_, err := service.CreateTaskCodeRun(ctx, userID, taskID, CreateCodeRunRequest{
		Language:        "go",
		LanguageVersion: "1.24",
		SourceCode:      "package main\nfunc main() {}\n",
		CustomTest: &CodeRunCustomTestRequest{
			Input:          "custom in\n",
			ExpectedOutput: &expectedOutput,
		},
	})
	if err != nil {
		t.Fatalf("CreateTaskCodeRun returned error: %v", err)
	}

	if len(judgeClient.submits) != 1 {
		t.Fatalf("judge submits = %d, want 1", len(judgeClient.submits))
	}

	tests := judgeClient.submits[0].Tests
	if len(tests) != 2 {
		t.Fatalf("judge tests = %d, want sample + custom", len(tests))
	}
	if tests[0].ID != repo.sampleTests[0].ID {
		t.Fatalf("first judge test id = %q, want sample id %q", tests[0].ID, repo.sampleTests[0].ID)
	}
	if tests[1].ID != customCodeRunTestID {
		t.Fatalf("second judge test id = %q, want custom", tests[1].ID)
	}
	if tests[1].Input != "custom in\n" {
		t.Fatalf("custom input = %q", tests[1].Input)
	}
	if tests[1].ExpectedOutput != expectedOutput {
		t.Fatalf("custom expected output = %q", tests[1].ExpectedOutput)
	}
	if tests[1].SkipOutputCheck {
		t.Fatalf("custom test with expected output should not skip output check")
	}
}

type fakeTaskRepo struct {
	task    *Task
	allowed bool
}

func (r *fakeTaskRepo) GetByID(context.Context, uuid.UUID) (*Task, error) {
	return r.task, nil
}

func (r *fakeTaskRepo) IsLanguageAllowed(context.Context, uuid.UUID, string, string) (bool, error) {
	return r.allowed, nil
}

type fakeJudgeClient struct {
	supported      []judge.SupportedLanguage
	submitResponse *judge.SubmitResponse
	submits        []judge.SubmitRequest
}

func (c *fakeJudgeClient) Submit(_ context.Context, req judge.SubmitRequest) (*judge.SubmitResponse, error) {
	c.submits = append(c.submits, req)
	return c.submitResponse, nil
}

func (c *fakeJudgeClient) SupportedLanguages(context.Context) ([]judge.SupportedLanguage, error) {
	return c.supported, nil
}

type fakeCodeRunRepo struct {
	runs        map[string]*CodeRun
	limits      *JudgeTaskLimits
	sampleTests []JudgeTaskTestCase
	allTests    []JudgeTaskTestCase
}

func (r *fakeCodeRunRepo) Create(_ context.Context, _ *Submission) error {
	panic("unexpected Create call")
}

func (r *fakeCodeRunRepo) GetByID(context.Context, string) (*Submission, []SubmissionTestResult, error) {
	panic("unexpected GetByID call")
}

func (r *fakeCodeRunRepo) GetByJudgeSubmissionID(context.Context, string) (*Submission, error) {
	return nil, ErrSubmissionNotFound
}

func (r *fakeCodeRunRepo) MarkSentToJudge(context.Context, string, string, string) error {
	panic("unexpected MarkSentToJudge call")
}

func (r *fakeCodeRunRepo) MarkRunning(context.Context, string, string, *time.Time) error {
	panic("unexpected MarkRunning call")
}

func (r *fakeCodeRunRepo) Complete(context.Context, string, CompleteInput, []SubmissionTestResult) error {
	panic("unexpected Complete call")
}

func (r *fakeCodeRunRepo) CreateCodeRun(_ context.Context, run *CodeRun) error {
	if r.runs == nil {
		r.runs = make(map[string]*CodeRun)
	}
	copied := *run
	r.runs[run.ID] = &copied
	return nil
}

func (r *fakeCodeRunRepo) GetCodeRunByID(_ context.Context, id string) (*CodeRun, []CodeRunTestResult, error) {
	run, ok := r.runs[id]
	if !ok {
		return nil, nil, ErrSubmissionNotFound
	}
	copied := *run
	return &copied, nil, nil
}

func (r *fakeCodeRunRepo) GetCodeRunByJudgeSubmissionID(_ context.Context, judgeSubmissionID string) (*CodeRun, error) {
	for _, run := range r.runs {
		if run.JudgeSubmissionID != nil && *run.JudgeSubmissionID == judgeSubmissionID {
			copied := *run
			return &copied, nil
		}
	}
	return nil, ErrSubmissionNotFound
}

func (r *fakeCodeRunRepo) MarkCodeRunSentToJudge(_ context.Context, codeRunID, judgeSubmissionID, judgeStatus string) error {
	run, ok := r.runs[codeRunID]
	if !ok {
		return ErrSubmissionNotFound
	}
	run.Status = StatusSentToJudge
	run.JudgeSubmissionID = &judgeSubmissionID
	run.JudgeStatus = &judgeStatus
	return nil
}

func (r *fakeCodeRunRepo) MarkCodeRunRunning(_ context.Context, codeRunID, judgeStatus string, startedAt *time.Time) error {
	run, ok := r.runs[codeRunID]
	if !ok {
		return ErrSubmissionNotFound
	}
	run.Status = StatusRunning
	run.JudgeStatus = &judgeStatus
	run.StartedAt = startedAt
	return nil
}

func (r *fakeCodeRunRepo) CompleteCodeRun(_ context.Context, codeRunID string, input CompleteInput, _ []CodeRunTestResult) error {
	run, ok := r.runs[codeRunID]
	if !ok {
		return ErrSubmissionNotFound
	}
	run.Status = input.Status
	run.JudgeStatus = &input.JudgeStatus
	run.Verdict = input.Verdict
	run.ErrorMessage = input.ErrorMessage
	run.PassedTests = input.PassedTests
	run.TotalTests = input.TotalTests
	run.Score = input.Score
	return nil
}

func (r *fakeCodeRunRepo) GetJudgeTaskLimits(context.Context, string) (*JudgeTaskLimits, error) {
	return r.limits, nil
}

func (r *fakeCodeRunRepo) ListJudgeTaskTestCases(context.Context, string) ([]JudgeTaskTestCase, error) {
	return r.allTests, nil
}

func (r *fakeCodeRunRepo) ListJudgeTaskSampleTestCases(context.Context, string) ([]JudgeTaskTestCase, error) {
	return r.sampleTests, nil
}
