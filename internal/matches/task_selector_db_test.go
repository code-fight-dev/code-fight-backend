package matches

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestDBTaskSelectorSelectForHardMode(t *testing.T) {
	taskID := uuid.New()
	selector := &DBTaskSelector{db: &fakeTaskSelectorDB{rows: []pgx.Row{
		&fakePGRow{values: []any{taskID, TaskDifficultyHard}},
	}}}

	selected, err := selector.SelectForMatch(context.Background(), TaskModeHard)
	if err != nil {
		t.Fatalf("SelectForMatch returned error: %v", err)
	}
	if selected.TaskID != taskID || selected.Difficulty != TaskDifficultyHard {
		t.Fatalf("selected = %#v", selected)
	}
}

func TestDBTaskSelectorSelectForHardModeNoTasks(t *testing.T) {
	selector := &DBTaskSelector{db: &fakeTaskSelectorDB{rows: []pgx.Row{&fakePGRow{err: pgx.ErrNoRows}}}}

	_, err := selector.SelectForMatch(context.Background(), TaskModeHard)
	if !errors.Is(err, ErrNoTasksAvailable) {
		t.Fatalf("error = %v, want ErrNoTasksAvailable", err)
	}
}

func TestDBTaskSelectorSelectForNormalModeFallsBack(t *testing.T) {
	taskID := uuid.New()
	selector := &DBTaskSelector{db: &fakeTaskSelectorDB{rows: []pgx.Row{
		&fakePGRow{err: pgx.ErrNoRows},
		&fakePGRow{err: pgx.ErrNoRows},
		&fakePGRow{values: []any{taskID, TaskDifficultyHard}},
	}}}

	selected, err := selector.SelectForMatch(context.Background(), TaskModeNormal)
	if err != nil {
		t.Fatalf("SelectForMatch returned error: %v", err)
	}
	if selected.TaskID != taskID || selected.Difficulty != TaskDifficultyHard {
		t.Fatalf("selected = %#v", selected)
	}
}

func TestDBTaskSelectorSelectForMatchRejectsInvalidMode(t *testing.T) {
	selector := &DBTaskSelector{db: &fakeTaskSelectorDB{}}

	_, err := selector.SelectForMatch(context.Background(), TaskMode("turbo"))
	if !errors.Is(err, ErrInvalidTaskMode) {
		t.Fatalf("error = %v, want ErrInvalidTaskMode", err)
	}
}

type fakeTaskSelectorDB struct {
	rows []pgx.Row
}

func (db *fakeTaskSelectorDB) QueryRow(context.Context, string, ...any) pgx.Row {
	if len(db.rows) == 0 {
		return &fakePGRow{err: pgx.ErrNoRows}
	}
	row := db.rows[0]
	db.rows = db.rows[1:]
	return row
}
