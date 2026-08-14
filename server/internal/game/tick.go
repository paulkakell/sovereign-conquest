package game

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	portTickerLockKey   int64 = 77252001
	planetTickerLockKey int64 = 77252002
)

func StartPortTicker(ctx context.Context, pool *pgxpool.Pool, tickSeconds int) {
	startScheduledTicker(ctx, "port_regeneration", pool, portTickerLockKey, tickSeconds, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE ports SET
				ore_qty = LEAST(ore_base_qty, ore_qty + ore_regen),
				organics_qty = LEAST(organics_base_qty, organics_qty + organics_regen),
				equipment_qty = LEAST(equipment_base_qty, equipment_qty + equipment_regen)
		`)
		return err
	})
}

func StartPlanetTicker(ctx context.Context, pool *pgxpool.Pool, tickSeconds int) {
	startScheduledTicker(ctx, "planet_production", pool, planetTickerLockKey, tickSeconds, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE planets SET
				storage_ore = LEAST(storage_max, storage_ore + production_ore),
				storage_organics = LEAST(storage_max, storage_organics + production_organics),
				storage_equipment = LEAST(storage_max, storage_equipment + production_equipment),
				last_produced = now()
		`)
		return err
	})
}

func startScheduledTicker(
	ctx context.Context,
	name string,
	pool *pgxpool.Pool,
	lockKey int64,
	tickSeconds int,
	work func(context.Context, pgx.Tx) error,
) {
	if tickSeconds <= 0 {
		slog.Info("scheduled task disabled", "task", name)
		return
	}
	if tickSeconds < 5 {
		tickSeconds = 5
	}

	go func() {
		ticker := time.NewTicker(time.Duration(tickSeconds) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				executed, err := runScheduledTask(ctx, pool, lockKey, work)
				if err != nil {
					slog.Error("scheduled task failed", "task", name, "error", err)
					continue
				}
				if executed {
					slog.Debug("scheduled task completed", "task", name)
				}
			}
		}
	}()
}
