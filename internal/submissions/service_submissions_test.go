package submissions

import (
	"context"
	"testing"
	"time"

	"code-fight-backend/internal/judge"
	"code-fight-backend/internal/matches"

	"github.com/google/uuid"
)

func TestCreateTaskSubmissionSendsAllTestsToJudge(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()
	userID := uuid.New()
	repo := submissionRepoFixture()
	taskRepo := &fakeTaskRepo{task: &Task{ID: taskID, IsActive: true}, allowed: true}
	judgeClient := &fakeJudgeClient{
		supported: []judge.SupportedLanguage{{Language: "go", Version: "1.24"}},
		submitResponse: &judge.SubmitResponse{
			JudgeSubmissionID: "judge-sub-1",
			Status:            judge.StatusQueued,
		},
	}
	service := NewService(repo, taskRepo, nil, judgeClient, nil, "callback", "token", 1024, nil)

	sub, err := service.CreateTaskSubmission(ctx, userID, taskID, CreateSubmissionRequest{
		Language:        "go",
		LanguageVersion: "1.24",
		SourceCode:      "package main\nfunc main() {}\n",
	})
	if err != nil {
		t.Fatalf("CreateTaskSubmission returned error: %v", err)
	}
	if sub.Status != StatusSentToJudge || sub.JudgeSubmissionID == nil || *sub.JudgeSubmissionID != "judge-sub-1" {
		t.Fatalf("submission = %#v", sub)
	}
	if len(judgeClient.submits) != 1 || len(judgeClient.submits[0].Tests) != len(repo.allTests) {
		t.Fatalf("judge submits = %#v", judgeClient.submits)
	}
}

func TestCreateMatchSubmissionValidatesParticipantAndSendsToJudge(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()
	matchID := uuid.New()
	player1 := uuid.New()
	player2 := uuid.New()
	repo := submissionRepoFixture()
	taskRepo := &fakeTaskRepo{allowed: true}
	matchRepo := &fakeMatchRepo{match: matches.Match{
		ID:        matchID,
		TaskID:    &taskID,
		Player1ID: player1,
		Player2ID: player2,
		Status:    matches.MatchStatusRunning,
	}}
	judgeClient := &fakeJudgeClient{
		supported:      []judge.SupportedLanguage{{Language: "go", Version: "1.24"}},
		submitResponse: &judge.SubmitResponse{JudgeSubmissionID: "judge-match-1", Status: judge.StatusQueued},
	}
	service := NewService(repo, taskRepo, matchRepo, judgeClient, nil, "callback", "token", 1024, nil)

	sub, err := service.CreateMatchSubmission(ctx, player1, matchID, CreateSubmissionRequest{
		Language:        "go",
		LanguageVersion: "1.24",
		SourceCode:      "package main\nfunc main() {}\n",
	})
	if err != nil {
		t.Fatalf("CreateMatchSubmission returned error: %v", err)
	}
	if sub.MatchID == nil || *sub.MatchID != matchID.String() {
		t.Fatalf("match id = %v", sub.MatchID)
	}

	_, err = service.CreateMatchSubmission(ctx, uuid.New(), matchID, CreateSubmissionRequest{
		Language:        "go",
		LanguageVersion: "1.24",
		SourceCode:      "package main\nfunc main() {}\n",
	})
	if err != ErrUserNotMatchParticipant {
		t.Fatalf("non participant error = %v", err)
	}
}

func TestHandleJudgeCallbackCompletesSubmission(t *testing.T) {
	ctx := context.Background()
	repo := submissionRepoFixture()
	sub := &Submission{
		ID:                uuid.NewString(),
		TaskID:            uuid.NewString(),
		UserID:            uuid.NewString(),
		Language:          "go",
		Status:            StatusSentToJudge,
		JudgeSubmissionID: ptrString("judge-sub-1"),
		JudgeStatus:       ptrString(JudgeStatusQueued),
	}
	repo.submissions = map[string]*Submission{sub.ID: sub}
	service := NewService(repo, nil, nil, nil, nil, "", "", 1024, nil)
	verdict := VerdictAccepted
	finishedAt := time.Now().UTC()

	err := service.HandleJudgeCallback(ctx, judge.CallbackPayload{
		JudgeSubmissionID: "judge-sub-1",
		SubmissionID:      sub.ID,
		JudgeStatus:       JudgeStatusFinished,
		Verdict:           &verdict,
		PassedTests:       1,
		TotalTests:        1,
		Score:             100,
		FinishedAt:        &finishedAt,
		Tests:             []judge.CallbackTestResult{{TestCaseID: "case-1", Status: VerdictAccepted}},
	})
	if err != nil {
		t.Fatalf("HandleJudgeCallback returned error: %v", err)
	}
	updated := repo.submissions[sub.ID]
	if updated.Status != StatusFinished || updated.Verdict == nil || *updated.Verdict != VerdictAccepted {
		t.Fatalf("updated submission = %#v", updated)
	}
}

func TestGetByIDForUserAllowsOwnerAndMatchParticipants(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	matchID := uuid.New()
	repo := submissionRepoFixture()
	repo.submissions = map[string]*Submission{
		"task-sub":  {ID: "task-sub", UserID: ownerID.String()},
		"match-sub": {ID: "match-sub", UserID: ownerID.String(), MatchID: ptrString(matchID.String())},
	}
	service := NewService(repo, nil, &fakeMatchRepo{match: matches.Match{
		ID:        matchID,
		Player1ID: ownerID,
		Player2ID: otherID,
	}}, nil, nil, "", "", 1024, nil)

	if _, _, err := service.GetByIDForUser(context.Background(), "task-sub", ownerID); err != nil {
		t.Fatalf("owner GetByIDForUser returned error: %v", err)
	}
	if _, _, err := service.GetByIDForUser(context.Background(), "match-sub", otherID); err != nil {
		t.Fatalf("participant GetByIDForUser returned error: %v", err)
	}
	if _, _, err := service.GetByIDForUser(context.Background(), "task-sub", otherID); err != ErrSubmissionNotFound {
		t.Fatalf("other user error = %v", err)
	}
}

func submissionRepoFixture() *fakeCodeRunRepo {
	return &fakeCodeRunRepo{
		limits: &JudgeTaskLimits{TimeLimitMS: 1500, MemoryLimitKB: 131072},
		allTests: []JudgeTaskTestCase{
			{ID: "case-1", OrderNo: 1, InputBlobRef: "in", ExpectedBlobRef: "out", Weight: 1},
			{ID: "case-2", OrderNo: 2, InputBlobRef: "in2", ExpectedBlobRef: "out2", Weight: 1},
		},
		sampleTests: []JudgeTaskTestCase{
			{ID: "case-1", OrderNo: 1, InputBlobRef: "in", ExpectedBlobRef: "out", Weight: 1},
		},
	}
}

type fakeMatchRepo struct {
	match matches.Match
	err   error
}

func (r *fakeMatchRepo) GetMatch(context.Context, uuid.UUID) (matches.Match, error) {
	if r.err != nil {
		return matches.Match{}, r.err
	}
	return r.match, nil
}
