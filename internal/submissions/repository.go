package submissions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	timeLayout                = time.RFC3339
	maxActiveJudgeJobsPerUser = 4
)

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

	CreateCodeRun(ctx context.Context, run *CodeRun) error
	GetCodeRunByID(ctx context.Context, id string) (*CodeRun, []CodeRunTestResult, error)
	GetCodeRunByJudgeSubmissionID(ctx context.Context, judgeSubmissionID string) (*CodeRun, error)
	MarkCodeRunSentToJudge(ctx context.Context, codeRunID, judgeSubmissionID, judgeStatus string) error
	MarkCodeRunRunning(ctx context.Context, codeRunID, judgeStatus string, startedAt *time.Time) error
	CompleteCodeRun(ctx context.Context, codeRunID string, input CompleteInput, tests []CodeRunTestResult) error

	GetJudgeTaskLimits(ctx context.Context, taskID string) (*JudgeTaskLimits, error)
	ListJudgeTaskTestCases(ctx context.Context, taskID string) ([]JudgeTaskTestCase, error)
	ListJudgeTaskSampleTestCases(ctx context.Context, taskID string) ([]JudgeTaskTestCase, error)
}

type PGRepository struct {
	db submissionsDB
}

type submissionsDB interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
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
	return r.createSubmission(ctx, s, nil)
}

func (r *PGRepository) createSubmission(ctx context.Context, s *Submission, dispatch []byte) error {
	const q = `
with admission as (select clock_timestamp() as admitted_at)
insert into submissions (
    id, match_id, task_id, user_id, language, language_version, source_code,
    status, verdict, judge_submission_id, judge_status, error_message,
    passed_tests, total_tests, compile_time_ms, run_time_ms, peak_memory_kb,
    score, exit_code, compile_log, stdout_truncated, stderr_truncated,
    created_at, started_at, finished_at, updated_at, tournament_id
)
select
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12,
    $13, $14, $15, $16, $17,
    $18, $19, $20, $21, $22,
    case when $27::uuid is null then $23::timestamptz else a.admitted_at end, $24, $25,
    case when $27::uuid is null then $26::timestamptz else a.admitted_at end, $27
from admission a
where $27::uuid is null or exists (
    select 1 from tournaments t where t.id=$27 and t.status in ('published','running')
        and a.admitted_at >= t.starts_at and a.admitted_at < t.ends_at
)`

	return r.createWithJudgeCapacity(ctx, s.UserID, s.Status, s, dispatch, false, q,
		s.ID, s.MatchID, s.TaskID, s.UserID, s.Language, s.LanguageVersion, s.SourceCode,
		s.Status, s.Verdict, s.JudgeSubmissionID, s.JudgeStatus, s.ErrorMessage,
		s.PassedTests, s.TotalTests, s.CompileTimeMs, s.RunTimeMs, s.PeakMemoryKb,
		s.Score, s.ExitCode, s.CompileLog, s.StdoutTruncated, s.StderrTruncated,
		s.CreatedAt, s.StartedAt, s.FinishedAt, s.UpdatedAt, s.TournamentID,
	)
}

func (r *PGRepository) CreateCodeRun(ctx context.Context, run *CodeRun) error {
	return r.createCodeRun(ctx, run, nil)
}

func (r *PGRepository) createCodeRun(ctx context.Context, run *CodeRun, dispatch []byte) error {
	const q = `
insert into code_runs (
    id, task_id, user_id, language, language_version, source_code,
    status, verdict, judge_submission_id, judge_status, error_message,
    passed_tests, total_tests, compile_time_ms, run_time_ms, peak_memory_kb,
    score, exit_code, compile_log, stdout_truncated, stderr_truncated,
    created_at, started_at, finished_at, updated_at
) values (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    $12, $13, $14, $15, $16,
    $17, $18, $19, $20, $21,
    $22, $23, $24, $25
)`

	return r.createWithJudgeCapacity(ctx, run.UserID, run.Status, nil, dispatch, true, q,
		run.ID, run.TaskID, run.UserID, run.Language, run.LanguageVersion, run.SourceCode,
		run.Status, run.Verdict, run.JudgeSubmissionID, run.JudgeStatus, run.ErrorMessage,
		run.PassedTests, run.TotalTests, run.CompileTimeMs, run.RunTimeMs, run.PeakMemoryKb,
		run.Score, run.ExitCode, run.CompileLog, run.StdoutTruncated, run.StderrTruncated,
		run.CreatedAt, run.StartedAt, run.FinishedAt, run.UpdatedAt,
	)
}

func (r *PGRepository) createWithJudgeCapacity(ctx context.Context, userID, status string, submission *Submission, dispatch []byte, codeRun bool, query string, args ...any) error {
	inContest := submission != nil && submission.TournamentID != nil
	if status != StatusQueued && !inContest {
		_, err := r.db.Exec(ctx, query, args...)
		return err
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	// Every queued run or submission by this user shares the same admission lock.
	if _, err := tx.Exec(ctx,
		"select pg_advisory_xact_lock(hashtextextended('judge-admission:' || $1::text, 0))", userID,
	); err != nil {
		return err
	}

	// A queued job may already have been accepted by Judge even if its HTTP
	// response never arrived. Only a terminal state releases its reservation.
	const activeJobs = `
select count(*) from (
    select 1 from submissions
    where user_id = $1 and status in ('queued', 'sent_to_judge', 'running')
    union all
    select 1 from code_runs
    where user_id = $1 and status in ('queued', 'sent_to_judge', 'running')
    limit $2
) active`
	var count int
	if err := tx.QueryRow(ctx, activeJobs, userID, maxActiveJudgeJobsPerUser).Scan(&count); err != nil {
		return err
	}
	if count >= maxActiveJudgeJobsPerUser {
		return ErrJudgeActiveLimit
	}
	if submission != nil && submission.MatchID != nil {
		if err := requireOpenMatchForSubmission(ctx, tx, *submission.MatchID); err != nil {
			return err
		}
	}
	if inContest {
		if err := requireContestEntry(ctx, tx, submission); err != nil {
			return err
		}
	}
	result, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if inContest && result.RowsAffected() == 0 {
		return ErrContestSubmissionWindow
	}
	if dispatch != nil {
		if _, err := tx.Exec(ctx, `insert into judge_dispatch (id, submission_id, code_run_id, payload)
			values ($1::uuid, case when not $2::boolean then $1 end, case when $2 then $1 end, $3::jsonb)`, args[0], codeRun, dispatch); err != nil {
			return fmt.Errorf("queue judge dispatch: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func requireContestEntry(ctx context.Context, tx pgx.Tx, submission *Submission) error {
	// Hold the shared tournament lock through admission so completion must
	// include this submission or close the window before the insert runs.
	var joined bool
	err := tx.QueryRow(ctx, `select exists (select 1 from tournament_participants p
		where p.tournament_id=t.id and p.user_id=$3)
		from tournaments t join tournament_problems p on p.tournament_id=t.id and p.task_id=$2
		where t.id=$1 and t.format='shared_problem_set' for share of t`,
		*submission.TournamentID, submission.TaskID, submission.UserID).Scan(&joined)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrContestProblemNotFound
	}
	if err != nil {
		return err
	}
	if !joined {
		return ErrContestNotJoined
	}
	return nil
}

func requireOpenMatchForSubmission(ctx context.Context, tx pgx.Tx, matchID string) error {
	var status string
	var cutoff *time.Time
	// A scheduler finalizing a tournament takes the same match lock before it
	// checks for outstanding submissions. It cannot miss an admission in flight.
	err := tx.QueryRow(ctx, `select m.status,
			case when t.ends_at is not null then least(m.expires_at, t.ends_at) end
		from matches m left join tournament_fixtures f on f.match_id=m.id
		left join tournaments t on t.id=f.tournament_id where m.id=$1 for update of m`, matchID).Scan(&status, &cutoff)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrMatchNotFound
	}
	if err != nil {
		return err
	}
	if status != "running" {
		return ErrMatchNotStarted
	}
	if cutoff != nil {
		var open bool
		if err := tx.QueryRow(ctx, `select $1::timestamptz > clock_timestamp()`, cutoff).Scan(&open); err != nil {
			return err
		}
		if !open {
			return ErrMatchSubmissionDeadlinePassed
		}
	}
	return nil
}

func (r *PGRepository) GetByID(ctx context.Context, id string) (*Submission, []SubmissionTestResult, error) {
	sub, err := r.getSubmissionRow(ctx, r.db, `select
    id, match_id, task_id, user_id, language, language_version, source_code,
    status, verdict, judge_submission_id, judge_status, error_message,
    passed_tests, total_tests, compile_time_ms, run_time_ms, peak_memory_kb,
    score, exit_code, compile_log, stdout_truncated, stderr_truncated,
    created_at, started_at, finished_at, updated_at, tournament_id
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

func (r *PGRepository) GetCodeRunByID(ctx context.Context, id string) (*CodeRun, []CodeRunTestResult, error) {
	run, err := r.getCodeRunRow(ctx, r.db, `select
    id, task_id, user_id, language, language_version, source_code,
    status, verdict, judge_submission_id, judge_status, error_message,
    passed_tests, total_tests, compile_time_ms, run_time_ms, peak_memory_kb,
    score, exit_code, compile_log, stdout_truncated, stderr_truncated,
    created_at, started_at, finished_at, updated_at
from code_runs
where id = $1`, id)
	if err != nil {
		return nil, nil, err
	}

	const tq = `
select id, code_run_id, test_case_id, test_name, is_custom, status, time_ms, memory_kb, exit_code, checker_message, created_at
from code_run_test_results
where code_run_id = $1
order by created_at asc`

	rows, err := r.db.Query(ctx, tq, id)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var tests []CodeRunTestResult
	for rows.Next() {
		var t CodeRunTestResult
		if err := rows.Scan(
			&t.ID,
			&t.CodeRunID,
			&t.TestCaseID,
			&t.TestName,
			&t.IsCustom,
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

	return run, tests, nil
}

func (r *PGRepository) GetByJudgeSubmissionID(ctx context.Context, judgeSubmissionID string) (*Submission, error) {
	return r.getSubmissionRow(ctx, r.db, `select
    id, match_id, task_id, user_id, language, language_version, source_code,
    status, verdict, judge_submission_id, judge_status, error_message,
    passed_tests, total_tests, compile_time_ms, run_time_ms, peak_memory_kb,
    score, exit_code, compile_log, stdout_truncated, stderr_truncated,
    created_at, started_at, finished_at, updated_at, tournament_id
from submissions
where judge_submission_id = $1`, judgeSubmissionID)
}

func (r *PGRepository) GetCodeRunByJudgeSubmissionID(ctx context.Context, judgeSubmissionID string) (*CodeRun, error) {
	return r.getCodeRunRow(ctx, r.db, `select
    id, task_id, user_id, language, language_version, source_code,
    status, verdict, judge_submission_id, judge_status, error_message,
    passed_tests, total_tests, compile_time_ms, run_time_ms, peak_memory_kb,
    score, exit_code, compile_log, stdout_truncated, stderr_truncated,
    created_at, started_at, finished_at, updated_at
from code_runs
where judge_submission_id = $1`, judgeSubmissionID)
}

func (r *PGRepository) MarkSentToJudge(ctx context.Context, submissionID, judgeSubmissionID, judgeStatus string) error {
	const q = `
update submissions
set
    status = case when status = 'queued' then 'sent_to_judge' else status end,
    judge_submission_id = $2,
    judge_status = case when status = 'queued' then $3 else judge_status end,
    updated_at = now()
where id = $1
  and (judge_submission_id = $2 or (status = 'queued' and judge_submission_id is null))`

	ct, err := r.db.Exec(ctx, q, submissionID, judgeSubmissionID, judgeStatus)
	if err != nil {
		return mapPQError(err)
	}
	if ct.RowsAffected() == 0 {
		return ErrSubmissionNotFound
	}
	return nil
}

func (r *PGRepository) MarkCodeRunSentToJudge(ctx context.Context, codeRunID, judgeSubmissionID, judgeStatus string) error {
	const q = `
update code_runs
set
    status = case when status = 'queued' then 'sent_to_judge' else status end,
    judge_submission_id = $2,
    judge_status = case when status = 'queued' then $3 else judge_status end,
    updated_at = now()
where id = $1
  and (judge_submission_id = $2 or (status = 'queued' and judge_submission_id is null))`

	ct, err := r.db.Exec(ctx, q, codeRunID, judgeSubmissionID, judgeStatus)
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
where id = $1
  and (started_at is null or started_at <= $16)
  and (status not in ('finished', 'failed') or started_at < $16)`

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
		return ErrStaleJudgeCallback
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

func (r *PGRepository) CompleteCodeRun(ctx context.Context, codeRunID string, input CompleteInput, tests []CodeRunTestResult) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	const uq = `
update code_runs
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
where id = $1
  and (started_at is null or started_at <= $16)
  and (status not in ('finished', 'failed') or started_at < $16)`

	ct, err := tx.Exec(ctx, uq,
		codeRunID,
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
		return ErrStaleJudgeCallback
	}

	if _, err := tx.Exec(ctx, `delete from code_run_test_results where code_run_id = $1`, codeRunID); err != nil {
		return err
	}

	if len(tests) > 0 {
		const iq = `
insert into code_run_test_results (
    id, code_run_id, test_case_id, test_name, is_custom, status, time_ms, memory_kb, exit_code, checker_message, created_at
) values (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)`
		for _, t := range tests {
			if t.ID == "" {
				t.ID = uuid.NewString()
			}
			if t.CreatedAt.IsZero() {
				t.CreatedAt = time.Now().UTC()
			}
			if _, err := tx.Exec(ctx, iq,
				t.ID, t.CodeRunID, t.TestCaseID, t.TestName, t.IsCustom, t.Status, t.TimeMs, t.MemoryKb, t.ExitCode, t.CheckerMessage, t.CreatedAt,
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
		&s.TournamentID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *PGRepository) getCodeRunRow(ctx context.Context, q querier, sql string, arg any) (*CodeRun, error) {
	var run CodeRun
	err := q.QueryRow(ctx, sql, arg).Scan(
		&run.ID,
		&run.TaskID,
		&run.UserID,
		&run.Language,
		&run.LanguageVersion,
		&run.SourceCode,
		&run.Status,
		&run.Verdict,
		&run.JudgeSubmissionID,
		&run.JudgeStatus,
		&run.ErrorMessage,
		&run.PassedTests,
		&run.TotalTests,
		&run.CompileTimeMs,
		&run.RunTimeMs,
		&run.PeakMemoryKb,
		&run.Score,
		&run.ExitCode,
		&run.CompileLog,
		&run.StdoutTruncated,
		&run.StderrTruncated,
		&run.CreatedAt,
		&run.StartedAt,
		&run.FinishedAt,
		&run.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}
	return &run, nil
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

func (r *PGRepository) ListJudgeTaskSampleTestCases(ctx context.Context, taskID string) ([]JudgeTaskTestCase, error) {
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
		  and kind = 'sample'
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
    started_at = coalesce($3, started_at),
    updated_at = now()
where id = $1
  and (started_at is null or started_at <= $3)
  and (status not in ('finished', 'failed') or started_at < $3)`

	ct, err := r.db.Exec(ctx, q, submissionID, judgeStatus, startedAt)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrStaleJudgeCallback
	}
	return nil
}

func (r *PGRepository) MarkCodeRunRunning(ctx context.Context, codeRunID, judgeStatus string, startedAt *time.Time) error {
	const q = `
update code_runs
set
    status = 'running',
    judge_status = $2,
    started_at = coalesce($3, started_at),
    updated_at = now()
where id = $1
  and (started_at is null or started_at <= $3)
  and (status not in ('finished', 'failed') or started_at < $3)`

	ct, err := r.db.Exec(ctx, q, codeRunID, judgeStatus, startedAt)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrStaleJudgeCallback
	}
	return nil
}
