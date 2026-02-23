package game

import "testing"

func TestPricePerUnitWithPercent(t *testing.T) {
	basePrice := 10
	baseQty := 100
	qty := 100

	if got := PricePerUnitWithPercent(basePrice, baseQty, qty, 80); got != 8 {
		t.Fatalf("expected 8, got %d", got)
	}
	if got := PricePerUnitWithPercent(basePrice, baseQty, qty, 130); got != 13 {
		t.Fatalf("expected 13, got %d", got)
	}
	// Clamp low
	if got := PricePerUnitWithPercent(basePrice, baseQty, qty, 0); got != 1 {
		t.Fatalf("expected 1 (clamped), got %d", got)
	}
	// Clamp high
	if got := PricePerUnitWithPercent(basePrice, baseQty, qty, 500); got != 30 {
		t.Fatalf("expected 30 (clamped), got %d", got)
	}
}
