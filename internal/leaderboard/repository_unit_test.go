package leaderboard

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestNewRepository(t *testing.T) {
	if repo := NewRepository(nil); repo == nil {
		t.Fatal("NewRepository returned nil")
	}
}

func TestRepositoryListViewerRankAndStatsWithFakeDB(t *testing.T) {
	userID := uuid.New()
	entryValues := []any{
		1,
		2,
		userID,
		"ada",
		"",
		"https://avatar",
		"Germany",
		"DE",
		1800,
		10,
		7,
		2,
		1,
	}
	db := &fakeLeaderboardDB{
		rows: []pgx.Rows{&fakeLeaderboardRows{rows: [][]any{entryValues}}},
		row:  fakeLeaderboardRow{values: entryValues},
		statsRow: fakeLeaderboardRow{values: []any{
			3,
			4,
			5,
			6,
			2.0,
		}},
	}
	repo := &Repository{db: db}

	items, total, err := repo.List(context.Background(), "global", 50, 0)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 || total != 2 || items[0].DisplayName != "ada" || items[0].WinRate != 0.7 {
		t.Fatalf("items=%#v total=%d", items, total)
	}
	if !strings.Contains(db.queries[0], "order by rank asc") {
		t.Fatalf("list query = %s", db.queries[0])
	}

	viewerRank, err := repo.GetViewerRank(context.Background(), "global", userID)
	if err != nil {
		t.Fatalf("GetViewerRank returned error: %v", err)
	}
	if viewerRank == nil || viewerRank.UserID != userID {
		t.Fatalf("viewer rank = %#v", viewerRank)
	}

	stats, err := repo.GetStats(context.Background(), "global")
	if err != nil {
		t.Fatalf("GetStats returned error: %v", err)
	}
	if stats.QueuedPlayers != 3 || stats.MatchesTodayGrowthPercent != 200 || stats.CachedAt.IsZero() {
		t.Fatalf("stats = %#v", stats)
	}
}

func TestRepositoryListCountsWhenPageEmpty(t *testing.T) {
	db := &fakeLeaderboardDB{
		rows:     []pgx.Rows{&fakeLeaderboardRows{}},
		countRow: fakeLeaderboardRow{values: []any{42}},
	}
	repo := &Repository{db: db}

	items, total, err := repo.List(context.Background(), "global", 50, 0)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 0 || total != 42 {
		t.Fatalf("items=%#v total=%d", items, total)
	}
}

func TestRepositoryGetViewerRankReturnsNilForMissingUser(t *testing.T) {
	repo := &Repository{db: &fakeLeaderboardDB{row: fakeLeaderboardRow{err: pgx.ErrNoRows}}}

	entry, err := repo.GetViewerRank(context.Background(), "global", uuid.New())
	if err != nil {
		t.Fatalf("GetViewerRank returned error: %v", err)
	}
	if entry != nil {
		t.Fatalf("entry = %#v, want nil", entry)
	}
}

func TestScanEntryPropagatesErrors(t *testing.T) {
	if _, _, err := scanEntry(fakeLeaderboardRow{err: errors.New("scan failed")}); err == nil {
		t.Fatal("scanEntry returned nil error")
	}
}

type fakeLeaderboardDB struct {
	rows     []pgx.Rows
	row      pgx.Row
	countRow pgx.Row
	statsRow pgx.Row
	queries  []string
}

func (db *fakeLeaderboardDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (db *fakeLeaderboardDB) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	db.queries = append(db.queries, sql)
	if len(db.rows) == 0 {
		return &fakeLeaderboardRows{}, nil
	}
	rows := db.rows[0]
	db.rows = db.rows[1:]
	return rows, nil
}

func (db *fakeLeaderboardDB) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	if strings.Contains(sql, "count(*)") && db.countRow != nil {
		return db.countRow
	}
	if strings.Contains(sql, "queued_players") && db.statsRow != nil {
		return db.statsRow
	}
	if db.row != nil {
		return db.row
	}
	return fakeLeaderboardRow{err: pgx.ErrNoRows}
}

type fakeLeaderboardRows struct {
	rows [][]any
	idx  int
	err  error
}

func (r *fakeLeaderboardRows) Close()                                       {}
func (r *fakeLeaderboardRows) Err() error                                   { return r.err }
func (r *fakeLeaderboardRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("SELECT 1") }
func (r *fakeLeaderboardRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeLeaderboardRows) Next() bool {
	if r.idx >= len(r.rows) {
		return false
	}
	r.idx++
	return true
}
func (r *fakeLeaderboardRows) Scan(dest ...any) error {
	return scanFakeLeaderboardValues(r.rows[r.idx-1], dest...)
}
func (r *fakeLeaderboardRows) Values() ([]any, error) { return r.rows[r.idx-1], nil }
func (r *fakeLeaderboardRows) RawValues() [][]byte    { return nil }
func (r *fakeLeaderboardRows) Conn() *pgx.Conn        { return nil }

type fakeLeaderboardRow struct {
	values []any
	err    error
}

func (r fakeLeaderboardRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	return scanFakeLeaderboardValues(r.values, dest...)
}

func scanFakeLeaderboardValues(values []any, dest ...any) error {
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
