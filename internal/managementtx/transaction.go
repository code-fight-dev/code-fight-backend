package managementtx

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	operationTimeout = 10 * time.Second
	rollbackTimeout  = 3 * time.Second
)

type beginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

func Begin(ctx context.Context, db beginner) (context.Context, pgx.Tx, func(), error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	tx, err := db.Begin(ctx)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	cleanup := func() {
		defer cancel()
		rollbackCtx, rollbackCancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
		defer rollbackCancel()
		_ = tx.Rollback(rollbackCtx)
	}
	if _, err := tx.Exec(ctx, `set local lock_timeout = '2s'; set local statement_timeout = '10s'`); err != nil {
		cleanup()
		return nil, nil, nil, err
	}
	return ctx, tx, cleanup, nil
}
