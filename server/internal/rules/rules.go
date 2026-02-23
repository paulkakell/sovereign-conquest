package rules

import (
	"regexp"
	"strings"
)

const (
	MineTriggerCap    = 5
	MineDamagePerMine = int64(50)

	CitadelUpgradeBase = int64(5000)
)

func MineTriggerCount(totalHostile int) int {
	if totalHostile < 1 {
		return 0
	}
	if totalHostile > MineTriggerCap {
		return MineTriggerCap
	}
	return totalHostile
}

func MineDamageCredits(triggered int) int64 {
	if triggered < 1 {
		return 0
	}
	return int64(triggered) * MineDamagePerMine
}

func CitadelUpgradeCost(nextLevel int) int64 {
	if nextLevel < 1 {
		nextLevel = 1
	}
	return int64(nextLevel) * CitadelUpgradeBase
}

func SanitizeSingleLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.TrimSpace(s)
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

var corpNameRe = regexp.MustCompile(`^[A-Za-z0-9 _\-]+$`)

func ValidateCorpName(name string, minLen, maxLen int) (string, bool) {
	name = SanitizeSingleLine(name)
	if len(name) < minLen || len(name) > maxLen {
		return name, false
	}
	if !corpNameRe.MatchString(name) {
		return name, false
	}
	return name, true
}
