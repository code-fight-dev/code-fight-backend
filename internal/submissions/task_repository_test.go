package submissions

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestNewPGTaskRepository(t *testing.T) {
	if repo := NewPGTaskRepository(nil); repo == nil {
		t.Fatal("NewPGTaskRepository returned nil")
	}
}

func TestPGTaskRepositoryGetByID(t *testing.T) {
	taskID := uuid.New()
	repo := &PGTaskRepository{db: &fakeTaskDB{rows: []pgx.Row{
		&fakeSubmissionsRow{values: []any{taskID, true}},
		&fakeSubmissionsRow{err: pgx.ErrNoRows},
	}}}

	task, err := repo.GetByID(context.Background(), taskID)
	if err != nil || task == nil || task.ID != taskID || !task.IsActive {
		t.Fatalf("GetByID = %#v, %v", task, err)
	}
	task, err = repo.GetByID(context.Background(), uuid.New())
	if err != nil || task != nil {
		t.Fatalf("missing GetByID = %#v, %v", task, err)
	}
}

func TestPGTaskRepositoryIsLanguageAllowed(t *testing.T) {
	repo := &PGTaskRepository{db: &fakeTaskDB{rows: []pgx.Row{
		&fakeSubmissionsRow{values: []any{true}},
		&fakeSubmissionsRow{err: errors.New("boom")},
	}}}

	allowed, err := repo.IsLanguageAllowed(context.Background(), uuid.New(), "go", "1.24")
	if err != nil || !allowed {
		t.Fatalf("IsLanguageAllowed = %v, %v", allowed, err)
	}
	allowed, err = repo.IsLanguageAllowed(context.Background(), uuid.New(), "go", "1.24")
	if err == nil || allowed {
		t.Fatalf("IsLanguageAllowed error = %v/%v", allowed, err)
	}
}

type fakeTaskDB struct {
	rows []pgx.Row
}

func (db *fakeTaskDB) QueryRow(context.Context, string, ...any) pgx.Row {
	if len(db.rows) == 0 {
		return &fakeSubmissionsRow{err: pgx.ErrNoRows}
	}
	row := db.rows[0]
	db.rows = db.rows[1:]
	return row
}
