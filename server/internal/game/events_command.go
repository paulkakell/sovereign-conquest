package game

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func executeEventsCommand(ctx context.Context, tx pgx.Tx, p Player) (string, error) {
	rows, err := tx.Query(ctx, `
		SELECT
			e.kind,
			e.sector_id,
			s.name,
			e.commodity,
			e.price_percent,
			e.severity,
			e.title,
			e.description,
			e.ends_at
		FROM events e
		JOIN sectors s ON s.id = e.sector_id
		JOIN player_discoveries d ON d.sector_id = e.sector_id AND d.player_id = $1
		WHERE e.active=true AND e.ends_at > now()
		ORDER BY e.ends_at ASC
		LIMIT 20
	`, p.ID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	type rowT struct {
		Kind         string
		SectorID     int
		SectorName   string
		Commodity    string
		PricePercent int
		Severity     int
		Title        string
		Description  string
		EndsAt       time.Time
	}

	list := make([]rowT, 0, 8)
	for rows.Next() {
		var r rowT
		if err := rows.Scan(&r.Kind, &r.SectorID, &r.SectorName, &r.Commodity, &r.PricePercent, &r.Severity, &r.Title, &r.Description, &r.EndsAt); err != nil {
			return "", err
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "No known active events right now.", nil
	}

	lines := make([]string, 0, 1+len(list))
	lines = append(lines, "Active events (known sectors):")

	for _, r := range list {
		remaining := time.Until(r.EndsAt)
		if remaining < 0 {
			remaining = 0
		}
		effect := ""
		switch strings.ToUpper(r.Kind) {
		case "ANOMALY", "LIMITED":
			comm := strings.ToUpper(strings.TrimSpace(r.Commodity))
			if comm == "" {
				comm = "ALL"
			}
			effect = fmt.Sprintf("prices %d%% (%s)", r.PricePercent, comm)
		case "INVASION":
			penalty := int64(r.Severity) * 200
			effect = fmt.Sprintf("entry penalty ~%d credits", penalty)
		default:
			effect = ""
		}

		if effect != "" {
			lines = append(lines, fmt.Sprintf("- [%s] Sector %d (%s): %s (ends in %s) | %s", strings.ToUpper(r.Kind), r.SectorID, r.SectorName, r.Title, formatDurationShort(remaining), effect))
		} else {
			lines = append(lines, fmt.Sprintf("- [%s] Sector %d (%s): %s (ends in %s)", strings.ToUpper(r.Kind), r.SectorID, r.SectorName, r.Title, formatDurationShort(remaining)))
		}

		desc := strings.TrimSpace(r.Description)
		if desc != "" {
			// Keep descriptions readable in the terminal pane.
			lines = append(lines, fmt.Sprintf("  %s", desc))
		}
	}
	return strings.Join(lines, "\n"), nil
}
