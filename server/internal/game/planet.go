package game

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	planetColonizeCostCredits int64 = 1500
	planetMaxNameLen                = 40
	planetMinNameLen                = 3
	citadelMaxLevel                 = 10
)

type planetForUpdate struct {
	ID                  int64
	SectorID            int
	Name                string
	OwnerPlayerID       pgtype.Text
	OwnerCorpID         pgtype.Text
	ProductionOre       int
	ProductionOrganics  int
	ProductionEquipment int
	StorageOre          int
	StorageOrganics     int
	StorageEquipment    int
	StorageMax          int
	CitadelLevel        int
}

func executePlanetCommand(ctx context.Context, tx pgx.Tx, p *Player, cmd CommandRequest) (phase2Result, error) {
	action := cmd.Action
	if action == "" {
		action = "INFO"
	}

	switch action {
	case "INFO":
		return planetInfo(ctx, tx, p)
	case "COLONIZE":
		return planetColonize(ctx, tx, p, cmd.Name)
	case "LOAD":
		return planetTransfer(ctx, tx, p, "LOAD", cmd.Commodity, cmd.Quantity)
	case "UNLOAD":
		return planetTransfer(ctx, tx, p, "UNLOAD", cmd.Commodity, cmd.Quantity)
	case "UPGRADE_CITADEL":
		return planetUpgradeCitadel(ctx, tx, p)
	default:
		return phase2Result{OK: false, Message: "Unknown PLANET subcommand.", ErrorCode: "UNKNOWN_SUBCOMMAND"}, nil
	}
}

func planetInfo(ctx context.Context, tx pgx.Tx, p *Player) (phase2Result, error) {
	pl, exists, err := loadPlanet(ctx, tx, p.SectorID, false)
	if err != nil {
		return phase2Result{}, err
	}
	if !exists {
		return phase2Result{OK: true, Message: "No planet in this sector.", Logs: []logToInsert{{kind: "SYSTEM", msg: "No planet in this sector."}}}, nil
	}

	owner := "Unclaimed"
	if pl.OwnerCorpID.Valid || pl.OwnerPlayerID.Valid {
		if canAccessPlanet(*p, pl) {
			if pl.OwnerCorpID.Valid && p.CorpID != "" && pl.OwnerCorpID.String == p.CorpID {
				owner = fmt.Sprintf("Your corp (%s)", p.CorpName)
			} else {
				owner = "You"
			}
		} else if pl.OwnerCorpID.Valid {
			owner = "Another corporation"
		} else {
			owner = "Another player"
		}
	}

	msg := strings.Join([]string{
		fmt.Sprintf("Planet: %s", pl.Name),
		fmt.Sprintf("Owner: %s", owner),
		fmt.Sprintf("Citadel: %d", pl.CitadelLevel),
		fmt.Sprintf("Production/tick: Ore %d, Org %d, Eq %d", pl.ProductionOre, pl.ProductionOrganics, pl.ProductionEquipment),
		fmt.Sprintf("Storage: Ore %d/%d, Org %d/%d, Eq %d/%d", pl.StorageOre, pl.StorageMax, pl.StorageOrganics, pl.StorageMax, pl.StorageEquipment, pl.StorageMax),
	}, "\n")

	return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "SYSTEM", msg: msg}}}, nil
}

func planetColonize(ctx context.Context, tx pgx.Tx, p *Player, name string) (phase2Result, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = fmt.Sprintf("Planet %d", p.SectorID)
	}
	name = sanitizePlanetName(name)
	if len(name) < planetMinNameLen {
		return phase2Result{OK: false, Message: "Planet name is too short.", ErrorCode: "INVALID_NAME"}, nil
	}
	if len(name) > planetMaxNameLen {
		return phase2Result{OK: false, Message: fmt.Sprintf("Planet name is too long (max %d).", planetMaxNameLen), ErrorCode: "INVALID_NAME"}, nil
	}

	pl, exists, err := loadPlanet(ctx, tx, p.SectorID, true)
	if err != nil {
		return phase2Result{}, err
	}

	if exists {
		// already owned?
		if pl.OwnerCorpID.Valid || pl.OwnerPlayerID.Valid {
			if canAccessPlanet(*p, pl) {
				return phase2Result{OK: false, Message: "You already control this planet.", ErrorCode: "ALREADY_OWNED"}, nil
			}
			return phase2Result{OK: false, Message: "This planet is already controlled.", ErrorCode: "ALREADY_OWNED"}, nil
		}

		if p.Credits < planetColonizeCostCredits {
			return phase2Result{OK: false, Message: fmt.Sprintf("Colonization requires %d credits.", planetColonizeCostCredits), ErrorCode: "INSUFFICIENT_CREDITS"}, nil
		}

		p.Credits -= planetColonizeCostCredits
		pl.Name = name
		pl.OwnerPlayerID = pgtype.Text{String: p.ID, Valid: true}
		if p.CorpID != "" {
			pl.OwnerCorpID = pgtype.Text{String: p.CorpID, Valid: true}
		}

		if err := savePlanetOwnershipAndName(ctx, tx, pl); err != nil {
			return phase2Result{}, err
		}

		msg := fmt.Sprintf("Colonized existing planet '%s' for %d credits.", pl.Name, planetColonizeCostCredits)
		return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
	}

	if p.Credits < planetColonizeCostCredits {
		return phase2Result{OK: false, Message: fmt.Sprintf("Colonization requires %d credits.", planetColonizeCostCredits), ErrorCode: "INSUFFICIENT_CREDITS"}, nil
	}

	p.Credits -= planetColonizeCostCredits

	ownerPlayerID := p.ID
	var ownerCorpID any = nil
	if p.CorpID != "" {
		ownerCorpID = p.CorpID
	}

	// Default production is modest; the universe generator may create stronger unclaimed planets.
	prodOre, prodOrg, prodEq := 10, 6, 3
	storageMax := 2000

	var planetID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO planets(
			sector_id, name,
			owner_player_id, owner_corp_id,
			production_ore, production_organics, production_equipment,
			storage_max
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id
	`, p.SectorID, name, ownerPlayerID, ownerCorpID, prodOre, prodOrg, prodEq, storageMax).Scan(&planetID)
	if err != nil {
		return phase2Result{}, err
	}

	msg := fmt.Sprintf("Established new planet '%s' for %d credits.", name, planetColonizeCostCredits)
	return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
}

func planetTransfer(ctx context.Context, tx pgx.Tx, p *Player, mode, commodity string, qty int) (phase2Result, error) {
	commodity = strings.ToUpper(strings.TrimSpace(commodity))
	if commodity != "ORE" && commodity != "ORGANICS" && commodity != "EQUIPMENT" {
		return phase2Result{OK: false, Message: "Commodity must be ORE, ORGANICS, or EQUIPMENT.", ErrorCode: "INVALID_COMMODITY"}, nil
	}
	if qty < 1 || qty > 1000000 {
		return phase2Result{OK: false, Message: "Quantity must be at least 1.", ErrorCode: "INVALID_QTY"}, nil
	}

	pl, exists, err := loadPlanet(ctx, tx, p.SectorID, true)
	if err != nil {
		return phase2Result{}, err
	}
	if !exists {
		return phase2Result{OK: false, Message: "No planet in this sector.", ErrorCode: "NO_PLANET"}, nil
	}
	if !canAccessPlanet(*p, pl) {
		return phase2Result{OK: false, Message: "You do not have access to this planet.", ErrorCode: "NO_ACCESS"}, nil
	}

	totalCargo := p.CargoOre + p.CargoOrganics + p.CargoEquipment
	freeCargo := p.CargoMax - totalCargo
	if freeCargo < 0 {
		freeCargo = 0
	}

	if mode == "LOAD" {
		if qty > freeCargo {
			return phase2Result{OK: false, Message: "Not enough cargo space.", ErrorCode: "NO_CARGO_SPACE"}, nil
		}
		switch commodity {
		case "ORE":
			if pl.StorageOre < qty {
				return phase2Result{OK: false, Message: "Planet does not have that much ore stored.", ErrorCode: "INSUFFICIENT_STORED"}, nil
			}
			pl.StorageOre -= qty
			p.CargoOre += qty
		case "ORGANICS":
			if pl.StorageOrganics < qty {
				return phase2Result{OK: false, Message: "Planet does not have that many organics stored.", ErrorCode: "INSUFFICIENT_STORED"}, nil
			}
			pl.StorageOrganics -= qty
			p.CargoOrganics += qty
		case "EQUIPMENT":
			if pl.StorageEquipment < qty {
				return phase2Result{OK: false, Message: "Planet does not have that much equipment stored.", ErrorCode: "INSUFFICIENT_STORED"}, nil
			}
			pl.StorageEquipment -= qty
			p.CargoEquipment += qty
		}

		if err := savePlanetStorage(ctx, tx, pl); err != nil {
			return phase2Result{}, err
		}

		msg := fmt.Sprintf("Loaded %d %s from %s.", qty, strings.ToLower(commodity), pl.Name)
		return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
	}

	// UNLOAD
	switch commodity {
	case "ORE":
		if p.CargoOre < qty {
			return phase2Result{OK: false, Message: "You do not have that much ore.", ErrorCode: "INSUFFICIENT_CARGO"}, nil
		}
		if pl.StorageOre+qty > pl.StorageMax {
			return phase2Result{OK: false, Message: "Planet ore storage is full.", ErrorCode: "STORAGE_FULL"}, nil
		}
		p.CargoOre -= qty
		pl.StorageOre += qty
	case "ORGANICS":
		if p.CargoOrganics < qty {
			return phase2Result{OK: false, Message: "You do not have that many organics.", ErrorCode: "INSUFFICIENT_CARGO"}, nil
		}
		if pl.StorageOrganics+qty > pl.StorageMax {
			return phase2Result{OK: false, Message: "Planet organics storage is full.", ErrorCode: "STORAGE_FULL"}, nil
		}
		p.CargoOrganics -= qty
		pl.StorageOrganics += qty
	case "EQUIPMENT":
		if p.CargoEquipment < qty {
			return phase2Result{OK: false, Message: "You do not have that much equipment.", ErrorCode: "INSUFFICIENT_CARGO"}, nil
		}
		if pl.StorageEquipment+qty > pl.StorageMax {
			return phase2Result{OK: false, Message: "Planet equipment storage is full.", ErrorCode: "STORAGE_FULL"}, nil
		}
		p.CargoEquipment -= qty
		pl.StorageEquipment += qty
	}

	if err := savePlanetStorage(ctx, tx, pl); err != nil {
		return phase2Result{}, err
	}

	msg := fmt.Sprintf("Unloaded %d %s to %s.", qty, strings.ToLower(commodity), pl.Name)
	return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
}

func planetUpgradeCitadel(ctx context.Context, tx pgx.Tx, p *Player) (phase2Result, error) {
	pl, exists, err := loadPlanet(ctx, tx, p.SectorID, true)
	if err != nil {
		return phase2Result{}, err
	}
	if !exists {
		return phase2Result{OK: false, Message: "No planet in this sector.", ErrorCode: "NO_PLANET"}, nil
	}
	if !canAccessPlanet(*p, pl) {
		return phase2Result{OK: false, Message: "You do not have access to this planet.", ErrorCode: "NO_ACCESS"}, nil
	}
	if pl.CitadelLevel >= citadelMaxLevel {
		return phase2Result{OK: false, Message: "Citadel is already at max level.", ErrorCode: "MAX_LEVEL"}, nil
	}

	next := pl.CitadelLevel + 1
	cost := citadelUpgradeCost(next)
	if p.Credits < cost {
		return phase2Result{OK: false, Message: fmt.Sprintf("Citadel upgrade requires %d credits.", cost), ErrorCode: "INSUFFICIENT_CREDITS"}, nil
	}

	p.Credits -= cost
	pl.CitadelLevel = next
	if err := savePlanetCitadel(ctx, tx, pl); err != nil {
		return phase2Result{}, err
	}

	msg := fmt.Sprintf("Citadel upgraded to level %d for %d credits.", next, cost)
	return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
}

func loadPlanet(ctx context.Context, tx pgx.Tx, sectorID int, forUpdate bool) (planetForUpdate, bool, error) {
	var pl planetForUpdate
	pl.SectorID = sectorID

	q := `
		SELECT
			id,
			sector_id,
			name,
			owner_player_id,
			owner_corp_id,
			production_ore,
			production_organics,
			production_equipment,
			storage_ore,
			storage_organics,
			storage_equipment,
			storage_max,
			citadel_level
		FROM planets
		WHERE sector_id = $1
	`
	if forUpdate {
		q += " FOR UPDATE"
	}

	err := tx.QueryRow(ctx, q, sectorID).Scan(
		&pl.ID,
		&pl.SectorID,
		&pl.Name,
		&pl.OwnerPlayerID,
		&pl.OwnerCorpID,
		&pl.ProductionOre,
		&pl.ProductionOrganics,
		&pl.ProductionEquipment,
		&pl.StorageOre,
		&pl.StorageOrganics,
		&pl.StorageEquipment,
		&pl.StorageMax,
		&pl.CitadelLevel,
	)
	if err == nil {
		return pl, true, nil
	}
	if err == pgx.ErrNoRows {
		return planetForUpdate{}, false, nil
	}
	return planetForUpdate{}, false, err
}

func savePlanetStorage(ctx context.Context, tx pgx.Tx, pl planetForUpdate) error {
	_, err := tx.Exec(ctx, `
		UPDATE planets SET
			storage_ore = $2,
			storage_organics = $3,
			storage_equipment = $4
		WHERE id = $1
	`, pl.ID, pl.StorageOre, pl.StorageOrganics, pl.StorageEquipment)
	return err
}

func savePlanetCitadel(ctx context.Context, tx pgx.Tx, pl planetForUpdate) error {
	_, err := tx.Exec(ctx, `
		UPDATE planets SET
			citadel_level = $2
		WHERE id = $1
	`, pl.ID, pl.CitadelLevel)
	return err
}

func savePlanetOwnershipAndName(ctx context.Context, tx pgx.Tx, pl planetForUpdate) error {
	var ownerPlayer any = nil
	var ownerCorp any = nil
	if pl.OwnerPlayerID.Valid {
		ownerPlayer = pl.OwnerPlayerID.String
	}
	if pl.OwnerCorpID.Valid {
		ownerCorp = pl.OwnerCorpID.String
	}
	_, err := tx.Exec(ctx, `
		UPDATE planets SET
			name = $2,
			owner_player_id = $3,
			owner_corp_id = $4
		WHERE id = $1
	`, pl.ID, pl.Name, ownerPlayer, ownerCorp)
	return err
}

func canAccessPlanet(p Player, pl planetForUpdate) bool {
	if pl.OwnerPlayerID.Valid && pl.OwnerPlayerID.String == p.ID {
		return true
	}
	if pl.OwnerCorpID.Valid && p.CorpID != "" && pl.OwnerCorpID.String == p.CorpID {
		return true
	}
	return false
}

func sanitizePlanetName(name string) string {
	name = strings.ReplaceAll(name, "\n", " ")
	name = strings.ReplaceAll(name, "\r", " ")
	name = strings.TrimSpace(name)
	// collapse runs of spaces
	fields := strings.Fields(name)
	return strings.Join(fields, " ")
}

func citadelUpgradeCost(nextLevel int) int64 {
	if nextLevel < 1 {
		nextLevel = 1
	}
	return int64(nextLevel) * 5000
}
