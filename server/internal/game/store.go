package game

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

func LoadPlayerForUpdate(ctx context.Context, tx pgx.Tx, playerID string) (Player, error) {
	var p Player
	err := tx.QueryRow(ctx, `
		SELECT
			p.id,
			u.id,
			u.username,
			u.is_admin,
			u.must_change_password,
			p.credits,
			p.turns,
			p.turns_max,
			p.sector_id,
			p.cargo_max,
			p.cargo_ore,
			p.cargo_organics,
			p.cargo_equipment,
			p.last_turn_regen,
			p.season_id,
			s.name,
			COALESCE(cm.corp_id, ''),
			COALESCE(c.name, ''),
			COALESCE(cm.role, ''),
			COALESCE(c.credits, 0)
		FROM players p
		JOIN users u ON u.id = p.user_id
		JOIN seasons s ON s.id = p.season_id
		LEFT JOIN corp_members cm ON cm.player_id = p.id
		LEFT JOIN corporations c ON c.id = cm.corp_id
		WHERE p.id = $1
		FOR UPDATE
	`, playerID).Scan(
		&p.ID,
		&p.UserID,
		&p.Username,
		&p.IsAdmin,
		&p.MustChangePass,
		&p.Credits,
		&p.Turns,
		&p.TurnsMax,
		&p.SectorID,
		&p.CargoMax,
		&p.CargoOre,
		&p.CargoOrganics,
		&p.CargoEquipment,
		&p.LastTurnRegen,
		&p.SeasonID,
		&p.SeasonName,
		&p.CorpID,
		&p.CorpName,
		&p.CorpRole,
		&p.CorpCredits,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Player{}, ErrNotFound
	}
	return p, err
}

func SavePlayer(ctx context.Context, tx pgx.Tx, p Player) error {
	_, err := tx.Exec(ctx, `
		UPDATE players SET
			credits = $2,
			turns = $3,
			turns_max = $4,
			sector_id = $5,
			cargo_max = $6,
			cargo_ore = $7,
			cargo_organics = $8,
			cargo_equipment = $9,
			last_turn_regen = $10,
			season_id = $11
		WHERE id = $1
	`, p.ID, p.Credits, p.Turns, p.TurnsMax, p.SectorID, p.CargoMax, p.CargoOre, p.CargoOrganics, p.CargoEquipment, p.LastTurnRegen, p.SeasonID)
	return err
}

func MarkDiscovered(ctx context.Context, tx pgx.Tx, playerID string, sectorID int) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO player_discoveries(player_id, sector_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, playerID, sectorID)
	return err
}

func InsertLog(ctx context.Context, tx pgx.Tx, playerID, kind, message string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO logs(player_id, kind, message)
		VALUES ($1, $2, $3)
	`, playerID, kind, message)
	return err
}

func LoadRecentLogs(ctx context.Context, pool *pgxpool.Pool, playerID string, limit int) ([]LogEntry, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
		SELECT created_at, kind, message
		FROM logs
		WHERE player_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, playerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]LogEntry, 0, limit)
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(&e.At, &e.Kind, &e.Message); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

type PortRow struct {
	OreMode            string
	OreQty             int
	OreBaseQty         int
	OreBasePrice       int
	OrganicsMode       string
	OrganicsQty        int
	OrganicsBaseQty    int
	OrganicsBasePrice  int
	EquipmentMode      string
	EquipmentQty       int
	EquipmentBaseQty   int
	EquipmentBasePrice int
}

func LoadSectorView(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}, sectorID int) (SectorView, error) {
	var s SectorView
	if err := q.QueryRow(ctx, "SELECT id, name FROM sectors WHERE id = $1", sectorID).Scan(&s.ID, &s.Name); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SectorView{}, ErrNotFound
		}
		return SectorView{}, err
	}

	rows, err := q.Query(ctx, "SELECT to_sector FROM warps WHERE from_sector = $1 ORDER BY to_sector", sectorID)
	if err != nil {
		return SectorView{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var to int
		if err := rows.Scan(&to); err != nil {
			return SectorView{}, err
		}
		s.Warps = append(s.Warps, to)
	}
	if err := rows.Err(); err != nil {
		return SectorView{}, err
	}

	// Event (optional)
	var activeEv *ActiveEvent
	if ev, ok, err := LoadActiveEvent(ctx, q, sectorID); err != nil {
		return SectorView{}, err
	} else if ok {
		v := ev.ToView()
		s.Event = &v
		activeEv = &ev
	}

	// Port (optional)
	var pr PortRow
	perr := q.QueryRow(ctx, `
		SELECT
			ore_mode, ore_qty, ore_base_qty, ore_base_price,
			organics_mode, organics_qty, organics_base_qty, organics_base_price,
			equipment_mode, equipment_qty, equipment_base_qty, equipment_base_price
		FROM ports
		WHERE sector_id = $1
	`, sectorID).Scan(
		&pr.OreMode, &pr.OreQty, &pr.OreBaseQty, &pr.OreBasePrice,
		&pr.OrganicsMode, &pr.OrganicsQty, &pr.OrganicsBaseQty, &pr.OrganicsBasePrice,
		&pr.EquipmentMode, &pr.EquipmentQty, &pr.EquipmentBaseQty, &pr.EquipmentBasePrice,
	)
	if perr == nil {
		orePct := 100
		orgPct := 100
		eqPct := 100
		if activeEv != nil {
			orePct = pricePercentForCommodity(*activeEv, "ORE")
			orgPct = pricePercentForCommodity(*activeEv, "ORGANICS")
			eqPct = pricePercentForCommodity(*activeEv, "EQUIPMENT")
		}
		pv := &PortView{
			OreMode:          pr.OreMode,
			OreQty:           pr.OreQty,
			OreBaseQty:       pr.OreBaseQty,
			OrePrice:         PricePerUnitWithPercent(pr.OreBasePrice, pr.OreBaseQty, pr.OreQty, orePct),
			OrganicsMode:     pr.OrganicsMode,
			OrganicsQty:      pr.OrganicsQty,
			OrganicsBaseQty:  pr.OrganicsBaseQty,
			OrganicsPrice:    PricePerUnitWithPercent(pr.OrganicsBasePrice, pr.OrganicsBaseQty, pr.OrganicsQty, orgPct),
			EquipmentMode:    pr.EquipmentMode,
			EquipmentQty:     pr.EquipmentQty,
			EquipmentBaseQty: pr.EquipmentBaseQty,
			EquipmentPrice:   PricePerUnitWithPercent(pr.EquipmentBasePrice, pr.EquipmentBaseQty, pr.EquipmentQty, eqPct),
		}
		s.Port = pv
	} else if !errors.Is(perr, pgx.ErrNoRows) {
		return SectorView{}, perr
	}

	// Planet (optional)
	var planet PlanetView
	var ownerPlayerID pgtype.Text
	var ownerCorpID pgtype.Text
	var ownerUsername string
	var ownerCorpName string
	plErr := q.QueryRow(ctx, `
		SELECT
			pl.id,
			pl.name,
			pl.owner_player_id,
			pl.owner_corp_id,
			COALESCE(u.username, ''),
			COALESCE(c.name, ''),
			pl.production_ore,
			pl.production_organics,
			pl.production_equipment,
			pl.storage_ore,
			pl.storage_organics,
			pl.storage_equipment,
			pl.storage_max,
			pl.citadel_level
		FROM planets pl
		LEFT JOIN players op ON op.id = pl.owner_player_id
		LEFT JOIN users u ON u.id = op.user_id
		LEFT JOIN corporations c ON c.id = pl.owner_corp_id
		WHERE pl.sector_id = $1
	`, sectorID).Scan(
		&planet.ID,
		&planet.Name,
		&ownerPlayerID,
		&ownerCorpID,
		&ownerUsername,
		&ownerCorpName,
		&planet.ProductionOre,
		&planet.ProductionOrganics,
		&planet.ProductionEquipment,
		&planet.StorageOre,
		&planet.StorageOrganics,
		&planet.StorageEquipment,
		&planet.StorageMax,
		&planet.CitadelLevel,
	)
	if plErr == nil {
		if ownerCorpID.Valid && ownerCorpName != "" {
			planet.OwnerType = "CORP"
			planet.Owner = ownerCorpName
		} else if ownerPlayerID.Valid && ownerUsername != "" {
			planet.OwnerType = "PLAYER"
			planet.Owner = ownerUsername
		}
		s.Planet = &planet
	} else if !errors.Is(plErr, pgx.ErrNoRows) {
		return SectorView{}, plErr
	}

	// Mines (sum)
	_ = q.QueryRow(ctx, "SELECT COALESCE(SUM(qty),0) FROM mines WHERE sector_id=$1", sectorID).Scan(&s.Mines)

	return s, nil
}

func NowUTC() time.Time {
	return time.Now().UTC()
}
