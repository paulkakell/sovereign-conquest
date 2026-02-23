package game

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type portForUpdate struct {
	OreMode      string
	OreQty       int
	OreBaseQty   int
	OreBasePrice int

	OrganicsMode      string
	OrganicsQty       int
	OrganicsBaseQty   int
	OrganicsBasePrice int

	EquipmentMode      string
	EquipmentQty       int
	EquipmentBaseQty   int
	EquipmentBasePrice int
}

func executeTrade(ctx context.Context, tx pgx.Tx, p *Player, cmd CommandRequest) (string, bool, error) {
	action := strings.ToUpper(cmd.Action)
	commodity := strings.ToUpper(cmd.Commodity)
	qty := cmd.Quantity

	if action != "BUY" && action != "SELL" {
		return "Trade action must be BUY or SELL.", false, nil
	}
	if commodity != "ORE" && commodity != "ORGANICS" && commodity != "EQUIPMENT" {
		return "Commodity must be ORE, ORGANICS, or EQUIPMENT.", false, nil
	}
	if qty < 1 || qty > 1000000 {
		return "Quantity must be at least 1.", false, nil
	}

	var port portForUpdate
	err := tx.QueryRow(ctx, `
		SELECT
			ore_mode, ore_qty, ore_base_qty, ore_base_price,
			organics_mode, organics_qty, organics_base_qty, organics_base_price,
			equipment_mode, equipment_qty, equipment_base_qty, equipment_base_price
		FROM ports
		WHERE sector_id = $1
		FOR UPDATE
	`, p.SectorID).Scan(
		&port.OreMode, &port.OreQty, &port.OreBaseQty, &port.OreBasePrice,
		&port.OrganicsMode, &port.OrganicsQty, &port.OrganicsBaseQty, &port.OrganicsBasePrice,
		&port.EquipmentMode, &port.EquipmentQty, &port.EquipmentBaseQty, &port.EquipmentBasePrice,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return "No port in this sector.", false, nil
	}
	if err != nil {
		return "Trade failed.", false, err
	}

	totalCargo := p.CargoOre + p.CargoOrganics + p.CargoEquipment
	freeCargo := p.CargoMax - totalCargo
	if freeCargo < 0 {
		freeCargo = 0
	}

	switch commodity {
	case "ORE":
		return tradeCommodity(ctx, tx, p, action, qty,
			"ORE", port.OreMode, &port.OreQty, port.OreBaseQty, port.OreBasePrice, freeCargo)
	case "ORGANICS":
		return tradeCommodity(ctx, tx, p, action, qty,
			"ORGANICS", port.OrganicsMode, &port.OrganicsQty, port.OrganicsBaseQty, port.OrganicsBasePrice, freeCargo)
	case "EQUIPMENT":
		return tradeCommodity(ctx, tx, p, action, qty,
			"EQUIPMENT", port.EquipmentMode, &port.EquipmentQty, port.EquipmentBaseQty, port.EquipmentBasePrice, freeCargo)
	default:
		return "Invalid commodity.", false, nil
	}
}

func tradeCommodity(ctx context.Context, tx pgx.Tx, p *Player, action string, qty int,
	name string, portMode string, portQty *int, baseQty int, basePrice int, freeCargo int,
) (string, bool, error) {
	pricePercent := 100
	if ev, ok, err := LoadActiveEvent(ctx, tx, p.SectorID); err != nil {
		return "Trade failed.", false, err
	} else if ok {
		pricePercent = pricePercentForCommodity(ev, name)
	}
	pricePerUnit := PricePerUnitWithPercent(basePrice, baseQty, *portQty, pricePercent)
	totalPrice := int64(pricePerUnit) * int64(qty)

	if action == "BUY" {
		if portMode != "SELL" {
			return fmt.Sprintf("This port is not selling %s.", name), false, nil
		}
		if qty > *portQty {
			return "Port does not have enough inventory.", false, nil
		}
		if qty > freeCargo {
			return "Not enough cargo space.", false, nil
		}
		if p.Credits < totalPrice {
			return "Not enough credits.", false, nil
		}

		p.Credits -= totalPrice
		switch name {
		case "ORE":
			p.CargoOre += qty
		case "ORGANICS":
			p.CargoOrganics += qty
		case "EQUIPMENT":
			p.CargoEquipment += qty
		}
		*portQty -= qty

	} else { // SELL
		if portMode != "BUY" {
			return fmt.Sprintf("This port is not buying %s.", name), false, nil
		}

		switch name {
		case "ORE":
			if p.CargoOre < qty {
				return "You do not have that much ore.", false, nil
			}
		case "ORGANICS":
			if p.CargoOrganics < qty {
				return "You do not have that many organics.", false, nil
			}
		case "EQUIPMENT":
			if p.CargoEquipment < qty {
				return "You do not have that much equipment.", false, nil
			}
		}

		if *portQty+qty > baseQty {
			return "Port demand is saturated right now.", false, nil
		}

		p.Credits += totalPrice
		switch name {
		case "ORE":
			p.CargoOre -= qty
		case "ORGANICS":
			p.CargoOrganics -= qty
		case "EQUIPMENT":
			p.CargoEquipment -= qty
		}
		*portQty += qty
	}

	// Persist port qty changes (update the single commodity column).
	// Keeping it simple: update all commodity qty columns based on player's sector.
	_, err := tx.Exec(ctx, `
		UPDATE ports SET
			ore_qty = CASE WHEN $2 = 'ORE' THEN $3 ELSE ore_qty END,
			organics_qty = CASE WHEN $2 = 'ORGANICS' THEN $3 ELSE organics_qty END,
			equipment_qty = CASE WHEN $2 = 'EQUIPMENT' THEN $3 ELSE equipment_qty END
		WHERE sector_id = $1
	`, p.SectorID, name, *portQty)
	if err != nil {
		return "Trade failed.", false, err
	}

	verb := "bought"
	if action == "SELL" {
		verb = "sold"
	}
	return fmt.Sprintf("You %s %d %s at %d credits each (%d total).", verb, qty, strings.ToLower(name), pricePerUnit, totalPrice), true, nil
}
