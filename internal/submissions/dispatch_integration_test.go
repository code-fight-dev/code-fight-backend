package submissions

import (
	"context"
	"io"
	"testing"
	"time"

	"code-fight-backend/internal/judge"
	"code-fight-backend/internal/testsupport"
)

type dispatchTestClient struct {
	judge.Client
	submit func(context.Context, judge.SubmitRequest) (*judge.SubmitResponse, error)
}

func (c dispatchTestClient) Submit(ctx context.Context, request judge.SubmitRequest) (*judge.SubmitResponse, error) {
	return c.submit(ctx, request)
}

func TestDurableDispatchRetriesWithSameID(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	repo := NewPGRepository(db)
	userID := createSubmissionTestUser(t, db, ctx)
	taskID, _, _ := createSubmissionTestTask(t, db, ctx)
	now := time.Now().UTC()
	service := &Service{
		repo: repo, taskRepo: NewPGTaskRepository(db), maxSourceBytes: 65536,
		callbackURL: "http://backend/callback", callbackToken: "integration-token",
	}
	sub, err := service.CreateTaskSubmission(ctx, userID, taskID, CreateSubmissionRequest{
		Language: "go", LanguageVersion: "1.24", SourceCode: "package main",
	})
	if err != nil || sub.Status != StatusQueued {
		t.Fatalf("submit while Judge is unavailable = %#v, %v", sub, err)
	}

	var payload judge.SubmitRequest
	if err := db.QueryRow(ctx, "select payload from judge_dispatch where submission_id=$1", sub.ID).Scan(&payload); err != nil {
		t.Fatalf("read persisted dispatch: %v", err)
	}
	if payload.SubmissionID != sub.ID || payload.SourceCode != sub.SourceCode || len(payload.Tests) != 2 ||
		payload.CallbackToken != "" || payload.CallbackURL != "" {
		t.Fatalf("unexpected persisted dispatch: %#v", payload)
	}

	attempts := 0
	service.judgeClient = dispatchTestClient{submit: func(ctx context.Context, request judge.SubmitRequest) (*judge.SubmitResponse, error) {
		attempts++
		if request.SubmissionID != sub.ID || !request.DurableDispatch || request.CallbackToken != service.callbackToken {
			t.Fatalf("unexpected request on attempt %d: %#v", attempts, request)
		}
		if attempts == 1 {
			return nil, io.ErrUnexpectedEOF
		}
		// A worker can report progress before the retry acknowledgement arrives.
		if err := repo.MarkRunning(ctx, sub.ID, JudgeStatusRunning, &now); err != nil {
			t.Fatalf("mark running before acknowledgement: %v", err)
		}
		return &judge.SubmitResponse{JudgeSubmissionID: sub.ID, Status: JudgeStatusQueued}, nil
	}}
	if processed, err := service.dispatchNext(ctx, repo); !processed || err == nil {
		t.Fatalf("first dispatch = %t, %v", processed, err)
	}
	queued, err := repo.GetByJudgeSubmissionID(ctx, sub.ID)
	if err != nil || queued.Status != StatusQueued {
		t.Fatalf("queued submission = %#v, %v", queued, err)
	}
	if _, err := db.Exec(ctx, "update judge_dispatch set next_attempt_at=now() where id=$1 and attempts=1", sub.ID); err != nil {
		t.Fatalf("make retry due: %v", err)
	}
	if processed, err := service.dispatchNext(ctx, repo); !processed || err != nil {
		t.Fatalf("retry dispatch = %t, %v", processed, err)
	}
	if processed, err := service.dispatchNext(ctx, repo); processed || err != nil || attempts != 2 {
		t.Fatalf("dispatch after acknowledgement = %t, %v, attempts=%d", processed, err, attempts)
	}
	running, err := repo.GetByJudgeSubmissionID(ctx, sub.ID)
	if err != nil || running.Status != StatusRunning {
		t.Fatalf("submission after acknowledgement = %#v, %v", running, err)
	}
}
