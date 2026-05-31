package submissions

import (
	"context"
	"errors"
	"strings"
	"testing"

	"code-fight-backend/internal/judge"
	"code-fight-backend/internal/matches"

	"github.com/google/uuid"
)

func TestCreateTaskSubmissionValidationFailures(t *testing.T) {
	taskID := uuid.New()
	userID := uuid.New()
	tests := []struct {
		name        string
		taskRepo    *fakeTaskRepo
		judgeClient *fakeJudgeClient
		input       CreateSubmissionRequest
		want        error
	}{
		{name: "task not found", taskRepo: &fakeTaskRepo{}, input: validSubmissionRequest(), want: ErrTaskNotFound},
		{name: "task inactive", taskRepo: &fakeTaskRepo{task: &Task{ID: taskID}}, input: validSubmissionRequest(), want: ErrTaskInactive},
		{name: "empty language", taskRepo: &fakeTaskRepo{task: &Task{ID: taskID, IsActive: true}, allowed: true}, input: CreateSubmissionRequest{SourceCode: "package main"}, want: ErrTaskLanguageNotAllowed},
		{name: "empty source", taskRepo: &fakeTaskRepo{task: &Task{ID: taskID, IsActive: true}, allowed: true}, input: CreateSubmissionRequest{Language: "go", LanguageVersion: "1.24", SourceCode: " "}, want: ErrInvalidSourceCode},
		{name: "language denied by task", taskRepo: &fakeTaskRepo{task: &Task{ID: taskID, IsActive: true}}, input: validSubmissionRequest(), want: ErrTaskLanguageNotAllowed},
		{name: "language denied by judge", taskRepo: &fakeTaskRepo{task: &Task{ID: taskID, IsActive: true}, allowed: true}, judgeClient: &fakeJudgeClient{}, input: validSubmissionRequest(), want: ErrTaskLanguageNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			judgeClient := tt.judgeClient
			if judgeClient == nil {
				judgeClient = &fakeJudgeClient{supported: []judge.SupportedLanguage{{Language: "go", Version: "1.24"}}}
			}
			service := NewService(submissionRepoFixture(), tt.taskRepo, nil, judgeClient, nil, "", "", 16, nil)

			_, err := service.CreateTaskSubmission(context.Background(), userID, taskID, tt.input)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCreateTaskSubmissionDependencyErrors(t *testing.T) {
	taskID := uuid.New()
	userID := uuid.New()
	depErr := errors.New("dependency failed")
	validTask := &fakeTaskRepo{task: &Task{ID: taskID, IsActive: true}, allowed: true}
	supportedJudge := &fakeJudgeClient{
		supported:      []judge.SupportedLanguage{{Language: "go", Version: "1.24"}},
		submitResponse: &judge.SubmitResponse{JudgeSubmissionID: "judge-sub", Status: judge.StatusQueued},
	}

	tests := []struct {
		name   string
		repo   *fakeCodeRunRepo
		tasks  *fakeTaskRepo
		judge  *fakeJudgeClient
		assert func(error) bool
	}{
		{name: "task get", repo: submissionRepoFixture(), tasks: &fakeTaskRepo{getErr: depErr}, judge: supportedJudge, assert: func(err error) bool { return errors.Is(err, depErr) }},
		{name: "language allowed", repo: submissionRepoFixture(), tasks: &fakeTaskRepo{task: &Task{ID: taskID, IsActive: true}, allowedErr: depErr}, judge: supportedJudge, assert: func(err error) bool { return errors.Is(err, depErr) }},
		{name: "judge supported", repo: submissionRepoFixture(), tasks: validTask, judge: &fakeJudgeClient{supportedErr: depErr}, assert: func(err error) bool { return errors.Is(err, ErrJudgeSubmitFailed) }},
		{name: "create", repo: &fakeCodeRunRepo{createErr: depErr}, tasks: validTask, judge: supportedJudge, assert: func(err error) bool { return errors.Is(err, depErr) }},
		{name: "limits", repo: &fakeCodeRunRepo{limitsErr: depErr}, tasks: validTask, judge: supportedJudge, assert: func(err error) bool { return strings.Contains(err.Error(), "get judge task limits") }},
		{name: "tests", repo: &fakeCodeRunRepo{limits: &JudgeTaskLimits{}, allTestsErr: depErr}, tasks: validTask, judge: supportedJudge, assert: func(err error) bool { return strings.Contains(err.Error(), "list judge task test cases") }},
		{name: "empty tests", repo: &fakeCodeRunRepo{limits: &JudgeTaskLimits{}}, tasks: validTask, judge: supportedJudge, assert: func(err error) bool { return strings.Contains(err.Error(), "task has no active test cases") }},
		{name: "judge submit", repo: submissionRepoFixture(), tasks: validTask, judge: &fakeJudgeClient{supported: supportedJudge.supported, submitErr: depErr}, assert: func(err error) bool { return errors.Is(err, ErrJudgeSubmitFailed) }},
		{name: "mark sent", repo: &fakeCodeRunRepo{limits: &JudgeTaskLimits{}, allTests: []JudgeTaskTestCase{{ID: "case-1"}}, markErr: depErr}, tasks: validTask, judge: supportedJudge, assert: func(err error) bool { return errors.Is(err, depErr) }},
		{name: "reload", repo: &fakeCodeRunRepo{limits: &JudgeTaskLimits{}, allTests: []JudgeTaskTestCase{{ID: "case-1"}}, getErr: depErr}, tasks: validTask, judge: supportedJudge, assert: func(err error) bool { return errors.Is(err, depErr) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(tt.repo, tt.tasks, nil, tt.judge, nil, "", "", 1024, nil)
			_, err := service.CreateTaskSubmission(context.Background(), userID, taskID, validSubmissionRequest())
			if err == nil || !tt.assert(err) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestCreateMatchSubmissionValidationFailures(t *testing.T) {
	taskID := uuid.New()
	matchID := uuid.New()
	userID := uuid.New()
	req := validSubmissionRequest()
	taskRepo := &fakeTaskRepo{allowed: true}
	judgeClient := &fakeJudgeClient{supported: []judge.SupportedLanguage{{Language: "go", Version: "1.24"}}}

	tests := []struct {
		name  string
		match matches.Match
		user  uuid.UUID
		want  error
	}{
		{name: "match not found", match: matches.Match{}, user: userID, want: ErrMatchNotFound},
		{name: "not running", match: matches.Match{ID: matchID, Status: matches.MatchStatusPending}, user: userID, want: ErrMatchNotStarted},
		{name: "task not assigned", match: matches.Match{ID: matchID, Status: matches.MatchStatusRunning, Player1ID: userID}, user: userID, want: ErrMatchTaskNotAssigned},
		{name: "non participant", match: matches.Match{ID: matchID, Status: matches.MatchStatusRunning, TaskID: &taskID, Player1ID: uuid.New(), Player2ID: uuid.New()}, user: userID, want: ErrUserNotMatchParticipant},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(submissionRepoFixture(), taskRepo, &fakeMatchRepo{match: tt.match}, judgeClient, nil, "", "", 1024, nil)
			_, err := service.CreateMatchSubmission(context.Background(), tt.user, matchID, req)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCreateMatchSubmissionDependencyErrors(t *testing.T) {
	taskID := uuid.New()
	matchID := uuid.New()
	userID := uuid.New()
	depErr := errors.New("dependency failed")
	match := matches.Match{
		ID:        matchID,
		TaskID:    &taskID,
		Player1ID: userID,
		Player2ID: uuid.New(),
		Status:    matches.MatchStatusRunning,
	}
	service := NewService(
		submissionRepoFixture(),
		&fakeTaskRepo{allowed: true, allowedErr: depErr},
		&fakeMatchRepo{match: match},
		&fakeJudgeClient{supported: []judge.SupportedLanguage{{Language: "go", Version: "1.24"}}},
		nil,
		"",
		"",
		1024,
		nil,
	)
	if _, err := service.CreateMatchSubmission(context.Background(), userID, matchID, validSubmissionRequest()); !errors.Is(err, depErr) {
		t.Fatalf("language error = %v, want %v", err, depErr)
	}

	service = NewService(submissionRepoFixture(), &fakeTaskRepo{allowed: true}, &fakeMatchRepo{err: depErr}, nil, nil, "", "", 1024, nil)
	if _, err := service.CreateMatchSubmission(context.Background(), userID, matchID, validSubmissionRequest()); !errors.Is(err, depErr) {
		t.Fatalf("match repo error = %v, want %v", err, depErr)
	}
}

func TestCreateTaskCodeRunValidationFailures(t *testing.T) {
	taskID := uuid.New()
	userID := uuid.New()
	tests := []struct {
		name        string
		repo        *fakeCodeRunRepo
		taskRepo    *fakeTaskRepo
		judgeClient *fakeJudgeClient
		input       CreateCodeRunRequest
		want        error
	}{
		{name: "task not found", repo: submissionRepoFixture(), taskRepo: &fakeTaskRepo{}, input: validCodeRunRequest(), want: ErrTaskNotFound},
		{name: "task inactive", repo: submissionRepoFixture(), taskRepo: &fakeTaskRepo{task: &Task{ID: taskID}}, input: validCodeRunRequest(), want: ErrTaskInactive},
		{name: "custom too large", repo: submissionRepoFixture(), taskRepo: &fakeTaskRepo{task: &Task{ID: taskID, IsActive: true}, allowed: true}, input: CreateCodeRunRequest{
			Language:        "go",
			LanguageVersion: "1.24",
			SourceCode:      "package main",
			CustomTest:      &CodeRunCustomTestRequest{Input: strings.Repeat("x", 32)},
		}, want: ErrInvalidSourceCode},
		{name: "language denied", repo: submissionRepoFixture(), taskRepo: &fakeTaskRepo{task: &Task{ID: taskID, IsActive: true}}, input: validCodeRunRequest(), want: ErrTaskLanguageNotAllowed},
		{name: "judge unsupported", repo: submissionRepoFixture(), taskRepo: &fakeTaskRepo{task: &Task{ID: taskID, IsActive: true}, allowed: true}, judgeClient: &fakeJudgeClient{}, input: validCodeRunRequest(), want: ErrTaskLanguageNotAllowed},
		{name: "no sample tests", repo: &fakeCodeRunRepo{limits: &JudgeTaskLimits{}}, taskRepo: &fakeTaskRepo{task: &Task{ID: taskID, IsActive: true}, allowed: true}, input: validCodeRunRequest(), want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			judgeClient := tt.judgeClient
			if judgeClient == nil {
				judgeClient = &fakeJudgeClient{supported: []judge.SupportedLanguage{{Language: "go", Version: "1.24"}}}
			}
			service := NewService(tt.repo, tt.taskRepo, nil, judgeClient, nil, "", "", 16, nil)

			_, err := service.CreateTaskCodeRun(context.Background(), userID, taskID, tt.input)
			if tt.want == nil {
				if err == nil || !strings.Contains(err.Error(), "task has no active sample test cases") {
					t.Fatalf("error = %v, want no sample test cases", err)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCreateTaskCodeRunDependencyErrors(t *testing.T) {
	taskID := uuid.New()
	userID := uuid.New()
	depErr := errors.New("dependency failed")
	validTask := &fakeTaskRepo{task: &Task{ID: taskID, IsActive: true}, allowed: true}
	supportedJudge := &fakeJudgeClient{
		supported:      []judge.SupportedLanguage{{Language: "go", Version: "1.24"}},
		submitResponse: &judge.SubmitResponse{JudgeSubmissionID: "judge-run", Status: judge.StatusQueued},
	}

	tests := []struct {
		name   string
		repo   *fakeCodeRunRepo
		tasks  *fakeTaskRepo
		judge  *fakeJudgeClient
		assert func(error) bool
	}{
		{name: "task get", repo: submissionRepoFixture(), tasks: &fakeTaskRepo{getErr: depErr}, judge: supportedJudge, assert: func(err error) bool { return errors.Is(err, depErr) }},
		{name: "language allowed", repo: submissionRepoFixture(), tasks: &fakeTaskRepo{task: &Task{ID: taskID, IsActive: true}, allowedErr: depErr}, judge: supportedJudge, assert: func(err error) bool { return errors.Is(err, depErr) }},
		{name: "judge supported", repo: submissionRepoFixture(), tasks: validTask, judge: &fakeJudgeClient{supportedErr: depErr}, assert: func(err error) bool { return errors.Is(err, ErrJudgeSubmitFailed) }},
		{name: "create run", repo: &fakeCodeRunRepo{codeRunErr: depErr}, tasks: validTask, judge: supportedJudge, assert: func(err error) bool { return errors.Is(err, depErr) }},
		{name: "limits", repo: &fakeCodeRunRepo{limitsErr: depErr}, tasks: validTask, judge: supportedJudge, assert: func(err error) bool { return strings.Contains(err.Error(), "get judge task limits") }},
		{name: "samples", repo: &fakeCodeRunRepo{limits: &JudgeTaskLimits{}, samplesErr: depErr}, tasks: validTask, judge: supportedJudge, assert: func(err error) bool { return strings.Contains(err.Error(), "list judge task sample test cases") }},
		{name: "judge submit", repo: submissionRepoFixture(), tasks: validTask, judge: &fakeJudgeClient{supported: supportedJudge.supported, submitErr: depErr}, assert: func(err error) bool { return errors.Is(err, ErrJudgeSubmitFailed) }},
		{name: "mark run", repo: &fakeCodeRunRepo{limits: &JudgeTaskLimits{}, sampleTests: []JudgeTaskTestCase{{ID: "case-1"}}, markRunErr: depErr}, tasks: validTask, judge: supportedJudge, assert: func(err error) bool { return errors.Is(err, depErr) }},
		{name: "reload run", repo: &fakeCodeRunRepo{limits: &JudgeTaskLimits{}, sampleTests: []JudgeTaskTestCase{{ID: "case-1"}}, getRunErr: depErr}, tasks: validTask, judge: supportedJudge, assert: func(err error) bool { return errors.Is(err, depErr) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(tt.repo, tt.tasks, nil, tt.judge, nil, "", "", 1024, nil)
			_, err := service.CreateTaskCodeRun(context.Background(), userID, taskID, validCodeRunRequest())
			if err == nil || !tt.assert(err) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestGetCodeRunByIDForUser(t *testing.T) {
	userID := uuid.New()
	otherID := uuid.New()
	repo := submissionRepoFixture()
	repo.runs = map[string]*CodeRun{
		"run-1": {ID: "run-1", UserID: userID.String()},
	}
	service := NewService(repo, nil, nil, nil, nil, "", "", 1024, nil)

	if _, _, err := service.GetCodeRunByIDForUser(context.Background(), "run-1", userID); err != nil {
		t.Fatalf("owner GetCodeRunByIDForUser returned error: %v", err)
	}
	if _, _, err := service.GetCodeRunByIDForUser(context.Background(), "run-1", otherID); !errors.Is(err, ErrSubmissionNotFound) {
		t.Fatalf("other user error = %v", err)
	}
}

func TestSubmissionAndCodeRunPublishing(t *testing.T) {
	userID := uuid.New()
	matchID := uuid.New()
	player2 := uuid.New()
	notifier := &fakeSubmissionNotifier{}
	service := NewService(submissionRepoFixture(), nil, &fakeMatchRepo{match: matches.Match{
		ID:        matchID,
		Player1ID: userID,
		Player2ID: player2,
	}}, nil, nil, "", "", 1024, nil)
	service.SetNotifier(notifier)

	service.publishSubmission(context.Background(), "submission.created", &Submission{
		ID:     "sub-1",
		UserID: userID.String(),
	}, nil)
	if notifier.singleEvent != "submission.created" || notifier.singleUser != userID {
		t.Fatalf("single publish = %#v", notifier)
	}

	notifier = &fakeSubmissionNotifier{}
	service.SetNotifier(notifier)
	service.publishSubmission(context.Background(), "submission.finished", &Submission{
		ID:      "sub-2",
		UserID:  userID.String(),
		MatchID: ptrString(matchID.String()),
	}, nil)
	if notifier.manyEvent != "submission.finished" || len(notifier.manyUsers) != 2 || notifier.manyUsers[1] != player2 {
		t.Fatalf("many publish = %#v", notifier)
	}

	notifier = &fakeSubmissionNotifier{}
	service.SetNotifier(notifier)
	service.publishCodeRun("code_run.created", &CodeRun{ID: "run-1", UserID: userID.String()}, nil)
	if notifier.singleEvent != "code_run.created" || notifier.singleUser != userID {
		t.Fatalf("code run publish = %#v", notifier)
	}
}

func TestServiceMetricsRecorders(t *testing.T) {
	metrics := &fakeSubmissionMetrics{}
	service := NewService(nil, nil, nil, nil, nil, "", "", 0, metrics)

	service.recordSubmissionCreated("task", " go ", "success")
	service.recordSubmissionFinished("match", " go ", StatusFinished, VerdictAccepted)
	service.recordCodeRunCreated(" go ", "success")
	service.recordCodeRunFinished(" go ", StatusFailed, "none")

	if metrics.submissionCreated != "task/go/success" ||
		metrics.submissionFinished != "match/go/finished/accepted" ||
		metrics.codeRunCreated != "go/success" ||
		metrics.codeRunFinished != "go/failed/none" {
		t.Fatalf("metrics = %#v", metrics)
	}
}

func validSubmissionRequest() CreateSubmissionRequest {
	return CreateSubmissionRequest{
		Language:        "go",
		LanguageVersion: "1.24",
		SourceCode:      "package main",
	}
}

func validCodeRunRequest() CreateCodeRunRequest {
	return CreateCodeRunRequest{
		Language:        "go",
		LanguageVersion: "1.24",
		SourceCode:      "package main",
	}
}

type fakeSubmissionNotifier struct {
	singleUser  uuid.UUID
	singleEvent string
	manyUsers   []uuid.UUID
	manyEvent   string
}

func (n *fakeSubmissionNotifier) Publish(userID uuid.UUID, eventType string, _ any) {
	n.singleUser = userID
	n.singleEvent = eventType
}

func (n *fakeSubmissionNotifier) PublishMany(userIDs []uuid.UUID, eventType string, _ any) {
	n.manyUsers = userIDs
	n.manyEvent = eventType
}

type fakeSubmissionMetrics struct {
	submissionCreated  string
	submissionFinished string
	codeRunCreated     string
	codeRunFinished    string
}

func (m *fakeSubmissionMetrics) RecordSubmissionCreated(scope string, language string, status string) {
	m.submissionCreated = scope + "/" + language + "/" + status
}

func (m *fakeSubmissionMetrics) RecordSubmissionFinished(scope string, language string, status string, verdict string) {
	m.submissionFinished = scope + "/" + language + "/" + status + "/" + verdict
}

func (m *fakeSubmissionMetrics) RecordCodeRunCreated(language string, status string) {
	m.codeRunCreated = language + "/" + status
}

func (m *fakeSubmissionMetrics) RecordCodeRunFinished(language string, status string, verdict string) {
	m.codeRunFinished = language + "/" + status + "/" + verdict
}
