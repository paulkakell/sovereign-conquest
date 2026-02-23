package game

import (
	"fmt"
	"time"
)

func formatDurationShort(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		sec := int(d.Seconds())
		if sec < 1 {
			sec = 1
		}
		return fmt.Sprintf("%ds", sec)
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh%dm", h, m)
	}

	days := int(d.Hours()) / 24
	h := int(d.Hours()) % 24
	if h == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd%dh", days, h)
}

func formatAgeShort(now time.Time, t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return formatDurationShort(now.Sub(t))
}
