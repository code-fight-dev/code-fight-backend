package managementtx

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type beginFunc func(context.Context) (pgx.Tx, error)

func (f beginFunc) Begin(ctx context.Context) (pgx.Tx, error) { return f(ctx) }

type testTx struct {
	pgx.Tx
	exec     func(context.Context) error
	rollback func(context.Context) error
}

func (tx *testTx) Exec(ctx context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, tx.exec(ctx)
}

func (tx *testTx) Rollback(ctx context.Context) error { return tx.rollback(ctx) }

func TestBeginBoundsPoolAcquisitionAndHonorsEarlierDeadline(t *testing.T) {
	for _, earlier := range []bool{false, true} {
		t.Run(map[bool]string{false: "default", true: "earlier deadline"}[earlier], func(t *testing.T) {
			parent := context.Background()
			if earlier {
				var cancel context.CancelFunc
				parent, cancel = context.WithTimeout(parent, time.Second)
				defer cancel()
			}
			failure := errors.New("pool unavailable")
			var acquisitionCtx context.Context
			_, _, _, err := Begin(parent, beginFunc(func(ctx context.Context) (pgx.Tx, error) {
				acquisitionCtx = ctx
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > operationTimeout {
					t.Fatal("pool acquisition has no bounded operation deadline")
				}
				if parentDeadline, ok := parent.Deadline(); ok && !deadline.Equal(parentDeadline) {
					t.Fatal("caller deadline was extended")
				}
				return nil, failure
			}))
			if !errors.Is(err, failure) || acquisitionCtx.Err() != context.Canceled {
				t.Fatalf("failed acquisition was not cleaned up: %v, %v", err, acquisitionCtx.Err())
			}
		})
	}
}

func TestRollbackSurvivesCanceledOperation(t *testing.T) {
	for _, setupFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "operation canceled", true: "setup canceled"}[setupFailure], func(t *testing.T) {
			parent, cancel := context.WithCancel(context.Background())
			defer cancel()
			rolledBack := false
			tx := &testTx{
				exec: func(ctx context.Context) error {
					if setupFailure {
						cancel()
						return ctx.Err()
					}
					return nil
				},
				rollback: func(ctx context.Context) error {
					rolledBack = true
					deadline, ok := ctx.Deadline()
					if ctx.Err() != nil || !ok || time.Until(deadline) <= 0 || time.Until(deadline) > rollbackTimeout {
						t.Fatal("rollback needs a live, bounded context independent of the operation")
					}
					return nil
				},
			}
			ctx, _, cleanup, err := Begin(parent, beginFunc(func(context.Context) (pgx.Tx, error) { return tx, nil }))
			if setupFailure {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("setup error = %v", err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				cancel()
				cleanup()
				if ctx.Err() != context.Canceled {
					t.Fatal("operation context was not canceled")
				}
			}
			if !rolledBack {
				t.Fatal("transaction was not rolled back")
			}
		})
	}
}
