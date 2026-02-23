package game

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func executeMarketCommand(ctx context.Context, tx pgx.Tx, p Player, cmd CommandRequest) (string, error) {
	filter := strings.ToUpper(strings.TrimSpace(cmd.Commodity))
	if filter != "" && filter != "ORE" && filter != "ORGANICS" && filter != "EQUIPMENT" {
		filter = ""
	}

	intel, err := LoadPortIntel(ctx, tx, p.ID)
	if err != nil {
		return "", err
	}
	if len(intel) == 0 {
		return "No market intel yet. Use SCAN in sectors with ports to record prices.", nil
	}

	now := time.Now().UTC()

	type quote struct {
		SectorID   int
		SectorName string
		Price      int
		Qty        int
		BaseQty    int
		ScannedAt  time.Time
	}

	inf := int(^uint(0) >> 1)

	bestBuy := map[string]quote{
		"ORE":       {Price: inf},
		"ORGANICS":  {Price: inf},
		"EQUIPMENT": {Price: inf},
	}
	bestSell := map[string]quote{
		"ORE":       {Price: -1},
		"ORGANICS":  {Price: -1},
		"EQUIPMENT": {Price: -1},
	}
	buyCount := map[string]int{"ORE": 0, "ORGANICS": 0, "EQUIPMENT": 0}
	sellCount := map[string]int{"ORE": 0, "ORGANICS": 0, "EQUIPMENT": 0}

	consider := func(comm string, mode string, price int, qty int, baseQty int, scannedAt time.Time, sectorID int, sectorName string) {
		mode = strings.ToUpper(mode)
		if mode == "SELL" {
			buyCount[comm]++
			q := bestBuy[comm]
			if price > 0 && price < q.Price {
				bestBuy[comm] = quote{SectorID: sectorID, SectorName: sectorName, Price: price, Qty: qty, BaseQty: baseQty, ScannedAt: scannedAt}
			}
		}
		if mode == "BUY" {
			sellCount[comm]++
			q := bestSell[comm]
			if price > q.Price {
				bestSell[comm] = quote{SectorID: sectorID, SectorName: sectorName, Price: price, Qty: qty, BaseQty: baseQty, ScannedAt: scannedAt}
			}
		}
	}

	for _, pi := range intel {
		consider("ORE", pi.OreMode, pi.OrePrice, pi.OreQty, pi.OreBaseQty, pi.ScannedAt, pi.SectorID, pi.SectorName)
		consider("ORGANICS", pi.OrganicsMode, pi.OrganicsPrice, pi.OrganicsQty, pi.OrganicsBaseQty, pi.ScannedAt, pi.SectorID, pi.SectorName)
		consider("EQUIPMENT", pi.EquipmentMode, pi.EquipmentPrice, pi.EquipmentQty, pi.EquipmentBaseQty, pi.ScannedAt, pi.SectorID, pi.SectorName)
	}

	lines := make([]string, 0, 12)
	lines = append(lines, fmt.Sprintf("Market intel: %d scanned ports.", len(intel)))
	lines = append(lines, "Note: MARKET uses your scanned intel (SCAN) only; remote prices may be stale.")

	appendCommodity := func(comm string) {
		b := bestBuy[comm]
		s := bestSell[comm]
		if buyCount[comm] == 0 || sellCount[comm] == 0 || b.Price == inf || s.Price < 0 {
			lines = append(lines, fmt.Sprintf("%s: insufficient intel (need at least one SELL and one BUY port scanned).", comm))
			return
		}
		spread := s.Price - b.Price
		buyStr := fmt.Sprintf("BUY @ Sector %d (%s) %d cr (qty %d/%d, age %s)", b.SectorID, b.SectorName, b.Price, b.Qty, b.BaseQty, formatAgeShort(now, b.ScannedAt))
		sellStr := fmt.Sprintf("SELL @ Sector %d (%s) %d cr (qty %d/%d, age %s)", s.SectorID, s.SectorName, s.Price, s.Qty, s.BaseQty, formatAgeShort(now, s.ScannedAt))
		lines = append(lines, fmt.Sprintf("%s: %s | %s | spread %d/unit", comm, buyStr, sellStr, spread))
	}

	if filter != "" {
		appendCommodity(filter)
	} else {
		appendCommodity("ORE")
		appendCommodity("ORGANICS")
		appendCommodity("EQUIPMENT")
	}

	return strings.Join(lines, "\n"), nil
}
