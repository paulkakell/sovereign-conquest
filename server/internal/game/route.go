package game

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type RouteSuggestion struct {
	Commodity      string
	BuySectorID    int
	BuySectorName  string
	SellSectorID   int
	SellSectorName string

	BuyPrice  int
	SellPrice int

	BuyScannedAt  time.Time
	SellScannedAt time.Time

	StepsToBuy     int
	StepsBuyToSell int
	TotalMoves     int
	TotalTurns     int
	TradeQty       int
	ProfitPerUnit  int
	ProfitPerTrip  int64
	ScoreX1        int64 // score used for selection (scaled)
}

func executeRouteCommand(ctx context.Context, tx pgx.Tx, p Player, cmd CommandRequest) (string, error) {
	filter := strings.ToUpper(strings.TrimSpace(cmd.Commodity))
	if filter != "" && filter != "ORE" && filter != "ORGANICS" && filter != "EQUIPMENT" {
		filter = ""
	}

	intel, err := LoadPortIntel(ctx, tx, p.ID)
	if err != nil {
		return "", err
	}
	if len(intel) == 0 {
		return "No route intel yet. Use SCAN in sectors with ports to record prices.", nil
	}

	discovered, err := loadDiscoveredSectors(ctx, tx, p.ID)
	if err != nil {
		return "", err
	}

	adj, err := loadDiscoveredAdjacency(ctx, tx, discovered)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	sug, ok := BestRouteSuggestion(now, p.SectorID, p.CargoMax, adj, intel, filter)
	if !ok {
		if filter != "" {
			return fmt.Sprintf("No profitable %s route found with current scanned intel.", filter), nil
		}
		return "No profitable route found with current scanned intel.", nil
	}

	lines := make([]string, 0, 12)
	lines = append(lines, "Route suggestion (uses your scanned intel only):")
	lines = append(lines, fmt.Sprintf("Commodity: %s", sug.Commodity))
	lines = append(lines, fmt.Sprintf("Step 1: Travel to Sector %d (%s) in %d move(s).", sug.BuySectorID, sug.BuySectorName, sug.StepsToBuy))
	lines = append(lines, fmt.Sprintf("        Buy at %d credits/unit (scan age %s).", sug.BuyPrice, formatAgeShort(now, sug.BuyScannedAt)))
	lines = append(lines, fmt.Sprintf("Step 2: Travel to Sector %d (%s) in %d move(s).", sug.SellSectorID, sug.SellSectorName, sug.StepsBuyToSell))
	lines = append(lines, fmt.Sprintf("        Sell at %d credits/unit (scan age %s).", sug.SellPrice, formatAgeShort(now, sug.SellScannedAt)))
	lines = append(lines, fmt.Sprintf("Spread: %d/unit | Qty assumed: %d | Profit/trip: %d credits", sug.ProfitPerUnit, sug.TradeQty, sug.ProfitPerTrip))
	lines = append(lines, fmt.Sprintf("Moves: %d | Est. turns (moves + 2 trades): %d | Profit/turn: %.2f", sug.TotalMoves, sug.TotalTurns, float64(sug.ProfitPerTrip)/float64(max(1, sug.TotalTurns))))

	return strings.Join(lines, "\n"), nil
}

func BestRouteSuggestion(now time.Time, currentSector int, cargoMax int, adjacency map[int][]int, intel []PortIntel, commodityFilter string) (RouteSuggestion, bool) {
	if cargoMax < 1 {
		cargoMax = 1
	}
	distFromCurrent := bfsDistances(currentSector, adjacency)

	commodities := []string{"ORE", "ORGANICS", "EQUIPMENT"}
	if commodityFilter != "" {
		commodities = []string{commodityFilter}
	}

	best := RouteSuggestion{}
	bestOK := false

	type quote struct {
		SectorID   int
		SectorName string
		Price      int
		MaxQty     int
		ScannedAt  time.Time
	}

	for _, comm := range commodities {
		buys := make([]quote, 0, 32)
		sells := make([]quote, 0, 32)

		for _, pi := range intel {
			mode, price, qty, baseQty, ok := extractCommodity(pi, comm)
			if !ok {
				continue
			}
			mode = strings.ToUpper(mode)
			if mode == "SELL" {
				maxQty := qty
				if maxQty > 0 {
					buys = append(buys, quote{SectorID: pi.SectorID, SectorName: pi.SectorName, Price: price, MaxQty: maxQty, ScannedAt: pi.ScannedAt})
				}
			}
			if mode == "BUY" {
				demand := baseQty - qty
				if demand > 0 {
					sells = append(sells, quote{SectorID: pi.SectorID, SectorName: pi.SectorName, Price: price, MaxQty: demand, ScannedAt: pi.ScannedAt})
				}
			}
		}

		if len(buys) == 0 || len(sells) == 0 {
			continue
		}

		for _, b := range buys {
			toBuy, ok := distFromCurrent[b.SectorID]
			if !ok {
				continue
			}

			distFromBuy := bfsDistances(b.SectorID, adjacency)

			for _, s := range sells {
				if s.SectorID == b.SectorID {
					continue
				}
				toSell, ok := distFromBuy[s.SectorID]
				if !ok {
					continue
				}
				profitPerUnit := s.Price - b.Price
				if profitPerUnit <= 0 {
					continue
				}

				tradeQty := cargoMax
				if b.MaxQty < tradeQty {
					tradeQty = b.MaxQty
				}
				if s.MaxQty < tradeQty {
					tradeQty = s.MaxQty
				}
				if tradeQty < 1 {
					continue
				}

				totalMoves := toBuy + toSell
				totalTurns := totalMoves + 2
				if totalTurns < 1 {
					totalTurns = 1
				}

				profitTrip := int64(profitPerUnit) * int64(tradeQty)
				baseScore := (profitTrip * 1000) / int64(totalTurns)
				ageBuy := now.Sub(b.ScannedAt)
				ageSell := now.Sub(s.ScannedAt)
				maxAge := ageBuy
				if ageSell > maxAge {
					maxAge = ageSell
				}
				weighted := (baseScore * freshnessWeight(maxAge)) / 1000

				if !bestOK || weighted > best.ScoreX1 {
					bestOK = true
					best = RouteSuggestion{
						Commodity:      comm,
						BuySectorID:    b.SectorID,
						BuySectorName:  b.SectorName,
						SellSectorID:   s.SectorID,
						SellSectorName: s.SectorName,
						BuyPrice:       b.Price,
						SellPrice:      s.Price,
						BuyScannedAt:   b.ScannedAt,
						SellScannedAt:  s.ScannedAt,
						StepsToBuy:     toBuy,
						StepsBuyToSell: toSell,
						TotalMoves:     totalMoves,
						TotalTurns:     totalTurns,
						TradeQty:       tradeQty,
						ProfitPerUnit:  profitPerUnit,
						ProfitPerTrip:  profitTrip,
						ScoreX1:        weighted,
					}
				}
			}
		}
	}

	return best, bestOK
}

func freshnessWeight(age time.Duration) int64 {
	if age < 0 {
		age = 0
	}
	if age <= 15*time.Minute {
		return 1000
	}
	if age <= time.Hour {
		return 900
	}
	if age <= 3*time.Hour {
		return 750
	}
	if age <= 12*time.Hour {
		return 600
	}
	return 500
}

func extractCommodity(pi PortIntel, commodity string) (mode string, price int, qty int, baseQty int, ok bool) {
	switch commodity {
	case "ORE":
		return pi.OreMode, pi.OrePrice, pi.OreQty, pi.OreBaseQty, true
	case "ORGANICS":
		return pi.OrganicsMode, pi.OrganicsPrice, pi.OrganicsQty, pi.OrganicsBaseQty, true
	case "EQUIPMENT":
		return pi.EquipmentMode, pi.EquipmentPrice, pi.EquipmentQty, pi.EquipmentBaseQty, true
	default:
		return "", 0, 0, 0, false
	}
}

func bfsDistances(start int, adjacency map[int][]int) map[int]int {
	dist := map[int]int{start: 0}
	queue := make([]int, 0, 64)
	queue = append(queue, start)

	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		for _, next := range adjacency[n] {
			if _, ok := dist[next]; ok {
				continue
			}
			dist[next] = dist[n] + 1
			queue = append(queue, next)
		}
	}
	return dist
}

func loadDiscoveredSectors(ctx context.Context, tx pgx.Tx, playerID string) (map[int]bool, error) {
	rows, err := tx.Query(ctx, `SELECT sector_id FROM player_discoveries WHERE player_id=$1`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int]bool, 64)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func loadDiscoveredAdjacency(ctx context.Context, tx pgx.Tx, discovered map[int]bool) (map[int][]int, error) {
	rows, err := tx.Query(ctx, `SELECT from_sector, to_sector FROM warps`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	adj := make(map[int][]int, len(discovered))
	for from := range discovered {
		adj[from] = nil
	}

	for rows.Next() {
		var from, to int
		if err := rows.Scan(&from, &to); err != nil {
			return nil, err
		}
		if !discovered[from] || !discovered[to] {
			continue
		}
		adj[from] = append(adj[from], to)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return adj, nil
}
