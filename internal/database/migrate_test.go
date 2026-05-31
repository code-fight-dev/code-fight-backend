package database

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMigrateAppliesPendingMigrations(t *testing.T) {
	db := &fakeMigrateDB{}

	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}
	if db.beginCalls == 0 || db.tx.commits != db.beginCalls {
		t.Fatalf("begin=%d commits=%d", db.beginCalls, db.tx.commits)
	}
}

func TestMigrateSkipsAlreadyAppliedMigrations(t *testing.T) {
	db := &fakeMigrateDB{allApplied: true}

	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}
	if db.beginCalls != 0 {
		t.Fatalf("begin calls = %d, want 0", db.beginCalls)
	}
}

func TestMigrateReturnsApplyError(t *testing.T) {
	db := &fakeMigrateDB{tx: &fakeMigrateTx{execErrAt: 1}}

	if err := Migrate(context.Background(), db); err == nil {
		t.Fatal("Migrate returned nil error")
	}
	if db.tx.rollbacks == 0 {
		t.Fatal("expected rollback on failed migration")
	}
}

func TestMigrateReturnsSchemaTableError(t *testing.T) {
	db := &fakeMigrateDB{execErr: errors.New("create failed")}

	if err := Migrate(context.Background(), db); err == nil {
		t.Fatal("Migrate returned nil error")
	}
}

func TestMigrateReturnsBeginError(t *testing.T) {
	db := &fakeMigrateDB{beginErr: errors.New("begin failed")}

	if err := Migrate(context.Background(), db); err == nil {
		t.Fatal("Migrate returned nil error")
	}
}

func TestMigrateReturnsRecordError(t *testing.T) {
	db := &fakeMigrateDB{tx: &fakeMigrateTx{execErrAt: 2}}

	if err := Migrate(context.Background(), db); err == nil {
		t.Fatal("Migrate returned nil error")
	}
	if db.tx.rollbacks == 0 {
		t.Fatal("expected rollback on failed migration record")
	}
}

func TestMigrateReturnsCommitError(t *testing.T) {
	db := &fakeMigrateDB{tx: &fakeMigrateTx{commitErr: errors.New("commit failed")}}

	if err := Migrate(context.Background(), db); err == nil {
		t.Fatal("Migrate returned nil error")
	}
}

func TestMigrationAppliedMapsScanError(t *testing.T) {
	db := &fakeMigrateDB{rowErr: errors.New("boom")}

	if _, err := migrationApplied(context.Background(), db, "001.sql"); err == nil {
		t.Fatal("migrationApplied returned nil error")
	}
}

type fakeMigrateDB struct {
	allApplied bool
	execErr    error
	rowErr     error
	beginErr   error
	beginCalls int
	tx         *fakeMigrateTx
}

func (db *fakeMigrateDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	if db.execErr != nil {
		return pgconn.CommandTag{}, db.execErr
	}
	return pgconn.NewCommandTag("CREATE TABLE"), nil
}

func (db *fakeMigrateDB) QueryRow(context.Context, string, ...any) pgx.Row {
	if db.rowErr != nil {
		return &fakeMigrateRow{err: db.rowErr}
	}
	return &fakeMigrateRow{values: []any{db.allApplied}}
}

func (db *fakeMigrateDB) Begin(context.Context) (pgx.Tx, error) {
	db.beginCalls++
	if db.beginErr != nil {
		return nil, db.beginErr
	}
	if db.tx == nil {
		db.tx = &fakeMigrateTx{}
	}
	return db.tx, nil
}

type fakeMigrateTx struct {
	execCalls int
	execErrAt int
	commitErr error
	commits   int
	rollbacks int
}

func (tx *fakeMigrateTx) Begin(context.Context) (pgx.Tx, error) { return tx, nil }
func (tx *fakeMigrateTx) Commit(context.Context) error {
	tx.commits++
	if tx.commitErr != nil {
		return tx.commitErr
	}
	return nil
}
func (tx *fakeMigrateTx) Rollback(context.Context) error {
	tx.rollbacks++
	return nil
}
func (tx *fakeMigrateTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (tx *fakeMigrateTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (tx *fakeMigrateTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *fakeMigrateTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (tx *fakeMigrateTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	tx.execCalls++
	if tx.execErrAt > 0 && tx.execCalls == tx.execErrAt {
		return pgconn.CommandTag{}, errors.New("apply failed")
	}
	return pgconn.NewCommandTag("OK"), nil
}
func (tx *fakeMigrateTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (tx *fakeMigrateTx) QueryRow(context.Context, string, ...any) pgx.Row        { return &fakeMigrateRow{} }
func (tx *fakeMigrateTx) Conn() *pgx.Conn                                         { return nil }

type fakeMigrateRow struct {
	values []any
	err    error
}

func (r *fakeMigrateRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		target := reflect.ValueOf(dest[i]).Elem()
		value := reflect.ValueOf(r.values[i])
		target.Set(value)
	}
	return nil
}
