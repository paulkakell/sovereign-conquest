package game

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	maxCargoUpgrades = 20
	maxTurnUpgrades  = 10
)

type shipDef struct {
	Type     string
	CargoMax int
	TurnsMax int
	Price    int64
}

var shipCatalog = []shipDef{
	{Type: "SCOUT", CargoMax: 30, TurnsMax: 100, Price: 0},
	{Type: "TRADER", CargoMax: 60, TurnsMax: 110, Price: 25000},
	{Type: "FREIGHTER", CargoMax: 90, TurnsMax: 110, Price: 60000},
	{Type: "INTERCEPTOR", CargoMax: 40, TurnsMax: 140, Price: 50000},
}

func findShipDef(shipType string) (shipDef, bool) {
	st := strings.ToUpper(strings.TrimSpace(shipType))
	for _, d := range shipCatalog {
		if d.Type == st {
			return d, true
		}
	}
	return shipDef{}, false
}

func isShipyardAvailable(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, sectorID int) (bool, error) {
	var isProt bool
	err := q.QueryRow(ctx, "SELECT is_protectorate FROM sectors WHERE id=$1", sectorID).Scan(&isProt)
	if err != nil {
		return false, err
	}
	return isProt, nil
}

func executeShipyardCommand(ctx context.Context, tx pgx.Tx, p *Player, cmd CommandRequest) (phase2Result, error) {
	available, err := isShipyardAvailable(ctx, tx, p.SectorID)
	if err != nil {
		return phase2Result{}, err
	}
	if !available {
		msg := "No shipyard is available in this sector. Shipyards are found in Protectorate sectors."
		return phase2Result{OK: false, Message: msg, ErrorCode: "NO_SHIPYARD"}, nil
	}

	action := strings.ToUpper(strings.TrimSpace(cmd.Action))
	name := strings.ToUpper(strings.TrimSpace(cmd.Name))
	if action == "" {
		action = "INFO"
	}

	switch action {
	case "INFO":
		msg := shipyardInfo(*p)
		return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "SYSTEM", msg: msg}}}, nil
	case "BUY":
		if name == "" {
			return phase2Result{OK: false, Message: "SHIPYARD BUY requires a ship type (e.g., TRADER).", ErrorCode: "INVALID_SHIP"}, nil
		}
		return shipyardBuy(p, name)
	case "SELL":
		return shipyardSell(p)
	case "UPGRADE":
		if name == "" {
			return phase2Result{OK: false, Message: "SHIPYARD UPGRADE requires CARGO or TURNS.", ErrorCode: "INVALID_UPGRADE"}, nil
		}
		return shipyardUpgrade(p, name)
	default:
		return phase2Result{OK: false, Message: "Unknown SHIPYARD subcommand.", ErrorCode: "UNKNOWN_SUBCOMMAND"}, nil
	}
}

func shipyardInfo(p Player) string {
	ship, _ := findShipDef(p.ShipType)
	if ship.Type == "" {
		ship.Type = strings.ToUpper(strings.TrimSpace(p.ShipType))
	}

	lines := []string{}
	lines = append(lines, "SHIPYARD commands: SHIPYARD BUY {type} | SHIPYARD SELL | SHIPYARD UPGRADE {CARGO|TURNS}")
	lines = append(lines, fmt.Sprintf("Current ship: %s (CargoMax=%d, TurnsMax=%d)", ship.Type, p.CargoMax, p.TurnsMax))
	lines = append(lines, fmt.Sprintf("Upgrades: Cargo +%d (%d/%d), Turns +%d (%d/%d)", p.ShipCargoUpgrades*5, p.ShipCargoUpgrades, maxCargoUpgrades, p.ShipTurnUpgrades*10, p.ShipTurnUpgrades, maxTurnUpgrades))
	lines = append(lines, "Available ships:")
	for _, d := range shipCatalog {
		lines = append(lines, fmt.Sprintf("- %s: CargoMax=%d TurnsMax=%d Price=%d", d.Type, d.CargoMax, d.TurnsMax, d.Price))
	}
	lines = append(lines, fmt.Sprintf("Next cargo upgrade cost: %d", cargoUpgradeCost(p.ShipCargoUpgrades)))
	lines = append(lines, fmt.Sprintf("Next turns upgrade cost: %d", turnsUpgradeCost(p.ShipTurnUpgrades)))
	return strings.Join(lines, "\n")
}

func cargoUpgradeCost(current int) int64 {
	return int64(2000 * (current + 1))
}

func turnsUpgradeCost(current int) int64 {
	return int64(1500 * (current + 1))
}

func totalCargo(p *Player) int {
	if p == nil {
		return 0
	}
	return p.CargoOre + p.CargoOrganics + p.CargoEquipment
}

func shipyardBuy(p *Player, shipType string) (phase2Result, error) {
	d, ok := findShipDef(shipType)
	if !ok {
		return phase2Result{OK: false, Message: "Unknown ship type.", ErrorCode: "INVALID_SHIP"}, nil
	}
	if strings.EqualFold(p.ShipType, d.Type) {
		return phase2Result{OK: false, Message: "You already own this ship type.", ErrorCode: "INVALID_SHIP"}, nil
	}
	if p.Credits < d.Price {
		return phase2Result{OK: false, Message: "Insufficient credits to buy that ship.", ErrorCode: "INSUFFICIENT_CREDITS"}, nil
	}
	if totalCargo(p) > d.CargoMax {
		return phase2Result{OK: false, Message: "Your current cargo exceeds the capacity of that ship. Reduce cargo before buying.", ErrorCode: "CARGO_TOO_LARGE"}, nil
	}

	p.Credits -= d.Price
	p.ShipType = d.Type
	p.ShipCargoUpgrades = 0
	p.ShipTurnUpgrades = 0
	p.CargoMax = d.CargoMax
	p.TurnsMax = d.TurnsMax
	if p.Turns > p.TurnsMax {
		p.Turns = p.TurnsMax
	}

	msg := fmt.Sprintf("Purchased %s for %d credits. CargoMax=%d, TurnsMax=%d.", d.Type, d.Price, p.CargoMax, p.TurnsMax)
	return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
}

func shipyardSell(p *Player) (phase2Result, error) {
	cur, ok := findShipDef(p.ShipType)
	if !ok || cur.Type == "" {
		return phase2Result{OK: false, Message: "Unknown current ship type.", ErrorCode: "INVALID_SHIP"}, nil
	}
	if cur.Type == "SCOUT" {
		return phase2Result{OK: false, Message: "You cannot sell your starter SCOUT.", ErrorCode: "INVALID_SHIP"}, nil
	}

	scout, _ := findShipDef("SCOUT")
	if totalCargo(p) > scout.CargoMax {
		return phase2Result{OK: false, Message: "Your cargo exceeds SCOUT capacity. Reduce cargo before selling.", ErrorCode: "CARGO_TOO_LARGE"}, nil
	}

	resale := cur.Price * 70 / 100
	p.Credits += resale
	p.ShipType = scout.Type
	p.ShipCargoUpgrades = 0
	p.ShipTurnUpgrades = 0
	p.CargoMax = scout.CargoMax
	p.TurnsMax = scout.TurnsMax
	if p.Turns > p.TurnsMax {
		p.Turns = p.TurnsMax
	}

	msg := fmt.Sprintf("Sold %s for %d credits. You are now flying a %s.", cur.Type, resale, scout.Type)
	return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
}

func shipyardUpgrade(p *Player, which string) (phase2Result, error) {
	which = strings.ToUpper(strings.TrimSpace(which))

	switch which {
	case "CARGO":
		if p.ShipCargoUpgrades >= maxCargoUpgrades {
			return phase2Result{OK: false, Message: "Cargo upgrades are already at maximum.", ErrorCode: "MAX_UPGRADES"}, nil
		}
		cost := cargoUpgradeCost(p.ShipCargoUpgrades)
		if p.Credits < cost {
			return phase2Result{OK: false, Message: "Insufficient credits for cargo upgrade.", ErrorCode: "INSUFFICIENT_CREDITS"}, nil
		}
		p.Credits -= cost
		p.ShipCargoUpgrades++
		p.CargoMax += 5
		msg := fmt.Sprintf("Cargo upgraded (+5). New CargoMax=%d. Cost=%d.", p.CargoMax, cost)
		return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
	case "TURNS":
		if p.ShipTurnUpgrades >= maxTurnUpgrades {
			return phase2Result{OK: false, Message: "Turns upgrades are already at maximum.", ErrorCode: "MAX_UPGRADES"}, nil
		}
		cost := turnsUpgradeCost(p.ShipTurnUpgrades)
		if p.Credits < cost {
			return phase2Result{OK: false, Message: "Insufficient credits for turns upgrade.", ErrorCode: "INSUFFICIENT_CREDITS"}, nil
		}
		p.Credits -= cost
		p.ShipTurnUpgrades++
		p.TurnsMax += 10
		msg := fmt.Sprintf("Turns capacity upgraded (+10). New TurnsMax=%d. Cost=%d.", p.TurnsMax, cost)
		return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
	default:
		return phase2Result{OK: false, Message: "Unknown upgrade type. Use CARGO or TURNS.", ErrorCode: "INVALID_UPGRADE"}, nil
	}
}
