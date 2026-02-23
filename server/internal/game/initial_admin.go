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
	Created       bool
	Promoted      bool
	PasswordReset bool
	Username      string
}

type initialAdminPlan int

const (
	planCreateNew initialAdminPlan = iota
	planEnsureExistingAdmin
	planNoopExistingNonAdmin
	planPromoteExisting
)

func decideInitialAdminPlan(userFound bool, existingIsAdmin bool, anyAdminsExist bool) initialAdminPlan {
	if !userFound {
		return planCreateNew
	}
	if existingIsAdmin {
		return planEnsureExistingAdmin
	}
	if anyAdminsExist {
		return planNoopExistingNonAdmin
	}
	return planPromoteExisting
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

	// Lookup whether the configured initial admin username already exists.
	var existingUserID string
	var existingIsAdmin bool
	userLookupErr := tx.QueryRow(ctx, "SELECT id, is_admin FROM users WHERE username=$1", uname).Scan(&existingUserID, &existingIsAdmin)
	userFound := userLookupErr == nil
	if userLookupErr != nil && !errors.Is(userLookupErr, pgx.ErrNoRows) {
		return InitialAdminResult{}, userLookupErr
	}

	// Determine whether any admin user exists (needed to safely decide promotion behavior).
	anyAdminsExist := false
	var ok int
	if err := tx.QueryRow(ctx, "SELECT 1 FROM users WHERE is_admin=true LIMIT 1").Scan(&ok); err == nil {
		anyAdminsExist = true
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return InitialAdminResult{}, err
	}

	plan := decideInitialAdminPlan(userFound, existingIsAdmin, anyAdminsExist)
	if plan == planNoopExistingNonAdmin {
		// Username exists but is not admin, and an admin already exists elsewhere.
		if err := tx.Commit(ctx); err != nil {
			return InitialAdminResult{}, err
		}
		return InitialAdminResult{Created: false, Username: uname}, nil
	}

	if plan == planEnsureExistingAdmin {
		// Ensure a player row exists for login.
		if err := ensurePlayerForUser(ctx, tx, existingUserID, startSector, seasonID); err != nil {
			return InitialAdminResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return InitialAdminResult{}, err
		}
		return InitialAdminResult{Created: false, Username: uname}, nil
	}

	if plan == planPromoteExisting {
		// Recovery path: if no admins exist, promote the configured username to admin and
		// reset its password to the configured initial password.
		//
		// This prevents an "admin" username collision from permanently blocking bootstrap.
		hash, err := auth.HashPassword(pw)
		if err != nil {
			return InitialAdminResult{}, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE users
			SET is_admin=true,
				must_change_password=true,
				password_hash=$2,
				password_changed_at=now()
			WHERE id=$1
		`, existingUserID, hash); err != nil {
			return InitialAdminResult{}, err
		}
		if err := ensurePlayerForUser(ctx, tx, existingUserID, startSector, seasonID); err != nil {
			return InitialAdminResult{}, err
		}
		var pid string
		if err := tx.QueryRow(ctx, "SELECT id FROM players WHERE user_id=$1", existingUserID).Scan(&pid); err == nil {
			_ = MarkDiscovered(ctx, tx, pid, startSector)
			_ = InsertLog(ctx, tx, pid, "SYSTEM", "Initial admin bootstrap: account promoted to admin; password reset required.")
		}
		if err := tx.Commit(ctx); err != nil {
			return InitialAdminResult{}, err
		}
		return InitialAdminResult{Created: false, Promoted: true, PasswordReset: true, Username: uname}, nil
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
