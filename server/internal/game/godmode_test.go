package game

import "testing"

func TestEffectiveCommandCost_AdminAlwaysZero(t *testing.T) {
	admin := Player{IsAdmin: true}

	if got := effectiveCommandCost(admin, CommandRequest{Type: "SCAN"}); got != 0 {
		t.Fatalf("admin SCAN cost: got %d want 0", got)
	}
	if got := effectiveCommandCost(admin, CommandRequest{Type: "MOVE"}); got != 0 {
		t.Fatalf("admin MOVE cost: got %d want 0", got)
	}
	if got := effectiveCommandCost(admin, CommandRequest{Type: "TRADE"}); got != 0 {
		t.Fatalf("admin TRADE cost: got %d want 0", got)
	}
}

func TestEffectiveCommandCost_NormalUsesCommandCost(t *testing.T) {
	p := Player{IsAdmin: false}
	if got := effectiveCommandCost(p, CommandRequest{Type: "SCAN"}); got != 1 {
		t.Fatalf("player SCAN cost: got %d want 1", got)
	}
	if got := effectiveCommandCost(p, CommandRequest{Type: "HELP"}); got != 0 {
		t.Fatalf("player HELP cost: got %d want 0", got)
	}
}
