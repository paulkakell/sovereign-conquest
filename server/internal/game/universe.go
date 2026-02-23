package game

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UniverseConfig struct {
	Seed    int64
	Sectors int
}

func EnsureUniverse(ctx context.Context, pool *pgxpool.Pool, cfg UniverseConfig) error {
	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(1) FROM sectors").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if cfg.Sectors < 20 {
		cfg.Sectors = 20
	}
	rng := rand.New(rand.NewSource(cfg.Seed))

	// Insert sectors
	batch := &pgx.Batch{}
	for i := 1; i <= cfg.Sectors; i++ {
		name := fmt.Sprintf("Sector %d", i)
		batch.Queue("INSERT INTO sectors(id, name) VALUES ($1, $2)", i, name)
	}
	br := pool.SendBatch(ctx, batch)
	if err := br.Close(); err != nil {
		return err
	}

	// Create a ring for guaranteed connectivity (bidirectional)
	warpBatch := &pgx.Batch{}
	for i := 1; i <= cfg.Sectors; i++ {
		next := i + 1
		if next > cfg.Sectors {
			next = 1
		}
		warpBatch.Queue("INSERT INTO warps(from_sector, to_sector, one_way) VALUES ($1,$2,false) ON CONFLICT DO NOTHING", i, next)
		warpBatch.Queue("INSERT INTO warps(from_sector, to_sector, one_way) VALUES ($1,$2,false) ON CONFLICT DO NOTHING", next, i)
	}
	// Add random extra connections
	for i := 1; i <= cfg.Sectors; i++ {
		extra := 2 + rng.Intn(3) // 2..4 extra edges
		for j := 0; j < extra; j++ {
			to := 1 + rng.Intn(cfg.Sectors)
			if to == i {
				continue
			}
			warpBatch.Queue("INSERT INTO warps(from_sector, to_sector, one_way) VALUES ($1,$2,false) ON CONFLICT DO NOTHING", i, to)
			warpBatch.Queue("INSERT INTO warps(from_sector, to_sector, one_way) VALUES ($1,$2,false) ON CONFLICT DO NOTHING", to, i)
		}
	}
	wbr := pool.SendBatch(ctx, warpBatch)
	if err := wbr.Close(); err != nil {
		return err
	}

	// Insert ports for a subset of sectors
	portBatch := &pgx.Batch{}
	for i := 1; i <= cfg.Sectors; i++ {
		if rng.Float64() > 0.60 {
			continue
		}

		oreMode := pickMode(rng)
		orgMode := pickMode(rng)
		eqMode := pickMode(rng)

		// Ensure at least one BUY and one SELL across commodities
		if oreMode == orgMode && orgMode == eqMode {
			eqMode = flipMode(eqMode)
		}

		oreBaseQty := 4000 + rng.Intn(4001) // 4000..8000
		orgBaseQty := 2000 + rng.Intn(3001) // 2000..5000
		eqBaseQty := 1500 + rng.Intn(2501)  // 1500..4000

		oreBasePrice := 8 + rng.Intn(7)   // 8..14
		orgBasePrice := 15 + rng.Intn(16) // 15..30
		eqBasePrice := 40 + rng.Intn(41)  // 40..80

		oreRegen := max(50, oreBaseQty/25) // ~4%% per tick
		orgRegen := max(30, orgBaseQty/25)
		eqRegen := max(20, eqBaseQty/25)

		oreQty := initialQty(oreMode, oreBaseQty)
		orgQty := initialQty(orgMode, orgBaseQty)
		eqQty := initialQty(eqMode, eqBaseQty)

		portBatch.Queue(
			`INSERT INTO ports(
				sector_id,
				ore_mode, ore_qty, ore_base_qty, ore_base_price, ore_regen,
				organics_mode, organics_qty, organics_base_qty, organics_base_price, organics_regen,
				equipment_mode, equipment_qty, equipment_base_qty, equipment_base_price, equipment_regen
			) VALUES (
				$1,
				$2,$3,$4,$5,$6,
				$7,$8,$9,$10,$11,
				$12,$13,$14,$15,$16
			)`,
			i,
			oreMode, oreQty, oreBaseQty, oreBasePrice, oreRegen,
			orgMode, orgQty, orgBaseQty, orgBasePrice, orgRegen,
			eqMode, eqQty, eqBaseQty, eqBasePrice, eqRegen,
		)
	}
	pbr := pool.SendBatch(ctx, portBatch)
	if err := pbr.Close(); err != nil {
		return err
	}

	// Insert unowned planets for a subset of sectors (players can also colonize later).
	planetBatch := &pgx.Batch{}
	for i := 1; i <= cfg.Sectors; i++ {
		if rng.Float64() > 0.35 {
			continue
		}
		name := fmt.Sprintf("Planet %d", i)
		prodOre := 10 + rng.Intn(21)        // 10..30 per tick
		prodOrg := 5 + rng.Intn(16)         // 5..20 per tick
		prodEq := 2 + rng.Intn(9)           // 2..10 per tick
		storageMax := 2000 + rng.Intn(3001) // 2000..5000 per commodity
		planetBatch.Queue(
			`INSERT INTO planets(sector_id, name, production_ore, production_organics, production_equipment, storage_max)
			 VALUES ($1,$2,$3,$4,$5,$6)`,
			i, name, prodOre, prodOrg, prodEq, storageMax,
		)
	}
	plbr := pool.SendBatch(ctx, planetBatch)
	if err := plbr.Close(); err != nil {
		return err
	}

	// Small delay so the next immediate read sees committed data in some clients (mostly cosmetic)
	time.Sleep(50 * time.Millisecond)

	return nil
}

func pickMode(rng *rand.Rand) string {
	if rng.Intn(2) == 0 {
		return "BUY"
	}
	return "SELL"
}

func flipMode(m string) string {
	if m == "BUY" {
		return "SELL"
	}
	return "BUY"
}

func initialQty(mode string, baseQty int) int {
	if mode == "SELL" {
		return baseQty
	}
	q := baseQty / 4
	if q < 1 {
		q = 1
	}
	return q
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
