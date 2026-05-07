package submissions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const timeLayout = time.RFC3339

type CompleteInput struct {
	Status          string
	JudgeStatus     string
	Verdict         *string
	ErrorMessage    *string
	PassedTests     int
	TotalTests      int
	CompileTimeMs   *int
	RunTimeMs       *int
	PeakMemoryKb    *int
	Score           int
	ExitCode        *int
	CompileLog      *string
	StdoutTruncated *string
	StderrTruncated *string
	StartedAt       *time.Time
	FinishedAt      *time.Time
}

type Repository interface {
	Create(ctx context.Context, s *Submission) error
	GetByID(ctx context.Context, id string) (*Submission, []SubmissionTestResult, error)
	GetByJudgeSubmissionID(ctx context.Context, judgeSubmissionID string) (*Submission, error)
	MarkSentToJudge(ctx context.Context, submissionID, judgeSubmissionID, judgeStatus string) error
	MarkRunning(ctx context.Context, submissionID, judgeStatus string, startedAt *time.Time) error
	Complete(ctx context.Context, submissionID string, input CompleteInput, tests []SubmissionTestResult) error

	GetJudgeTaskLimits(ctx context.Context, taskID string) (*JudgeTaskLimits, error)
	ListJudgeTaskTestCases(ctx context.Context, taskID string) ([]JudgeTaskTestCase, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

type JudgeTaskLimits struct {
	TimeLimitMS   int
	MemoryLimitKB int
}

type JudgeTaskTestCase struct {
	ID              string
	OrderNo         int
	InputBlobRef    string
	ExpectedBlobRef string
	Weight          int
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) Create(ctx context.Context, s *Submission) error {
	const q = `
insert into submissions (
    id, match_id, task_id, user_id, language, language_version, source_code,
    status, verdict, judge_submission_id, judge_status, error_message,
    passed_tests, total_tests, compile_time_ms, run_time_ms, peak_memory_kb,
    score, exit_code, compile_log, stdout_truncated, stderr_truncated,
    created_at, started_at, finished_at, updated_at
) values (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12,
    $13, $14, $15, $16, $17,
    $18, $19, $20, $21, $22,
    $23, $24, $25, $26
)`

	_, err := r.db.Exec(ctx, q,
		s.ID, s.MatchID, s.TaskID, s.UserID, s.Language, s.LanguageVersion, s.SourceCode,
		s.Status, s.Verdict, s.JudgeSubmissionID, s.JudgeStatus, s.ErrorMessage,
		s.PassedTests, s.TotalTests, s.CompileTimeMs, s.RunTimeMs, s.PeakMemoryKb,
		s.Score, s.ExitCode, s.CompileLog, s.StdoutTruncated, s.StderrTruncated,
		s.CreatedAt, s.StartedAt, s.FinishedAt, s.UpdatedAt,
	)
	return err
}

func (r *PGRepository) GetByID(ctx context.Context, id string) (*Submission, []SubmissionTestResult, error) {
	sub, err := r.getSubmissionRow(ctx, r.db, `select
    id, match_id, task_id, user_id, language, language_version, source_code,
    status, verdict, judge_submission_id, judge_status, error_message,
    passed_tests, total_tests, compile_time_ms, run_time_ms, peak_memory_kb,
    score, exit_code, compile_log, stdout_truncated, stderr_truncated,
    created_at, started_at, finished_at, updated_at
from submissions
where id = $1`, id)
	if err != nil {
		return nil, nil, err
	}

	const tq = `
select id, submission_id, test_case_id, status, time_ms, memory_kb, exit_code, checker_message, created_at
from submission_test_results
where submission_id = $1
order by created_at asc`

	rows, err := r.db.Query(ctx, tq, id)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var tests []SubmissionTestResult
	for rows.Next() {
		var t SubmissionTestResult
		if err := rows.Scan(
			&t.ID,
			&t.SubmissionID,
			&t.TestCaseID,
			&t.Status,
			&t.TimeMs,
			&t.MemoryKb,
			&t.ExitCode,
			&t.CheckerMessage,
			&t.CreatedAt,
		); err != nil {
			return nil, nil, err
		}
		tests = append(tests, t)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return sub, tests, nil
}

func (r *PGRepository) GetByJudgeSubmissionID(ctx context.Context, judgeSubmissionID string) (*Submission, error) {
	return r.getSubmissionRow(ctx, r.db, `select
    id, match_id, task_id, user_id, language, language_version, source_code,
    status, verdict, judge_submission_id, judge_status, error_message,
    passed_tests, total_tests, compile_time_ms, run_time_ms, peak_memory_kb,
    score, exit_code, compile_log, stdout_truncated, stderr_truncated,
    created_at, started_at, finished_at, updated_at
from submissions
where judge_submission_id = $1`, judgeSubmissionID)
}

func (r *PGRepository) MarkSentToJudge(ctx context.Context, submissionID, judgeSubmissionID, judgeStatus string) error {
	const q = `
update submissions
set
    status = 'sent_to_judge',
    judge_submission_id = $2,
    judge_status = $3,
    updated_at = now()
where id = $1`

	ct, err := r.db.Exec(ctx, q, submissionID, judgeSubmissionID, judgeStatus)
	if err != nil {
		return mapPQError(err)
	}
	if ct.RowsAffected() == 0 {
		return ErrSubmissionNotFound
	}
	return nil
}

func (r *PGRepository) Complete(ctx context.Context, submissionID string, input CompleteInput, tests []SubmissionTestResult) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	const uq = `
update submissions
set
    status = $2,
    judge_status = $3,
    verdict = $4,
    error_message = $5,
    passed_tests = $6,
    total_tests = $7,
    compile_time_ms = $8,
    run_time_ms = $9,
    peak_memory_kb = $10,
    score = $11,
    exit_code = $12,
    compile_log = $13,
    stdout_truncated = $14,
    stderr_truncated = $15,
    started_at = $16,
    finished_at = $17,
    updated_at = now()
where id = $1`

	ct, err := tx.Exec(ctx, uq,
		submissionID,
		input.Status,
		input.JudgeStatus,
		input.Verdict,
		input.ErrorMessage,
		input.PassedTests,
		input.TotalTests,
		input.CompileTimeMs,
		input.RunTimeMs,
		input.PeakMemoryKb,
		input.Score,
		input.ExitCode,
		input.CompileLog,
		input.StdoutTruncated,
		input.StderrTruncated,
		input.StartedAt,
		input.FinishedAt,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrSubmissionNotFound
	}

	if _, err := tx.Exec(ctx, `delete from submission_test_results where submission_id = $1`, submissionID); err != nil {
		return err
	}

	if len(tests) > 0 {
		const iq = `
insert into submission_test_results (
    id, submission_id, test_case_id, status, time_ms, memory_kb, exit_code, checker_message, created_at
) values (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)`
		for _, t := range tests {
			if t.ID == "" {
				t.ID = uuid.NewString()
			}
			if t.CreatedAt.IsZero() {
				t.CreatedAt = time.Now().UTC()
			}
			if _, err := tx.Exec(ctx, iq,
				t.ID, t.SubmissionID, t.TestCaseID, t.Status, t.TimeMs, t.MemoryKb, t.ExitCode, t.CheckerMessage, t.CreatedAt,
			); err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

func (r *PGRepository) getSubmissionRow(ctx context.Context, q querier, sql string, arg any) (*Submission, error) {
	var s Submission
	err := q.QueryRow(ctx, sql, arg).Scan(
		&s.ID,
		&s.MatchID,
		&s.TaskID,
		&s.UserID,
		&s.Language,
		&s.LanguageVersion,
		&s.SourceCode,
		&s.Status,
		&s.Verdict,
		&s.JudgeSubmissionID,
		&s.JudgeStatus,
		&s.ErrorMessage,
		&s.PassedTests,
		&s.TotalTests,
		&s.CompileTimeMs,
		&s.RunTimeMs,
		&s.PeakMemoryKb,
		&s.Score,
		&s.ExitCode,
		&s.CompileLog,
		&s.StdoutTruncated,
		&s.StderrTruncated,
		&s.CreatedAt,
		&s.StartedAt,
		&s.FinishedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *PGRepository) GetJudgeTaskLimits(ctx context.Context, taskID string) (*JudgeTaskLimits, error) {
	var out JudgeTaskLimits

	err := r.db.QueryRow(ctx, `
		select
			time_limit_ms,
			memory_limit_kb
		from tasks
		where id = $1
		  and is_active = true
	`, taskID).Scan(
		&out.TimeLimitMS,
		&out.MemoryLimitKB,
	)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

func (r *PGRepository) ListJudgeTaskTestCases(ctx context.Context, taskID string) ([]JudgeTaskTestCase, error) {
	rows, err := r.db.Query(ctx, `
		select
			id,
			order_no,
			input_blob_ref,
			expected_blob_ref,
			weight
		from task_test_cases
		where task_id = $1
		  and is_active = true
		order by order_no asc
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tests []JudgeTaskTestCase

	for rows.Next() {
		var t JudgeTaskTestCase

		if err := rows.Scan(
			&t.ID,
			&t.OrderNo,
			&t.InputBlobRef,
			&t.ExpectedBlobRef,
			&t.Weight,
		); err != nil {
			return nil, err
		}

		tests = append(tests, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tests, nil
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func mapPQError(err error) error {
	msg := err.Error()
	if contains(msg, "submissions_judge_submission_id_uidx") {
		return ErrDuplicateJudgeSubmissionID
	}
	return err
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && fmt.Sprintf("%v", s) != "" && (func() bool {
		return stringContains(s, sub)
	})()
}

func stringContains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func (r *PGRepository) MarkRunning(ctx context.Context, submissionID, judgeStatus string, startedAt *time.Time) error {
	const q = `
update submissions
set
    status = 'running',
    judge_status = $2,
    started_at = coalesce(started_at, $3, now()),
    updated_at = now()
where id = $1
  and status not in ('finished', 'failed')`

	ct, err := r.db.Exec(ctx, q, submissionID, judgeStatus, startedAt)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return nil
	}
	return nil
}
