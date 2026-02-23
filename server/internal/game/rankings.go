package game

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func executeRankingsCommand(ctx context.Context, tx pgx.Tx, p Player) (string, error) {
	rows, err := tx.Query(ctx, `
		SELECT
			u.username,
			pl.credits,
			COALESCE(c.name, '')
		FROM players pl
		JOIN users u ON u.id = pl.user_id
		LEFT JOIN corp_members cm ON cm.player_id = pl.id
		LEFT JOIN corporations c ON c.id = cm.corp_id
		WHERE pl.season_id = $1
		ORDER BY pl.credits DESC, u.username ASC
		LIMIT 10
	`, p.SeasonID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	lines := []string{fmt.Sprintf("Rankings (%s)", p.SeasonName)}
	i := 0
	for rows.Next() {
		var username string
		var credits int64
		var corp string
		if err := rows.Scan(&username, &credits, &corp); err != nil {
			return "", err
		}
		i++
		label := username
		if corp != "" {
			label = fmt.Sprintf("%s [%s]", username, corp)
		}
		lines = append(lines, fmt.Sprintf("%d. %s - %d", i, label, credits))
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if i == 0 {
		lines = append(lines, "No rankings yet.")
	}
	return strings.Join(lines, "\n"), nil
}

func executeSeasonCommand(ctx context.Context, tx pgx.Tx, p Player) (string, error) {
	var started time.Time
	err := tx.QueryRow(ctx, "SELECT started_at FROM seasons WHERE id=$1", p.SeasonID).Scan(&started)
	if err != nil {
		return "", err
	}
	lines := []string{
		fmt.Sprintf("Current season: %s (ID %d)", p.SeasonName, p.SeasonID),
		fmt.Sprintf("Started: %s", started.UTC().Format(time.RFC3339)),
	}
	return strings.Join(lines, "\n"), nil
}
