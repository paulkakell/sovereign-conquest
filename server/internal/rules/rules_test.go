package rules

import "testing"

func TestMineTriggerCount(t *testing.T) {
	cases := []struct {
		total int
		want  int
	}{
		{0, 0},
		{1, 1},
		{3, 3},
		{MineTriggerCap, MineTriggerCap},
		{MineTriggerCap + 10, MineTriggerCap},
	}
	for _, c := range cases {
		if got := MineTriggerCount(c.total); got != c.want {
			t.Fatalf("MineTriggerCount(%d) = %d, want %d", c.total, got, c.want)
		}
	}
}

func TestMineDamageCredits(t *testing.T) {
	if MineDamageCredits(0) != 0 {
		t.Fatalf("expected 0 for triggered=0")
	}
	if MineDamageCredits(1) != MineDamagePerMine {
		t.Fatalf("expected %d for triggered=1", MineDamagePerMine)
	}
	if MineDamageCredits(5) != 5*MineDamagePerMine {
		t.Fatalf("expected %d for triggered=5", 5*MineDamagePerMine)
	}
}

func TestCitadelUpgradeCost(t *testing.T) {
	if CitadelUpgradeCost(1) != CitadelUpgradeBase {
		t.Fatalf("level 1 cost mismatch")
	}
	if CitadelUpgradeCost(3) != 3*CitadelUpgradeBase {
		t.Fatalf("level 3 cost mismatch")
	}
	if CitadelUpgradeCost(0) != CitadelUpgradeBase {
		t.Fatalf("level 0 should clamp to 1")
	}
}

func TestSanitizeSingleLine(t *testing.T) {
	in := "  Hello\nworld  \t  "
	out := SanitizeSingleLine(in)
	if out != "Hello world" {
		t.Fatalf("sanitize mismatch: %q", out)
	}
}

func TestValidateCorpName(t *testing.T) {
	name, ok := ValidateCorpName("  The Syndicate ", 3, 32)
	if !ok || name != "The Syndicate" {
		t.Fatalf("expected valid corp name")
	}
	_, ok = ValidateCorpName("!!bad!!", 3, 32)
	if ok {
		t.Fatalf("expected invalid name")
	}
}
