package game

import "testing"

func TestRankNamesLengthMatchesMaxLevel(t *testing.T) {
	if got, want := len(RankNames), MaxPlayerLevel; got != want {
		t.Fatalf("RankNames length mismatch: got %d want %d", got, want)
	}
}

func TestRankNameForLevel_Clamps(t *testing.T) {
	if got := RankNameForLevel(1); got != RankNames[0] {
		t.Fatalf("level 1 rank: got %q want %q", got, RankNames[0])
	}
	if got := RankNameForLevel(0); got != RankNames[0] {
		t.Fatalf("level 0 clamp: got %q want %q", got, RankNames[0])
	}
	if got := RankNameForLevel(MaxPlayerLevel); got != RankNames[MaxPlayerLevel-1] {
		t.Fatalf("max level rank: got %q want %q", got, RankNames[MaxPlayerLevel-1])
	}
	if got := RankNameForLevel(MaxPlayerLevel + 100); got != RankNames[MaxPlayerLevel-1] {
		t.Fatalf("over-max clamp: got %q want %q", got, RankNames[MaxPlayerLevel-1])
	}
}

func TestXPLevelRoundTrip(t *testing.T) {
	// Level 1 is always 0 XP.
	if got := XPForLevel(1); got != 0 {
		t.Fatalf("XPForLevel(1): got %d want 0", got)
	}
	// A small table of sanity points.
	points := []struct {
		level int
		xp    int64
	}{
		{level: 1, xp: 0},
		{level: 2, xp: XPForLevel(2)},
		{level: 3, xp: XPForLevel(3)},
		{level: 10, xp: XPForLevel(10)},
		{level: MaxPlayerLevel, xp: XPForLevel(MaxPlayerLevel)},
	}

	for _, p := range points {
		if got := LevelForXP(p.xp); got != p.level {
			t.Fatalf("LevelForXP(XPForLevel(%d)) => %d", p.level, got)
		}
	}

	// XP just below a threshold should still be the previous level.
	if MaxPlayerLevel >= 3 {
		thr := XPForLevel(3)
		if got := LevelForXP(thr - 1); got != 2 {
			t.Fatalf("LevelForXP(thr-1): got %d want 2", got)
		}
	}
}
