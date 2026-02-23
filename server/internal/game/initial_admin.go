package game

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sovereignconquest/internal/auth"
	"sovereignconquest/internal/util"
)

type InitialAdminResult struct {
	Created  bool
	Username string
}

// EnsureInitialAdmin creates (if missing) a single seeded admin account intended for
// local/dev bootstrap and server administration. The seeded account always requires
// a password change on first login.
//
// This function is safe to call on every startup.
func EnsureInitialAdmin(ctx context.Context, pool *pgxpool.Pool, username, password string) (InitialAdminResult, error) {
	uname := strings.TrimSpace(username)
	if uname == "" {
		uname = "admin"
	}
	pw := strings.TrimSpace(password)
	if pw == "" {
		pw = "ChangeMeNow!"
	}

	// A starting sector is required by the players.sector_id FK.
	var startSector int
	if err := pool.QueryRow(ctx, "SELECT id FROM sectors ORDER BY id LIMIT 1").Scan(&startSector); err != nil {
		return InitialAdminResult{}, err
	}

	// Ensure there is an active season for the new player.
	seasonID, err := ensureActiveSeason(ctx, pool)
	if err != nil {
		return InitialAdminResult{}, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return InitialAdminResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// If the username exists, do not escalate it to admin automatically.
	var existingUserID string
	var existingIsAdmin bool
	err = tx.QueryRow(ctx, "SELECT id, is_admin FROM users WHERE username=$1", uname).Scan(&existingUserID, &existingIsAdmin)
	if err == nil {
		// Ensure the record is marked admin if it already is; otherwise leave as-is.
		if existingIsAdmin {
			// Ensure a player row exists for login.
			if err := ensurePlayerForUser(ctx, tx, existingUserID, startSector, seasonID); err != nil {
				return InitialAdminResult{}, err
			}
			if err := tx.Commit(ctx); err != nil {
				return InitialAdminResult{}, err
			}
			return InitialAdminResult{Created: false, Username: uname}, nil
		}
		// Username exists but is not admin; do not promote.
		if err := tx.Commit(ctx); err != nil {
			return InitialAdminResult{}, err
		}
		return InitialAdminResult{Created: false, Username: uname}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return InitialAdminResult{}, err
	}

	userID, err := util.NewID()
	if err != nil {
		return InitialAdminResult{}, err
	}
	playerID, err := util.NewID()
	if err != nil {
		return InitialAdminResult{}, err
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		return InitialAdminResult{}, err
	}

	// Create the admin user.
	if _, err := tx.Exec(ctx, `
		INSERT INTO users(id, username, password_hash, is_admin, must_change_password)
		VALUES ($1,$2,$3,true,true)
	`, userID, uname, hash); err != nil {
		return InitialAdminResult{}, err
	}

	// Create matching player row.
	if _, err := tx.Exec(ctx, `
		INSERT INTO players(id, user_id, credits, turns, turns_max, sector_id, cargo_max, last_turn_regen, season_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, playerID, userID, int64(100000), 9999, 9999, startSector, 30, time.Now().UTC(), seasonID); err != nil {
		return InitialAdminResult{}, err
	}

	_ = MarkDiscovered(ctx, tx, playerID, startSector)
	_ = InsertLog(ctx, tx, playerID, "SYSTEM", "Initial admin account created. You must change the default password before playing.")

	if err := tx.Commit(ctx); err != nil {
		return InitialAdminResult{}, err
	}

	return InitialAdminResult{Created: true, Username: uname}, nil
}

func ensureActiveSeason(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) (int, error) {
	var seasonID int
	err := q.QueryRow(ctx, "SELECT id FROM seasons WHERE active=true ORDER BY id DESC LIMIT 1").Scan(&seasonID)
	if err == nil {
		return seasonID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}

	// No active season: create a new one.
	var next int
	_ = q.QueryRow(ctx, "SELECT COALESCE(MAX(id),0)+1 FROM seasons").Scan(&next)
	name := fmt.Sprintf("Season %d", next)
	err = q.QueryRow(ctx, "INSERT INTO seasons(name, active, started_at) VALUES ($1,true,now()) RETURNING id", name).Scan(&seasonID)
	if err != nil {
		return 0, err
	}
	return seasonID, nil
}

func ensurePlayerForUser(ctx context.Context, tx pgx.Tx, userID string, startSector, seasonID int) error {
	var pid string
	err := tx.QueryRow(ctx, "SELECT id FROM players WHERE user_id=$1", userID).Scan(&pid)
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	playerID, err := util.NewID()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO players(id, user_id, credits, turns, turns_max, sector_id, cargo_max, last_turn_regen, season_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, playerID, userID, int64(1000), 100, 100, startSector, 30, time.Now().UTC(), seasonID)
	if err != nil {
		return err
	}
	_ = MarkDiscovered(ctx, tx, playerID, startSector)
	return nil
}
