package game

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SoftWipeRequest struct {
	SeasonName string
	ResetCorps bool
}

type SoftWipeResult struct {
	NewSeasonID   int       `json:"new_season_id"`
	NewSeasonName string    `json:"new_season_name"`
	PlayersReset  int64     `json:"players_reset"`
	CorpsReset    bool      `json:"corps_reset"`
	At            time.Time `json:"at"`
}

func SoftWipe(ctx context.Context, pool *pgxpool.Pool, req SoftWipeRequest) (SoftWipeResult, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return SoftWipeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, _ = tx.Exec(ctx, "UPDATE seasons SET active=false, ended_at=now() WHERE active=true")

	seasonName := strings.TrimSpace(req.SeasonName)
	if seasonName == "" {
		var next int
		_ = tx.QueryRow(ctx, "SELECT COALESCE(MAX(id),0)+1 FROM seasons").Scan(&next)
		seasonName = fmt.Sprintf("Season %d", next)
	}

	var newID int
	if err := tx.QueryRow(ctx, "INSERT INTO seasons(name, active, started_at) VALUES ($1,true,now()) RETURNING id", seasonName).Scan(&newID); err != nil {
		return SoftWipeResult{}, err
	}

	cmdTag, err := tx.Exec(ctx, `
		UPDATE players SET
			credits = 1000,
			turns = 100,
			turns_max = 100,
			sector_id = 1,
			cargo_max = 30,
			cargo_ore = 0,
			cargo_organics = 0,
			cargo_equipment = 0,
			last_turn_regen = now(),
			season_id = $1
	`, newID)
	if err != nil {
		return SoftWipeResult{}, err
	}

	// Fog-of-war and activity are per-season in practice.
	_, _ = tx.Exec(ctx, "DELETE FROM player_discoveries")
	_, _ = tx.Exec(ctx, "DELETE FROM player_sector_intel")
	_, _ = tx.Exec(ctx, "DELETE FROM logs")
	_, _ = tx.Exec(ctx, "DELETE FROM events")

	// Clear deployed assets.
	_, _ = tx.Exec(ctx, "DELETE FROM mines")
	_, _ = tx.Exec(ctx, "UPDATE planets SET owner_player_id=NULL, owner_corp_id=NULL, storage_ore=0, storage_organics=0, storage_equipment=0, citadel_level=0")

	if req.ResetCorps {
		_, _ = tx.Exec(ctx, "DELETE FROM corp_messages")
		_, _ = tx.Exec(ctx, "DELETE FROM corp_members")
		_, _ = tx.Exec(ctx, "DELETE FROM corporations")
	} else {
		_, _ = tx.Exec(ctx, "DELETE FROM corp_messages")
		_, _ = tx.Exec(ctx, "UPDATE corporations SET credits=0")
	}

	if err := tx.Commit(ctx); err != nil {
		return SoftWipeResult{}, err
	}

	return SoftWipeResult{
		NewSeasonID:   newID,
		NewSeasonName: seasonName,
		PlayersReset:  cmdTag.RowsAffected(),
		CorpsReset:    req.ResetCorps,
		At:            time.Now().UTC(),
	}, nil
}
