package game

import (
	"testing"
	"time"
)

func TestBestRouteSuggestionSelectsBest(t *testing.T) {
	now := time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC)

	adj := map[int][]int{
		1: {2},
		2: {1, 3},
		3: {2, 4},
		4: {3},
	}

	intel := []PortIntel{
		{SectorID: 2, SectorName: "S2", ScannedAt: now.Add(-10 * time.Minute), OreMode: "SELL", OrePrice: 5, OreQty: 100, OreBaseQty: 100},
		{SectorID: 3, SectorName: "S3", ScannedAt: now.Add(-10 * time.Minute), OreMode: "SELL", OrePrice: 6, OreQty: 100, OreBaseQty: 100},
		{SectorID: 4, SectorName: "S4", ScannedAt: now.Add(-10 * time.Minute), OreMode: "BUY", OrePrice: 10, OreQty: 0, OreBaseQty: 100},
	}

	sug, ok := BestRouteSuggestion(now, 1, 10, adj, intel, "ORE")
	if !ok {
		t.Fatalf("expected ok")
	}
	if sug.BuySectorID != 2 || sug.SellSectorID != 4 {
		t.Fatalf("expected buy 2 sell 4, got buy %d sell %d", sug.BuySectorID, sug.SellSectorID)
	}
	if sug.TradeQty != 10 {
		t.Fatalf("expected trade qty 10, got %d", sug.TradeQty)
	}
	if sug.ProfitPerUnit != 5 {
		t.Fatalf("expected profit/unit 5, got %d", sug.ProfitPerUnit)
	}
	if sug.TotalTurns != 5 {
		t.Fatalf("expected total turns 5, got %d", sug.TotalTurns)
	}
}

func TestBestRouteSuggestionPenalizesStaleIntel(t *testing.T) {
	now := time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC)

	adj := map[int][]int{
		1: {2},
		2: {1, 3},
		3: {2, 4},
		4: {3, 5},
		5: {4},
	}

	stale := now.Add(-13 * time.Hour)
	fresh := now.Add(-5 * time.Minute)

	intel := []PortIntel{
		// Stale, high-profit route: 1->2 (buy), 2->4 (sell)
		{SectorID: 2, SectorName: "S2", ScannedAt: stale, OreMode: "SELL", OrePrice: 5, OreQty: 100, OreBaseQty: 100},
		{SectorID: 4, SectorName: "S4", ScannedAt: stale, OreMode: "BUY", OrePrice: 15, OreQty: 0, OreBaseQty: 100},
		// Fresh, slightly lower-profit route: 1->3 (buy), 3->5 (sell)
		{SectorID: 3, SectorName: "S3", ScannedAt: fresh, OreMode: "SELL", OrePrice: 6, OreQty: 100, OreBaseQty: 100},
		{SectorID: 5, SectorName: "S5", ScannedAt: fresh, OreMode: "BUY", OrePrice: 13, OreQty: 0, OreBaseQty: 100},
	}

	sug, ok := BestRouteSuggestion(now, 1, 10, adj, intel, "ORE")
	if !ok {
		t.Fatalf("expected ok")
	}
	if sug.BuySectorID != 3 || sug.SellSectorID != 5 {
		t.Fatalf("expected fresh route buy 3 sell 5, got buy %d sell %d", sug.BuySectorID, sug.SellSectorID)
	}
}
