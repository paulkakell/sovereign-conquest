package game

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// runScheduledTask starts a transaction, attempts a transaction-scoped
// advisory lock, and executes work only on the replica that obtained it.
func runScheduledTask(
	ctx context.Context,
	pool *pgxpool.Pool,
	lockKey int64,
	work func(context.Context, pgx.Tx) error,
) (bool, error) {
	if pool == nil {
		return false, fmt.Errorf("database pool is required")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var acquired bool
	if err := tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", lockKey).Scan(&acquired); err != nil {
		return false, err
	}
	if !acquired {
		return false, nil
	}

	if err := work(ctx, tx); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
