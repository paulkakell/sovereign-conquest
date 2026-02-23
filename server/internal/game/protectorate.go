package game

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	protectorateFraction    = 0.10
	protectorateMinFighters = 25
	protectorateMaxFighters = 200
)

// EnsureProtectorateSectors enforces that ~10% of sectors are marked as Galactic Protectorate space.
// It is safe to run on every startup.
func EnsureProtectorateSectors(ctx context.Context, pool *pgxpool.Pool, seed int64) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Collect all sector IDs.
	rows, err := tx.Query(ctx, "SELECT id FROM sectors ORDER BY id")
	if err != nil {
		return err
	}
	defer rows.Close()
	var all []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		all = append(all, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(all) == 0 {
		return tx.Commit(ctx)
	}

	target := int(math.Round(float64(len(all)) * protectorateFraction))
	if target < 1 {
		target = 1
	}

	// Existing protectorate sectors.
	protRows, err := tx.Query(ctx, "SELECT id FROM sectors WHERE is_protectorate=true ORDER BY id")
	if err != nil {
		return err
	}
	defer protRows.Close()
	protSet := map[int]struct{}{}
	var protIDs []int
	for protRows.Next() {
		var id int
		if err := protRows.Scan(&id); err != nil {
			return err
		}
		protSet[id] = struct{}{}
		protIDs = append(protIDs, id)
	}
	if err := protRows.Err(); err != nil {
		return err
	}

	need := target - len(protIDs)
	if need > 0 {
		// Pick additional sectors deterministically.
		candidates := make([]int, 0, len(all))
		for _, id := range all {
			if _, ok := protSet[id]; ok {
				continue
			}
			candidates = append(candidates, id)
		}

		rng := rand.New(rand.NewSource(seed + 0x53504350)) // "SPCP"
		rng.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
		if need > len(candidates) {
			need = len(candidates)
		}
		for i := 0; i < need; i++ {
			id := candidates[i]
			fighters := protectorateMinFighters + rng.Intn(protectorateMaxFighters-protectorateMinFighters+1)
			_, err := tx.Exec(ctx, "UPDATE sectors SET is_protectorate=true, protectorate_fighters=$2 WHERE id=$1", id, fighters)
			if err != nil {
				return err
			}
			protIDs = append(protIDs, id)
		}
	}

	// Ensure fighter counts and major ports exist for all protectorate sectors.
	// Fighters: if missing/zero, seed into range.
	_, err = tx.Exec(ctx, `
		UPDATE sectors
		SET protectorate_fighters = $2
		WHERE is_protectorate=true AND protectorate_fighters <= 0
	`, protectorateMinFighters, protectorateMinFighters)
	if err != nil {
		return err
	}

	for _, sectorID := range protIDs {
		if err := ensureProtectoratePort(ctx, tx, sectorID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func ensureProtectoratePort(ctx context.Context, tx pgx.Tx, sectorID int) error {
	// "Major port with all resources": make all commodities SELL, with large stock + regen.
	oreBaseQty := 12000
	orgBaseQty := 9000
	eqBaseQty := 7000

	oreBasePrice := 12
	orgBasePrice := 22
	eqBasePrice := 65

	oreRegen := max(200, oreBaseQty/30)
	orgRegen := max(150, orgBaseQty/30)
	eqRegen := max(120, eqBaseQty/30)

	// Start with full stock.
	oreQty := oreBaseQty
	orgQty := orgBaseQty
	eqQty := eqBaseQty

	_, err := tx.Exec(ctx, `
		INSERT INTO ports(
			sector_id,
			ore_mode, ore_qty, ore_base_qty, ore_base_price, ore_regen,
			organics_mode, organics_qty, organics_base_qty, organics_base_price, organics_regen,
			equipment_mode, equipment_qty, equipment_base_qty, equipment_base_price, equipment_regen
		) VALUES (
			$1,
			'SELL',$2,$3,$4,$5,
			'SELL',$6,$7,$8,$9,
			'SELL',$10,$11,$12,$13
		)
		ON CONFLICT (sector_id) DO UPDATE SET
			ore_mode='SELL', ore_base_qty=EXCLUDED.ore_base_qty, ore_base_price=EXCLUDED.ore_base_price, ore_regen=EXCLUDED.ore_regen,
			organics_mode='SELL', organics_base_qty=EXCLUDED.organics_base_qty, organics_base_price=EXCLUDED.organics_base_price, organics_regen=EXCLUDED.organics_regen,
			equipment_mode='SELL', equipment_base_qty=EXCLUDED.equipment_base_qty, equipment_base_price=EXCLUDED.equipment_base_price, equipment_regen=EXCLUDED.equipment_regen
	`,
		sectorID,
		oreQty, oreBaseQty, oreBasePrice, oreRegen,
		orgQty, orgBaseQty, orgBasePrice, orgRegen,
		eqQty, eqBaseQty, eqBasePrice, eqRegen,
	)
	if err != nil {
		return err
	}
	return nil
}

// StartProtectorateTicker fluctuates fighter counts in Protectorate sectors.
func StartProtectorateTicker(ctx context.Context, pool *pgxpool.Pool, tickSeconds int) {
	if tickSeconds <= 0 {
		return
	}
	if tickSeconds < 10 {
		tickSeconds = 10
	}
	go func() {
		t := time.NewTicker(time.Duration(tickSeconds) * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				// Randomize fighters into [min,max] each tick.
				_, _ = pool.Exec(ctx, `
					UPDATE sectors
					SET protectorate_fighters = ($1 + floor(random()*($2-$1+1)))::int
					WHERE is_protectorate=true
				`, protectorateMinFighters, protectorateMaxFighters)
			}
		}
	}()
}

func IsProtectorateSector(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, sectorID int) (bool, error) {
	var isProt bool
	err := q.QueryRow(ctx, "SELECT is_protectorate FROM sectors WHERE id=$1", sectorID).Scan(&isProt)
	if err != nil {
		return false, err
	}
	return isProt, nil
}

func ProtectorateSummary(sector SectorView) string {
	if !sector.IsProtectorate {
		return ""
	}
	return fmt.Sprintf("Protectorate space: %d fighters on patrol. Shipyard available.", sector.ProtectorateFighters)
}
