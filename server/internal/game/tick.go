package game

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func StartPortTicker(ctx context.Context, pool *pgxpool.Pool, tickSeconds int) {
	if tickSeconds < 5 {
		tickSeconds = 5
	}
	ticker := time.NewTicker(time.Duration(tickSeconds) * time.Second)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = pool.Exec(ctx, `
					UPDATE ports SET
						ore_qty = LEAST(ore_base_qty, ore_qty + ore_regen),
						organics_qty = LEAST(organics_base_qty, organics_qty + organics_regen),
						equipment_qty = LEAST(equipment_base_qty, equipment_qty + equipment_regen)
				`)
			}
		}
	}()
}

func StartPlanetTicker(ctx context.Context, pool *pgxpool.Pool, tickSeconds int) {
	if tickSeconds < 5 {
		tickSeconds = 5
	}
	ticker := time.NewTicker(time.Duration(tickSeconds) * time.Second)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = pool.Exec(ctx, `
					UPDATE planets SET
						storage_ore = LEAST(storage_max, storage_ore + production_ore),
						storage_organics = LEAST(storage_max, storage_organics + production_organics),
						storage_equipment = LEAST(storage_max, storage_equipment + production_equipment),
						last_produced = now()
				`)
			}
		}
	}()
}
