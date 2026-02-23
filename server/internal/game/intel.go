package game

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type PortIntel struct {
	SectorID   int
	SectorName string
	ScannedAt  time.Time

	OreMode    string
	OreQty     int
	OreBaseQty int
	OrePrice   int

	OrganicsMode    string
	OrganicsQty     int
	OrganicsBaseQty int
	OrganicsPrice   int

	EquipmentMode    string
	EquipmentQty     int
	EquipmentBaseQty int
	EquipmentPrice   int
}

func CaptureScanIntel(ctx context.Context, tx pgx.Tx, playerID string, sectorID int) error {
	// Capture a port snapshot only if the sector has a port.
	var port portForUpdate
	err := tx.QueryRow(ctx, `
		SELECT
			ore_mode, ore_qty, ore_base_qty, ore_base_price,
			organics_mode, organics_qty, organics_base_qty, organics_base_price,
			equipment_mode, equipment_qty, equipment_base_qty, equipment_base_price
		FROM ports
		WHERE sector_id = $1
	`, sectorID).Scan(
		&port.OreMode, &port.OreQty, &port.OreBaseQty, &port.OreBasePrice,
		&port.OrganicsMode, &port.OrganicsQty, &port.OrganicsBaseQty, &port.OrganicsBasePrice,
		&port.EquipmentMode, &port.EquipmentQty, &port.EquipmentBaseQty, &port.EquipmentBasePrice,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		// No port: remove any previous intel.
		_, _ = tx.Exec(ctx, `DELETE FROM player_sector_intel WHERE player_id=$1 AND sector_id=$2`, playerID, sectorID)
		return nil
	}
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	// Apply any active event price modifiers in this sector.
	oePct := 100
	orgPct := 100
	eqPct := 100
	if ev, ok, err := LoadActiveEvent(ctx, tx, sectorID); err != nil {
		return err
	} else if ok {
		oePct = pricePercentForCommodity(ev, "ORE")
		orgPct = pricePercentForCommodity(ev, "ORGANICS")
		eqPct = pricePercentForCommodity(ev, "EQUIPMENT")
	}

	oePrice := PricePerUnitWithPercent(port.OreBasePrice, port.OreBaseQty, port.OreQty, oePct)
	orgPrice := PricePerUnitWithPercent(port.OrganicsBasePrice, port.OrganicsBaseQty, port.OrganicsQty, orgPct)
	eqPrice := PricePerUnitWithPercent(port.EquipmentBasePrice, port.EquipmentBaseQty, port.EquipmentQty, eqPct)

	_, err = tx.Exec(ctx, `
		INSERT INTO player_sector_intel (
			player_id, sector_id, scanned_at,
			ore_mode, ore_qty, ore_base_qty, ore_price,
			organics_mode, organics_qty, organics_base_qty, organics_price,
			equipment_mode, equipment_qty, equipment_base_qty, equipment_price
		) VALUES (
			$1,$2,$3,
			$4,$5,$6,$7,
			$8,$9,$10,$11,
			$12,$13,$14,$15
		)
		ON CONFLICT (player_id, sector_id) DO UPDATE SET
			scanned_at = EXCLUDED.scanned_at,
			ore_mode = EXCLUDED.ore_mode,
			ore_qty = EXCLUDED.ore_qty,
			ore_base_qty = EXCLUDED.ore_base_qty,
			ore_price = EXCLUDED.ore_price,
			organics_mode = EXCLUDED.organics_mode,
			organics_qty = EXCLUDED.organics_qty,
			organics_base_qty = EXCLUDED.organics_base_qty,
			organics_price = EXCLUDED.organics_price,
			equipment_mode = EXCLUDED.equipment_mode,
			equipment_qty = EXCLUDED.equipment_qty,
			equipment_base_qty = EXCLUDED.equipment_base_qty,
			equipment_price = EXCLUDED.equipment_price
	`,
		playerID, sectorID, now,
		port.OreMode, port.OreQty, port.OreBaseQty, oePrice,
		port.OrganicsMode, port.OrganicsQty, port.OrganicsBaseQty, orgPrice,
		port.EquipmentMode, port.EquipmentQty, port.EquipmentBaseQty, eqPrice,
	)
	return err
}

func LoadPortIntel(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, playerID string) ([]PortIntel, error) {
	rows, err := q.Query(ctx, `
		SELECT
			psi.sector_id,
			s.name,
			psi.scanned_at,
			psi.ore_mode, psi.ore_qty, psi.ore_base_qty, psi.ore_price,
			psi.organics_mode, psi.organics_qty, psi.organics_base_qty, psi.organics_price,
			psi.equipment_mode, psi.equipment_qty, psi.equipment_base_qty, psi.equipment_price
		FROM player_sector_intel psi
		JOIN sectors s ON s.id = psi.sector_id
		WHERE psi.player_id = $1
		ORDER BY psi.scanned_at DESC
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]PortIntel, 0, 32)
	for rows.Next() {
		var pi PortIntel
		if err := rows.Scan(
			&pi.SectorID,
			&pi.SectorName,
			&pi.ScannedAt,
			&pi.OreMode, &pi.OreQty, &pi.OreBaseQty, &pi.OrePrice,
			&pi.OrganicsMode, &pi.OrganicsQty, &pi.OrganicsBaseQty, &pi.OrganicsPrice,
			&pi.EquipmentMode, &pi.EquipmentQty, &pi.EquipmentBaseQty, &pi.EquipmentPrice,
		); err != nil {
			return nil, err
		}
		out = append(out, pi)
	}
	return out, rows.Err()
}
