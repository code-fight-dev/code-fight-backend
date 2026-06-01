package tasks

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestNewRepository(t *testing.T) {
	if repo := NewRepository(nil); repo == nil {
		t.Fatal("NewRepository returned nil")
	}
}

func TestRepositoryListAndDetailWithFakeDB(t *testing.T) {
	now := time.Now().UTC()
	payload := `{
		"id":"payload-id",
		"title":"Payload Title",
		"summary":"summary",
		"description":["solve"],
		"tags":["arrays"],
		"category":"Algorithms",
		"kind":"algorithmic",
		"supportedLanguages":["go","python"],
		"starterCodeByLanguage":{"go":"package main"},
		"estimatedMinutes":12,
		"popularity":9,
		"progress":"locked"
	}`
	db := &fakeTasksDB{
		queryRows: []pgx.Rows{
			&fakeTasksRows{rows: [][]any{{
				"task-1",
				"two-sum",
				"DB Title",
				"easy",
				now,
				payload,
				[]string{"python", "go", "rust"},
				1,
				2,
				true,
				3,
			}}},
			&fakeTasksRows{rows: [][]any{{"go", "1.24"}, {"python", "3.12"}}},
			&fakeTasksRows{rows: [][]any{{1, "1 2", "3"}}},
		},
		rows: []pgx.Row{fakeTasksRow{values: []any{
			"task-1",
			"two-sum",
			"DB Title",
			"easy",
			now,
			payload,
			[]string{"python", "go", "rust"},
			1,
			2,
		}}},
	}
	repo := &Repository{db: db}

	items, err := repo.ListChallengesForUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListChallengesForUser returned error: %v", err)
	}
	if len(items) != 1 || items[0].ID != "payload-id" || items[0].Progress != "solved" ||
		items[0].Difficulty != "Easy" || items[0].AcceptanceRate != 50 ||
		len(items[0].SupportedLanguages) != 2 {
		t.Fatalf("items = %#v", items)
	}

	challenge, err := repo.GetChallengeBySlug(context.Background(), "two-sum")
	if err != nil {
		t.Fatalf("GetChallengeBySlug returned error: %v", err)
	}
	if challenge == nil || challenge.ID != "payload-id" ||
		challenge.LanguageVersions["go"] != "1.24" ||
		len(challenge.TestCases) != 1 ||
		challenge.AcceptedSubmissionRate != 50 {
		t.Fatalf("challenge = %#v", challenge)
	}
}

func TestRepositoryListChallengesWithoutUser(t *testing.T) {
	now := time.Now().UTC()
	payload := `{"id":"task-1","title":"Title","summary":"summary","supportedLanguages":["go"],"starterCodeByLanguage":{"go":"code"},"createdAt":"2026-01-01T00:00:00Z"}`
	repo := &Repository{db: &fakeTasksDB{queryRows: []pgx.Rows{&fakeTasksRows{rows: [][]any{{
		"task-1",
		"slug",
		"Title",
		"medium",
		now,
		payload,
		[]string{"go"},
		0,
		0,
		false,
		0,
	}}}}}}

	items, err := repo.ListChallenges(context.Background())
	if err != nil {
		t.Fatalf("ListChallenges returned error: %v", err)
	}
	if len(items) != 1 || items[0].Progress != "not-started" || items[0].Difficulty != "Medium" {
		t.Fatalf("items = %#v", items)
	}
}

func TestRepositoryGetChallengeByID(t *testing.T) {
	now := time.Now().UTC()
	payload := `{"id":"payload-id","title":"Payload Title","summary":"summary","supportedLanguages":["go"],"starterCodeByLanguage":{"go":"code"}}`
	repo := &Repository{db: &fakeTasksDB{
		rows: []pgx.Row{fakeTasksRow{values: []any{
			"task-1",
			"slug",
			"Title",
			"hard",
			now,
			payload,
			[]string{"go"},
			0,
			0,
		}}},
		queryRows: []pgx.Rows{
			&fakeTasksRows{},
			&fakeTasksRows{},
		},
	}}

	challenge, err := repo.GetChallengeByID(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("GetChallengeByID returned error: %v", err)
	}
	if challenge == nil || challenge.ID != "payload-id" || challenge.Difficulty != "Hard" {
		t.Fatalf("challenge = %#v", challenge)
	}
}

func TestRepositoryStatsAndSubmissionsWithFakeDB(t *testing.T) {
	now := time.Now().UTC()
	verdict := "accepted"
	db := &fakeTasksDB{
		rows: []pgx.Row{fakeTasksRow{values: []any{2, 1}}},
		queryRows: []pgx.Rows{&fakeTasksRows{rows: [][]any{{
			"sub-1",
			nil,
			"task-1",
			"user-1",
			"go",
			"1.24",
			"package main",
			"finished",
			&verdict,
			nil,
			nil,
			nil,
			1,
			1,
			nil,
			nil,
			nil,
			100,
			nil,
			nil,
			nil,
			nil,
			now,
			nil,
			nil,
			now,
			`[{"testCaseId":"case-1","status":"accepted"}]`,
		}}}},
	}
	repo := &Repository{db: db}

	stats, err := repo.GetUserTaskStats(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetUserTaskStats returned error: %v", err)
	}
	if stats.SolvedTasks != 2 || stats.InProgressTasks != 1 {
		t.Fatalf("stats = %#v", stats)
	}

	submissions, err := repo.ListUserTaskSubmissions(context.Background(), "task-1", "user-1")
	if err != nil {
		t.Fatalf("ListUserTaskSubmissions returned error: %v", err)
	}
	if len(submissions) != 1 || submissions[0].Verdict == nil || len(submissions[0].TestResults) != 1 {
		t.Fatalf("submissions = %#v", submissions)
	}
}

func TestRepositoryReturnsNilForMissingChallenge(t *testing.T) {
	repo := &Repository{db: &fakeTasksDB{rows: []pgx.Row{fakeTasksRow{err: pgx.ErrNoRows}}}}
	challenge, err := repo.GetChallengeByID(context.Background(), "missing")
	if err != nil {
		t.Fatalf("GetChallengeByID returned error: %v", err)
	}
	if challenge != nil {
		t.Fatalf("challenge = %#v, want nil", challenge)
	}
}

func TestRepositoryErrorBranches(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("db failed")
	now := time.Now().UTC()
	detailRow := fakeTasksRow{values: []any{
		"task-1",
		"slug",
		"Title",
		"medium",
		now,
		`{"supportedLanguages":["go"]}`,
		[]string{"go"},
		0,
		0,
	}}

	tests := []struct {
		name string
		repo *Repository
		call func(*Repository) error
	}{
		{
			name: "list query error",
			repo: &Repository{db: &fakeTasksDB{queryErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.ListChallenges(ctx)
				return err
			},
		},
		{
			name: "list scan error",
			repo: &Repository{db: &fakeTasksDB{queryRows: []pgx.Rows{
				&fakeTasksRows{rows: [][]any{{"too-few"}}},
			}}},
			call: func(repo *Repository) error {
				_, err := repo.ListChallengesForUser(ctx, "user-1")
				return err
			},
		},
		{
			name: "list rows error",
			repo: &Repository{db: &fakeTasksDB{queryRows: []pgx.Rows{
				&fakeTasksRows{err: dbErr},
			}}},
			call: func(repo *Repository) error {
				_, err := repo.ListChallenges(ctx)
				return err
			},
		},
		{
			name: "stats row error",
			repo: &Repository{db: &fakeTasksDB{rows: []pgx.Row{fakeTasksRow{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.GetUserTaskStats(ctx, "user-1")
				return err
			},
		},
		{
			name: "submissions query error",
			repo: &Repository{db: &fakeTasksDB{queryErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.ListUserTaskSubmissions(ctx, "task-1", "user-1")
				return err
			},
		},
		{
			name: "submissions scan error",
			repo: &Repository{db: &fakeTasksDB{queryRows: []pgx.Rows{
				&fakeTasksRows{rows: [][]any{{"too-few"}}},
			}}},
			call: func(repo *Repository) error {
				_, err := repo.ListUserTaskSubmissions(ctx, "task-1", "user-1")
				return err
			},
		},
		{
			name: "submissions rows error",
			repo: &Repository{db: &fakeTasksDB{queryRows: []pgx.Rows{
				&fakeTasksRows{err: dbErr},
			}}},
			call: func(repo *Repository) error {
				_, err := repo.ListUserTaskSubmissions(ctx, "task-1", "user-1")
				return err
			},
		},
		{
			name: "detail scan error",
			repo: &Repository{db: &fakeTasksDB{rows: []pgx.Row{fakeTasksRow{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.GetChallengeBySlug(ctx, "slug")
				return err
			},
		},
		{
			name: "language versions query error",
			repo: &Repository{db: &fakeTasksDB{
				rows:     []pgx.Row{detailRow},
				queryErr: dbErr,
			}},
			call: func(repo *Repository) error {
				_, err := repo.GetChallengeBySlug(ctx, "slug")
				return err
			},
		},
		{
			name: "language versions scan error",
			repo: &Repository{db: &fakeTasksDB{
				rows: []pgx.Row{detailRow},
				queryRows: []pgx.Rows{
					&fakeTasksRows{rows: [][]any{{"go"}}},
				},
			}},
			call: func(repo *Repository) error {
				_, err := repo.GetChallengeByID(ctx, "task-1")
				return err
			},
		},
		{
			name: "language versions rows error",
			repo: &Repository{db: &fakeTasksDB{
				rows: []pgx.Row{detailRow},
				queryRows: []pgx.Rows{
					&fakeTasksRows{err: dbErr},
				},
			}},
			call: func(repo *Repository) error {
				_, err := repo.GetChallengeByID(ctx, "task-1")
				return err
			},
		},
		{
			name: "sample tests query error",
			repo: &Repository{db: &fakeTasksDB{
				rows: []pgx.Row{detailRow},
				queryRows: []pgx.Rows{
					&fakeTasksRows{rows: [][]any{{"go", "1.24"}}},
				},
				queryErrAt: 2,
				queryErr:   dbErr,
			}},
			call: func(repo *Repository) error {
				_, err := repo.GetChallengeBySlug(ctx, "slug")
				return err
			},
		},
		{
			name: "sample tests scan error",
			repo: &Repository{db: &fakeTasksDB{
				rows: []pgx.Row{detailRow},
				queryRows: []pgx.Rows{
					&fakeTasksRows{rows: [][]any{{"go", "1.24"}}},
					&fakeTasksRows{rows: [][]any{{1, "only-input"}}},
				},
			}},
			call: func(repo *Repository) error {
				_, err := repo.GetChallengeByID(ctx, "task-1")
				return err
			},
		},
		{
			name: "sample tests rows error",
			repo: &Repository{db: &fakeTasksDB{
				rows: []pgx.Row{detailRow},
				queryRows: []pgx.Rows{
					&fakeTasksRows{rows: [][]any{{"go", "1.24"}}},
					&fakeTasksRows{err: dbErr},
				},
			}},
			call: func(repo *Repository) error {
				_, err := repo.GetChallengeByID(ctx, "task-1")
				return err
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

type fakeTasksDB struct {
	rows       []pgx.Row
	queryRows  []pgx.Rows
	queryErr   error
	queryErrAt int
	queryCalls int
}

func (db *fakeTasksDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (db *fakeTasksDB) QueryRow(context.Context, string, ...any) pgx.Row {
	if len(db.rows) == 0 {
		return fakeTasksRow{err: pgx.ErrNoRows}
	}
	row := db.rows[0]
	db.rows = db.rows[1:]
	return row
}

func (db *fakeTasksDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	db.queryCalls++
	if db.queryErr != nil && (db.queryErrAt == 0 || db.queryCalls == db.queryErrAt) {
		return nil, db.queryErr
	}
	if len(db.queryRows) == 0 {
		return &fakeTasksRows{}, nil
	}
	rows := db.queryRows[0]
	db.queryRows = db.queryRows[1:]
	return rows, nil
}

type fakeTasksRows struct {
	rows [][]any
	idx  int
	err  error
}

func (r *fakeTasksRows) Close()                                       {}
func (r *fakeTasksRows) Err() error                                   { return r.err }
func (r *fakeTasksRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("SELECT 1") }
func (r *fakeTasksRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeTasksRows) Next() bool {
	if r.idx >= len(r.rows) {
		return false
	}
	r.idx++
	return true
}
func (r *fakeTasksRows) Scan(dest ...any) error { return scanFakeTasksValues(r.rows[r.idx-1], dest...) }
func (r *fakeTasksRows) Values() ([]any, error) { return r.rows[r.idx-1], nil }
func (r *fakeTasksRows) RawValues() [][]byte    { return nil }
func (r *fakeTasksRows) Conn() *pgx.Conn        { return nil }

type fakeTasksRow struct {
	values []any
	err    error
}

func (r fakeTasksRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	return scanFakeTasksValues(r.values, dest...)
}

func scanFakeTasksValues(values []any, dest ...any) error {
	if len(values) != len(dest) {
		return errors.New("fake row destination count mismatch")
	}
	for i := range dest {
		target := reflect.ValueOf(dest[i])
		if target.Kind() != reflect.Pointer || target.IsNil() {
			return errors.New("fake row destination must be pointer")
		}
		value := reflect.ValueOf(values[i])
		elem := target.Elem()
		if !value.IsValid() {
			elem.Set(reflect.Zero(elem.Type()))
			continue
		}
		if value.Type().AssignableTo(elem.Type()) {
			elem.Set(value)
			continue
		}
		if value.Type().ConvertibleTo(elem.Type()) {
			elem.Set(value.Convert(elem.Type()))
			continue
		}
		return errors.New("fake row value type mismatch")
	}
	return nil
}
