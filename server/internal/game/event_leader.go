package game

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const eventLeaderLockKey int64 = 77252003

func withEventLeaderLock(ctx context.Context, pool *pgxpool.Pool, work func() error) (bool, error) {
	if pool == nil {
		return false, fmt.Errorf("database pool is required")
	}
	connection, err := pool.Acquire(ctx)
	if err != nil {
		return false, err
	}
	defer connection.Release()

	var acquired bool
	if err := connection.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", eventLeaderLockKey).Scan(&acquired); err != nil {
		return false, err
	}
	if !acquired {
		return false, nil
	}
	defer func() {
		unlockContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = connection.Exec(unlockContext, "SELECT pg_advisory_unlock($1)", eventLeaderLockKey)
	}()

	if err := work(); err != nil {
		return false, err
	}
	return true, nil
}

func StartEventTickerLeader(ctx context.Context, pool *pgxpool.Pool, tickSeconds int) {
	if tickSeconds <= 0 {
		slog.Info("scheduled task disabled", "task", "event_generation")
		return
	}
	if tickSeconds < 10 {
		tickSeconds = 10
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	go func() {
		ticker := time.NewTicker(time.Duration(tickSeconds) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, err := withEventLeaderLock(ctx, pool, func() error {
					if _, err := pool.Exec(ctx, "UPDATE events SET active=false WHERE active=true AND ends_at <= now()"); err != nil {
						return err
					}
					var activeCount int
					if err := pool.QueryRow(ctx, "SELECT COUNT(1) FROM events WHERE active=true").Scan(&activeCount); err != nil {
						return err
					}
					if activeCount >= 5 || rng.Float64() > 0.35 {
						return nil
					}
					return createRandomEvent(ctx, pool, rng)
				})
				if err != nil {
					slog.Error("scheduled task failed", "task", "event_generation", "error", err)
				}
			}
		}
	}()
}
