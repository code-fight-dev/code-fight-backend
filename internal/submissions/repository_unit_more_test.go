package submissions

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPGRepositoryReadAndUpdateMethodsWithFakeDB(t *testing.T) {
	now := time.Now().UTC()
	subValues := fakeSubmissionRowValues(&Submission{
		ID:              "sub-1",
		TaskID:          "task-1",
		UserID:          "user-1",
		Language:        "go",
		LanguageVersion: "1.24",
		SourceCode:      "package main",
		Status:          StatusSentToJudge,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	runValues := fakeCodeRunRowValues(&CodeRun{
		ID:              "run-1",
		TaskID:          "task-1",
		UserID:          "user-1",
		Language:        "go",
		LanguageVersion: "1.24",
		SourceCode:      "package main",
		Status:          StatusSentToJudge,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	timeMs := 10
	memoryKb := 1024

	db := &fakeSubmissionsDB{
		rowResults: []*fakeSubmissionsRow{
			{values: subValues},
			{values: runValues},
			{values: subValues},
			{values: runValues},
			{values: []any{1500, 131072}},
		},
		rows: []*fakeSubmissionsRows{
			{rows: [][]any{{"test-1", "sub-1", "case-1", VerdictAccepted, &timeMs, &memoryKb, nil, nil, now}}},
			{rows: [][]any{{"run-test-1", "run-1", ptrString("case-1"), nil, false, VerdictAccepted, &timeMs, &memoryKb, nil, nil, now}}},
			{rows: [][]any{{"case-1", 1, "in", "out", 2}}},
			{rows: [][]any{{"sample-1", 1, "sample in", "sample out", 1}}},
		},
		execTags: []pgconn.CommandTag{
			pgconn.NewCommandTag("INSERT 0 1"),
			pgconn.NewCommandTag("INSERT 0 1"),
			pgconn.NewCommandTag("UPDATE 1"),
			pgconn.NewCommandTag("UPDATE 1"),
			pgconn.NewCommandTag("UPDATE 1"),
			pgconn.NewCommandTag("UPDATE 1"),
		},
	}
	repo := &PGRepository{db: db}

	if err := repo.Create(context.Background(), &Submission{ID: "sub-1"}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := repo.CreateCodeRun(context.Background(), &CodeRun{ID: "run-1"}); err != nil {
		t.Fatalf("CreateCodeRun returned error: %v", err)
	}
	sub, tests, err := repo.GetByID(context.Background(), "sub-1")
	if err != nil || sub.ID != "sub-1" || len(tests) != 1 {
		t.Fatalf("GetByID = %#v/%#v, %v", sub, tests, err)
	}
	run, runTests, err := repo.GetCodeRunByID(context.Background(), "run-1")
	if err != nil || run.ID != "run-1" || len(runTests) != 1 {
		t.Fatalf("GetCodeRunByID = %#v/%#v, %v", run, runTests, err)
	}
	if _, err := repo.GetByJudgeSubmissionID(context.Background(), "judge-sub-1"); err != nil {
		t.Fatalf("GetByJudgeSubmissionID returned error: %v", err)
	}
	if _, err := repo.GetCodeRunByJudgeSubmissionID(context.Background(), "judge-run-1"); err != nil {
		t.Fatalf("GetCodeRunByJudgeSubmissionID returned error: %v", err)
	}
	if err := repo.MarkSentToJudge(context.Background(), "sub-1", "judge-sub-1", JudgeStatusQueued); err != nil {
		t.Fatalf("MarkSentToJudge returned error: %v", err)
	}
	if err := repo.MarkCodeRunSentToJudge(context.Background(), "run-1", "judge-run-1", JudgeStatusQueued); err != nil {
		t.Fatalf("MarkCodeRunSentToJudge returned error: %v", err)
	}
	if err := repo.MarkRunning(context.Background(), "sub-1", JudgeStatusRunning, &now); err != nil {
		t.Fatalf("MarkRunning returned error: %v", err)
	}
	if err := repo.MarkCodeRunRunning(context.Background(), "run-1", JudgeStatusRunning, &now); err != nil {
		t.Fatalf("MarkCodeRunRunning returned error: %v", err)
	}
	limits, err := repo.GetJudgeTaskLimits(context.Background(), "task-1")
	if err != nil || limits.TimeLimitMS != 1500 || limits.MemoryLimitKB != 131072 {
		t.Fatalf("GetJudgeTaskLimits = %#v, %v", limits, err)
	}
	cases, err := repo.ListJudgeTaskTestCases(context.Background(), "task-1")
	if err != nil || len(cases) != 1 || cases[0].Weight != 2 {
		t.Fatalf("ListJudgeTaskTestCases = %#v, %v", cases, err)
	}
	samples, err := repo.ListJudgeTaskSampleTestCases(context.Background(), "task-1")
	if err != nil || len(samples) != 1 || samples[0].ID != "sample-1" {
		t.Fatalf("ListJudgeTaskSampleTestCases = %#v, %v", samples, err)
	}
}

func TestPGRepositoryMapsMissingRowsAndUpdateMisses(t *testing.T) {
	repo := &PGRepository{db: &fakeSubmissionsDB{
		rowResults: []*fakeSubmissionsRow{{err: pgx.ErrNoRows}, {err: pgx.ErrNoRows}},
		execTags:   []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 0"), pgconn.NewCommandTag("UPDATE 0")},
	}}

	if _, _, err := repo.GetByID(context.Background(), "missing"); !errors.Is(err, ErrSubmissionNotFound) {
		t.Fatalf("GetByID error = %v", err)
	}
	if _, _, err := repo.GetCodeRunByID(context.Background(), "missing"); !errors.Is(err, ErrSubmissionNotFound) {
		t.Fatalf("GetCodeRunByID error = %v", err)
	}
	if err := repo.MarkSentToJudge(context.Background(), "missing", "judge", JudgeStatusQueued); !errors.Is(err, ErrSubmissionNotFound) {
		t.Fatalf("MarkSentToJudge error = %v", err)
	}
	if err := repo.MarkCodeRunSentToJudge(context.Background(), "missing", "judge", JudgeStatusQueued); !errors.Is(err, ErrSubmissionNotFound) {
		t.Fatalf("MarkCodeRunSentToJudge error = %v", err)
	}
}

func TestPGRepositoryCompleteTransactions(t *testing.T) {
	now := time.Now().UTC()
	verdict := VerdictAccepted
	tx := &fakeSubmissionsTx{
		execTags: []pgconn.CommandTag{
			pgconn.NewCommandTag("UPDATE 1"),
			pgconn.NewCommandTag("DELETE 1"),
			pgconn.NewCommandTag("INSERT 0 1"),
			pgconn.NewCommandTag("UPDATE 1"),
			pgconn.NewCommandTag("DELETE 1"),
			pgconn.NewCommandTag("INSERT 0 1"),
		},
	}
	repo := &PGRepository{db: &fakeSubmissionsDB{tx: tx}}

	err := repo.Complete(context.Background(), "sub-1", CompleteInput{
		Status:      StatusFinished,
		JudgeStatus: JudgeStatusFinished,
		Verdict:     &verdict,
		PassedTests: 1,
		TotalTests:  1,
		Score:       100,
		FinishedAt:  &now,
	}, []SubmissionTestResult{{SubmissionID: "sub-1", TestCaseID: "case-1", Status: VerdictAccepted}})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	err = repo.CompleteCodeRun(context.Background(), "run-1", CompleteInput{
		Status:      StatusFinished,
		JudgeStatus: JudgeStatusFinished,
		Verdict:     &verdict,
		PassedTests: 1,
		TotalTests:  1,
		Score:       100,
		FinishedAt:  &now,
	}, []CodeRunTestResult{{CodeRunID: "run-1", Status: VerdictAccepted}})
	if err != nil {
		t.Fatalf("CompleteCodeRun returned error: %v", err)
	}
	if tx.commits != 2 || tx.rollbacks != 2 {
		t.Fatalf("transaction counts commits=%d rollbacks=%d", tx.commits, tx.rollbacks)
	}
}

func TestPGRepositoryCompleteReturnsNotFoundOnMissingUpdate(t *testing.T) {
	repo := &PGRepository{db: &fakeSubmissionsDB{tx: &fakeSubmissionsTx{
		execTags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 0"), pgconn.NewCommandTag("UPDATE 0")},
	}}}

	if err := repo.Complete(context.Background(), "missing", CompleteInput{}, nil); !errors.Is(err, ErrSubmissionNotFound) {
		t.Fatalf("Complete error = %v", err)
	}
	if err := repo.CompleteCodeRun(context.Background(), "missing", CompleteInput{}, nil); !errors.Is(err, ErrSubmissionNotFound) {
		t.Fatalf("CompleteCodeRun error = %v", err)
	}
}

func TestPGRepositoryErrorBranchesWithFakeDB(t *testing.T) {
	now := time.Now().UTC()
	dbErr := errors.New("db failed")
	subValues := fakeSubmissionRowValues(&Submission{ID: "sub-1", TaskID: "task-1", UserID: "user-1", CreatedAt: now, UpdatedAt: now})
	runValues := fakeCodeRunRowValues(&CodeRun{ID: "run-1", TaskID: "task-1", UserID: "user-1", CreatedAt: now, UpdatedAt: now})

	tests := []struct {
		name string
		repo *PGRepository
		call func(*PGRepository) error
	}{
		{
			name: "create",
			repo: &PGRepository{db: &fakeSubmissionsDB{execErr: dbErr}},
			call: func(repo *PGRepository) error {
				return repo.Create(context.Background(), &Submission{ID: "sub-1"})
			},
		},
		{
			name: "create code run",
			repo: &PGRepository{db: &fakeSubmissionsDB{execErr: dbErr}},
			call: func(repo *PGRepository) error {
				return repo.CreateCodeRun(context.Background(), &CodeRun{ID: "run-1"})
			},
		},
		{
			name: "get tests query",
			repo: &PGRepository{db: &fakeSubmissionsDB{rowResults: []*fakeSubmissionsRow{{values: subValues}}, queryErr: dbErr}},
			call: func(repo *PGRepository) error {
				_, _, err := repo.GetByID(context.Background(), "sub-1")
				return err
			},
		},
		{
			name: "get tests scan",
			repo: &PGRepository{db: &fakeSubmissionsDB{rowResults: []*fakeSubmissionsRow{{values: subValues}}, rows: []*fakeSubmissionsRows{{rows: [][]any{{"test-1"}}}}}},
			call: func(repo *PGRepository) error {
				_, _, err := repo.GetByID(context.Background(), "sub-1")
				return err
			},
		},
		{
			name: "get tests iterate",
			repo: &PGRepository{db: &fakeSubmissionsDB{rowResults: []*fakeSubmissionsRow{{values: subValues}}, rows: []*fakeSubmissionsRows{{err: dbErr}}}},
			call: func(repo *PGRepository) error {
				_, _, err := repo.GetByID(context.Background(), "sub-1")
				return err
			},
		},
		{
			name: "get run tests query",
			repo: &PGRepository{db: &fakeSubmissionsDB{rowResults: []*fakeSubmissionsRow{{values: runValues}}, queryErr: dbErr}},
			call: func(repo *PGRepository) error {
				_, _, err := repo.GetCodeRunByID(context.Background(), "run-1")
				return err
			},
		},
		{
			name: "get run tests scan",
			repo: &PGRepository{db: &fakeSubmissionsDB{rowResults: []*fakeSubmissionsRow{{values: runValues}}, rows: []*fakeSubmissionsRows{{rows: [][]any{{"run-test-1"}}}}}},
			call: func(repo *PGRepository) error {
				_, _, err := repo.GetCodeRunByID(context.Background(), "run-1")
				return err
			},
		},
		{
			name: "get run tests iterate",
			repo: &PGRepository{db: &fakeSubmissionsDB{rowResults: []*fakeSubmissionsRow{{values: runValues}}, rows: []*fakeSubmissionsRows{{err: dbErr}}}},
			call: func(repo *PGRepository) error {
				_, _, err := repo.GetCodeRunByID(context.Background(), "run-1")
				return err
			},
		},
		{
			name: "limits",
			repo: &PGRepository{db: &fakeSubmissionsDB{rowResults: []*fakeSubmissionsRow{{err: dbErr}}}},
			call: func(repo *PGRepository) error {
				_, err := repo.GetJudgeTaskLimits(context.Background(), "task-1")
				return err
			},
		},
		{
			name: "test cases query",
			repo: &PGRepository{db: &fakeSubmissionsDB{queryErr: dbErr}},
			call: func(repo *PGRepository) error {
				_, err := repo.ListJudgeTaskTestCases(context.Background(), "task-1")
				return err
			},
		},
		{
			name: "test cases scan",
			repo: &PGRepository{db: &fakeSubmissionsDB{rows: []*fakeSubmissionsRows{{rows: [][]any{{"case-1"}}}}}},
			call: func(repo *PGRepository) error {
				_, err := repo.ListJudgeTaskTestCases(context.Background(), "task-1")
				return err
			},
		},
		{
			name: "test cases iterate",
			repo: &PGRepository{db: &fakeSubmissionsDB{rows: []*fakeSubmissionsRows{{err: dbErr}}}},
			call: func(repo *PGRepository) error {
				_, err := repo.ListJudgeTaskTestCases(context.Background(), "task-1")
				return err
			},
		},
		{
			name: "sample cases query",
			repo: &PGRepository{db: &fakeSubmissionsDB{queryErr: dbErr}},
			call: func(repo *PGRepository) error {
				_, err := repo.ListJudgeTaskSampleTestCases(context.Background(), "task-1")
				return err
			},
		},
		{
			name: "sample cases scan",
			repo: &PGRepository{db: &fakeSubmissionsDB{rows: []*fakeSubmissionsRows{{rows: [][]any{{"case-1"}}}}}},
			call: func(repo *PGRepository) error {
				_, err := repo.ListJudgeTaskSampleTestCases(context.Background(), "task-1")
				return err
			},
		},
		{
			name: "sample cases iterate",
			repo: &PGRepository{db: &fakeSubmissionsDB{rows: []*fakeSubmissionsRows{{err: dbErr}}}},
			call: func(repo *PGRepository) error {
				_, err := repo.ListJudgeTaskSampleTestCases(context.Background(), "task-1")
				return err
			},
		},
		{
			name: "mark running",
			repo: &PGRepository{db: &fakeSubmissionsDB{execErr: dbErr}},
			call: func(repo *PGRepository) error {
				return repo.MarkRunning(context.Background(), "sub-1", JudgeStatusRunning, nil)
			},
		},
		{
			name: "mark code run running",
			repo: &PGRepository{db: &fakeSubmissionsDB{execErr: dbErr}},
			call: func(repo *PGRepository) error {
				return repo.MarkCodeRunRunning(context.Background(), "run-1", JudgeStatusRunning, nil)
			},
		},
		{
			name: "complete begin",
			repo: &PGRepository{db: &fakeSubmissionsDB{beginErr: dbErr}},
			call: func(repo *PGRepository) error {
				return repo.Complete(context.Background(), "sub-1", CompleteInput{}, nil)
			},
		},
		{
			name: "complete code run begin",
			repo: &PGRepository{db: &fakeSubmissionsDB{beginErr: dbErr}},
			call: func(repo *PGRepository) error {
				return repo.CompleteCodeRun(context.Background(), "run-1", CompleteInput{}, nil)
			},
		},
		{
			name: "complete commit",
			repo: &PGRepository{db: &fakeSubmissionsDB{tx: &fakeSubmissionsTx{execTags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 1"), pgconn.NewCommandTag("DELETE 1")}, commitErr: dbErr}}},
			call: func(repo *PGRepository) error {
				return repo.Complete(context.Background(), "sub-1", CompleteInput{}, nil)
			},
		},
		{
			name: "complete code run commit",
			repo: &PGRepository{db: &fakeSubmissionsDB{tx: &fakeSubmissionsTx{execTags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 1"), pgconn.NewCommandTag("DELETE 1")}, commitErr: dbErr}}},
			call: func(repo *PGRepository) error {
				return repo.CompleteCodeRun(context.Background(), "run-1", CompleteInput{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(tt.repo); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestPGRepositoryDuplicateJudgeSubmissionID(t *testing.T) {
	duplicate := errors.New("violates submissions_judge_submission_id_uidx")
	repo := &PGRepository{db: &fakeSubmissionsDB{execErr: duplicate}}
	if err := repo.MarkSentToJudge(context.Background(), "sub-1", "judge", JudgeStatusQueued); !errors.Is(err, ErrDuplicateJudgeSubmissionID) {
		t.Fatalf("MarkSentToJudge error = %v", err)
	}
	repo = &PGRepository{db: &fakeSubmissionsDB{execErr: duplicate}}
	if err := repo.MarkCodeRunSentToJudge(context.Background(), "run-1", "judge", JudgeStatusQueued); !errors.Is(err, ErrDuplicateJudgeSubmissionID) {
		t.Fatalf("MarkCodeRunSentToJudge error = %v", err)
	}
}

type fakeSubmissionsDB struct {
	rowResults []*fakeSubmissionsRow
	rows       []*fakeSubmissionsRows
	execTags   []pgconn.CommandTag
	tx         *fakeSubmissionsTx
	execErr    error
	queryErr   error
	beginErr   error
}

func (db *fakeSubmissionsDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	if db.execErr != nil {
		return pgconn.CommandTag{}, db.execErr
	}
	if len(db.execTags) == 0 {
		return pgconn.NewCommandTag("UPDATE 0"), nil
	}
	tag := db.execTags[0]
	db.execTags = db.execTags[1:]
	return tag, nil
}

func (db *fakeSubmissionsDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	if db.queryErr != nil {
		return nil, db.queryErr
	}
	if len(db.rows) == 0 {
		return &fakeSubmissionsRows{}, nil
	}
	rows := db.rows[0]
	db.rows = db.rows[1:]
	return rows, nil
}

func (db *fakeSubmissionsDB) QueryRow(context.Context, string, ...any) pgx.Row {
	if len(db.rowResults) == 0 {
		return &fakeSubmissionsRow{err: pgx.ErrNoRows}
	}
	row := db.rowResults[0]
	db.rowResults = db.rowResults[1:]
	return row
}

func (db *fakeSubmissionsDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	if db.beginErr != nil {
		return nil, db.beginErr
	}
	if db.tx == nil {
		return nil, errors.New("tx not configured")
	}
	return db.tx, nil
}

type fakeSubmissionsTx struct {
	execTags  []pgconn.CommandTag
	commits   int
	rollbacks int
	commitErr error
	execErr   error
}

func (tx *fakeSubmissionsTx) Begin(context.Context) (pgx.Tx, error) { return tx, nil }
func (tx *fakeSubmissionsTx) Commit(context.Context) error {
	tx.commits++
	return tx.commitErr
}
func (tx *fakeSubmissionsTx) Rollback(context.Context) error {
	tx.rollbacks++
	return nil
}
func (tx *fakeSubmissionsTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (tx *fakeSubmissionsTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (tx *fakeSubmissionsTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *fakeSubmissionsTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (tx *fakeSubmissionsTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	if tx.execErr != nil {
		return pgconn.CommandTag{}, tx.execErr
	}
	if len(tx.execTags) == 0 {
		return pgconn.NewCommandTag("UPDATE 0"), nil
	}
	tag := tx.execTags[0]
	tx.execTags = tx.execTags[1:]
	return tag, nil
}
func (tx *fakeSubmissionsTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return &fakeSubmissionsRows{}, nil
}
func (tx *fakeSubmissionsTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return &fakeSubmissionsRow{err: pgx.ErrNoRows}
}
func (tx *fakeSubmissionsTx) Conn() *pgx.Conn { return nil }

type fakeSubmissionsRow struct {
	values []any
	err    error
}

func (r *fakeSubmissionsRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		assignSubmissionScanValue(dest[i], r.values[i])
	}
	return nil
}

type fakeSubmissionsRows struct {
	rows  [][]any
	index int
	err   error
}

func (r *fakeSubmissionsRows) Close()     {}
func (r *fakeSubmissionsRows) Err() error { return r.err }
func (r *fakeSubmissionsRows) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT")
}
func (r *fakeSubmissionsRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeSubmissionsRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}
func (r *fakeSubmissionsRows) Scan(dest ...any) error {
	values := r.rows[r.index-1]
	if len(dest) != len(values) {
		return errors.New("fake row destination count mismatch")
	}
	for i := range dest {
		assignSubmissionScanValue(dest[i], values[i])
	}
	return nil
}
func (r *fakeSubmissionsRows) Values() ([]any, error) { return r.rows[r.index-1], nil }
func (r *fakeSubmissionsRows) RawValues() [][]byte    { return nil }
func (r *fakeSubmissionsRows) Conn() *pgx.Conn        { return nil }

func assignSubmissionScanValue(dest any, value any) {
	target := reflect.ValueOf(dest).Elem()
	if value == nil {
		target.Set(reflect.Zero(target.Type()))
		return
	}
	source := reflect.ValueOf(value)
	if source.Type().AssignableTo(target.Type()) {
		target.Set(source)
		return
	}
	if source.Type().ConvertibleTo(target.Type()) {
		target.Set(source.Convert(target.Type()))
		return
	}
	panic("cannot assign " + source.Type().String() + " to " + target.Type().String())
}

func fakeSubmissionRowValues(sub *Submission) []any {
	return []any{
		sub.ID,
		sub.MatchID,
		sub.TaskID,
		sub.UserID,
		sub.Language,
		sub.LanguageVersion,
		sub.SourceCode,
		sub.Status,
		sub.Verdict,
		sub.JudgeSubmissionID,
		sub.JudgeStatus,
		sub.ErrorMessage,
		sub.PassedTests,
		sub.TotalTests,
		sub.CompileTimeMs,
		sub.RunTimeMs,
		sub.PeakMemoryKb,
		sub.Score,
		sub.ExitCode,
		sub.CompileLog,
		sub.StdoutTruncated,
		sub.StderrTruncated,
		sub.CreatedAt,
		sub.StartedAt,
		sub.FinishedAt,
		sub.UpdatedAt,
	}
}

func fakeCodeRunRowValues(run *CodeRun) []any {
	return []any{
		run.ID,
		run.TaskID,
		run.UserID,
		run.Language,
		run.LanguageVersion,
		run.SourceCode,
		run.Status,
		run.Verdict,
		run.JudgeSubmissionID,
		run.JudgeStatus,
		run.ErrorMessage,
		run.PassedTests,
		run.TotalTests,
		run.CompileTimeMs,
		run.RunTimeMs,
		run.PeakMemoryKb,
		run.Score,
		run.ExitCode,
		run.CompileLog,
		run.StdoutTruncated,
		run.StderrTruncated,
		run.CreatedAt,
		run.StartedAt,
		run.FinishedAt,
		run.UpdatedAt,
	}
}
