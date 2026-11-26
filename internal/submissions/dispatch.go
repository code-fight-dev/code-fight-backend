package submissions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"code-fight-backend/internal/judge"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type dispatchRepository interface {
	createSubmission(context.Context, *Submission, []byte) error
	createCodeRun(context.Context, *CodeRun, []byte) error
}

func (s *Service) persistSubmission(ctx context.Context, submission *Submission) error {
	if repo, ok := s.repo.(dispatchRepository); ok {
		request, err := s.submissionDispatch(ctx, submission)
		if err != nil {
			return err
		}
		payload, err := marshalDispatch(request)
		if err != nil {
			return err
		}
		submission.JudgeSubmissionID = &submission.ID
		return repo.createSubmission(ctx, submission, payload)
	}
	return s.sendToJudge(ctx, submission)
}

func (s *Service) persistCodeRun(ctx context.Context, run *CodeRun, input CreateCodeRunRequest) error {
	if repo, ok := s.repo.(dispatchRepository); ok {
		request, err := s.codeRunDispatch(ctx, run, input)
		if err != nil {
			return err
		}
		payload, err := marshalDispatch(request)
		if err != nil {
			return err
		}
		run.JudgeSubmissionID = &run.ID
		return repo.createCodeRun(ctx, run, payload)
	}
	return s.sendCodeRunToJudge(ctx, run, input)
}

func marshalDispatch(request judge.SubmitRequest) ([]byte, error) {
	// Credentials come from current configuration on every attempt.
	request.CallbackURL = ""
	request.CallbackToken = ""
	return json.Marshal(request)
}

func (s *Service) RunDispatch(ctx context.Context, repo *PGRepository, timeout time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for ctx.Err() == nil {
		attemptCtx, cancel := context.WithTimeout(ctx, timeout+5*time.Second)
		processed, err := s.dispatchNext(attemptCtx, repo)
		cancel()
		if err != nil && ctx.Err() == nil {
			logger.Warn("judge dispatch failed", "error", err)
		}
		if processed && err == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) dispatchNext(ctx context.Context, repo *PGRepository) (bool, error) {
	tx, err := repo.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	var id string
	var codeRun bool
	var request judge.SubmitRequest
	err = tx.QueryRow(ctx, `select id, code_run_id is not null, payload from judge_dispatch
		where next_attempt_at <= now() order by next_attempt_at,id limit 1 for update skip locked`).
		Scan(&id, &codeRun, &request)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	request.CallbackURL = s.callbackURL
	request.CallbackToken = s.callbackToken
	request.DurableDispatch = true
	response, failure := s.judgeClient.Submit(ctx, request)
	if failure == nil && (response == nil || response.JudgeSubmissionID != id) {
		failure = errors.New("judge returned a different job ID")
	}
	var rejection *judge.SubmitHTTPError
	permanent := errors.As(failure, &rejection) &&
		(rejection.Code == "validation_error" || rejection.Status == http.StatusRequestEntityTooLarge)
	if failure != nil && !permanent {
		_, err = tx.Exec(ctx, `update judge_dispatch
			set attempts=least(attempts+1,6),
			    next_attempt_at=clock_timestamp()+make_interval(secs => power(2,least(attempts,5)))
			where id=$1`, id)
		if err == nil {
			err = tx.Commit(ctx)
		}
		if err != nil {
			return false, err
		}
		return true, fmt.Errorf("retrying execution %s: %w", id, failure)
	}

	table := "submissions"
	if codeRun {
		table = "code_runs"
	}
	// Callbacks can complete before this acknowledgement. The queued guard
	// preserves any progress or result already saved by the callback.
	var updated pgconn.CommandTag
	if permanent {
		updated, err = tx.Exec(ctx, `update `+table+` set status='failed', judge_status='failed',
			error_message='Judge rejected this execution request', finished_at=now(), updated_at=now()
			where id=$1 and status='queued'`, id)
	} else {
		_, err = tx.Exec(ctx, `update `+table+` set status='sent_to_judge', judge_status='queued', updated_at=now()
			where id=$1 and status='queued'`, id)
	}
	if err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, "delete from judge_dispatch where id=$1", id); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	if permanent && updated.RowsAffected() != 0 {
		s.publishDispatchFailure(ctx, repo, id, codeRun)
		return true, fmt.Errorf("judge rejected execution %s: %w", id, failure)
	}
	return true, nil
}

func (s *Service) publishDispatchFailure(ctx context.Context, repo *PGRepository, id string, codeRun bool) {
	if codeRun {
		run, tests, err := repo.GetCodeRunByID(ctx, id)
		if err == nil {
			s.publishCodeRun("code_run.failed", run, tests)
		}
		return
	}
	submission, tests, err := repo.GetByID(ctx, id)
	if err == nil {
		s.publishSubmission(ctx, "submission.failed", submission, tests)
	}
}
