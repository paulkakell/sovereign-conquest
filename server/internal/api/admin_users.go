package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"sovereignconquest/internal/auth"
	"sovereignconquest/internal/game"
)

const adminUserColumns = `id, username, is_admin, must_change_password,
	password_changed_at, created_at, account_revision, account_status,
	suspended_until, moderation_reason, moderated_at`

// Credentials and session counters are deliberately absent from account responses.
type adminUser struct {
	ID                 string       `json:"id"`
	Username           string       `json:"username"`
	IsAdmin            bool         `json:"is_admin"`
	MustChangePassword bool         `json:"must_change_password"`
	PasswordChangedAt  *time.Time   `json:"password_changed_at"`
	CreatedAt          time.Time    `json:"created_at"`
	Revision           int64        `json:"revision"`
	Status             string       `json:"status"`
	EffectiveStatus    string       `json:"effective_status"`
	SuspendedUntil     *time.Time   `json:"suspended_until"`
	ModerationReason   string       `json:"moderation_reason"`
	ModeratedAt        *time.Time   `json:"moderated_at"`
	Player             *adminPlayer `json:"player,omitempty"`
}

type adminPlayer struct {
	ID                string    `json:"id"`
	Credits           int64     `json:"credits,string"`
	XP                int64     `json:"xp,string"`
	Level             int       `json:"level"`
	ShipType          string    `json:"ship_type"`
	ShipCargoUpgrades int       `json:"ship_cargo_upgrades"`
	ShipTurnUpgrades  int       `json:"ship_turn_upgrades"`
	Turns             int       `json:"turns"`
	TurnsMax          int       `json:"turns_max"`
	SectorID          int       `json:"sector_id"`
	CargoMax          int       `json:"cargo_max"`
	CargoOre          int       `json:"cargo_ore"`
	CargoOrganics     int       `json:"cargo_organics"`
	CargoEquipment    int       `json:"cargo_equipment"`
	LastTurnRegen     time.Time `json:"last_turn_regen"`
	SeasonID          int       `json:"season_id"`
	CreatedAt         time.Time `json:"created_at"`
}

type adminPlayerPatch struct {
	Credits           *int64  `json:"credits,string"`
	XP                *int64  `json:"xp,string"`
	ShipType          *string `json:"ship_type"`
	ShipCargoUpgrades *int    `json:"ship_cargo_upgrades"`
	ShipTurnUpgrades  *int    `json:"ship_turn_upgrades"`
	Turns             *int    `json:"turns"`
	TurnsMax          *int    `json:"turns_max"`
	SectorID          *int    `json:"sector_id"`
	CargoMax          *int    `json:"cargo_max"`
	CargoOre          *int    `json:"cargo_ore"`
	CargoOrganics     *int    `json:"cargo_organics"`
	CargoEquipment    *int    `json:"cargo_equipment"`
	SeasonID          *int    `json:"season_id"`
}

type adminUserPatch struct {
	Revision           *int64            `json:"revision"`
	Username           *string           `json:"username"`
	IsAdmin            *bool             `json:"is_admin"`
	MustChangePassword *bool             `json:"must_change_password"`
	NewPassword        *string           `json:"new_password"`
	Player             *adminPlayerPatch `json:"player"`
}

type moderationRequest struct {
	Revision       *int64     `json:"revision"`
	Action         string     `json:"action"`
	Reason         string     `json:"reason"`
	SuspendedUntil *time.Time `json:"suspended_until"`
}

func scanAdminUser(row pgx.Row) (adminUser, error) {
	var u adminUser
	err := row.Scan(&u.ID, &u.Username, &u.IsAdmin, &u.MustChangePassword,
		&u.PasswordChangedAt, &u.CreatedAt, &u.Revision, &u.Status,
		&u.SuspendedUntil, &u.ModerationReason, &u.ModeratedAt)
	u.EffectiveStatus = (auth.Session{Status: u.Status, SuspendedUntil: u.SuspendedUntil}).EffectiveStatus(time.Now().UTC())
	return u, err
}

func loadAdminUser(ctx context.Context, tx pgx.Tx, id string, lock bool) (adminUser, error) {
	query := `SELECT ` + adminUserColumns + ` FROM users WHERE id=$1`
	if lock {
		query += ` FOR UPDATE`
	}
	u, err := scanAdminUser(tx.QueryRow(ctx, query, id))
	if err != nil {
		return u, err
	}
	query = `SELECT id, credits, xp, level, ship_type, ship_cargo_upgrades,
		ship_turn_upgrades, turns, turns_max, sector_id, cargo_max, cargo_ore,
		cargo_organics, cargo_equipment, last_turn_regen, season_id, created_at
		FROM players WHERE user_id=$1 ORDER BY id LIMIT 1`
	if lock {
		query += ` FOR UPDATE`
	}
	var p adminPlayer
	err = tx.QueryRow(ctx, query, id).Scan(&p.ID, &p.Credits, &p.XP, &p.Level,
		&p.ShipType, &p.ShipCargoUpgrades, &p.ShipTurnUpgrades, &p.Turns,
		&p.TurnsMax, &p.SectorID, &p.CargoMax, &p.CargoOre, &p.CargoOrganics,
		&p.CargoEquipment, &p.LastTurnRegen, &p.SeasonID, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, nil
	}
	if err == nil {
		p.Level = game.LevelForXP(p.XP)
		u.Player = &p
	}
	return u, err
}

func decodeAdminJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request: check field names, types and JSON size")
		return false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request must contain one JSON object")
		return false
	}
	return true
}

func adminDatabaseError(w http.ResponseWriter, err error) {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, http.StatusNotFound, "user not found")
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		writeError(w, http.StatusConflict, "username unavailable")
	case errors.As(err, &pgErr) && pgErr.Code == "23503":
		writeError(w, http.StatusBadRequest, "sector or season does not exist")
	default:
		slog.Error("admin_user_database_error")
		writeError(w, http.StatusInternalServerError, "Unable to save account. No changes were committed.")
	}
}

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	status := r.URL.Query().Get("status")
	after := r.URL.Query().Get("after")
	if len(query) > 64 || len(after) > 256 || (status != "" && status != "active" && status != "suspended" && status != "banned") {
		writeError(w, http.StatusBadRequest, "invalid search or status filter")
		return
	}
	// Cursor pagination is stable, bounded and includes accounts without players.
	rows, err := s.Pool.Query(r.Context(), `SELECT `+adminUserColumns+` FROM users
		WHERE ($1='' OR strpos(lower(username), lower($1)) > 0 OR id=$1)
		AND ($2='' OR (CASE WHEN account_status='suspended' AND suspended_until <= now()
		THEN 'active' ELSE account_status END)=$2)
		AND ($3='' OR id > $3) ORDER BY id LIMIT 51`, query, status, after)
	if err != nil {
		adminDatabaseError(w, err)
		return
	}
	defer rows.Close()
	users := make([]adminUser, 0, 50)
	for rows.Next() {
		u, err := scanAdminUser(rows)
		if err != nil {
			adminDatabaseError(w, err)
			return
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		adminDatabaseError(w, err)
		return
	}
	cursor := ""
	if len(users) > 50 {
		users = users[:50]
		cursor = users[49].ID
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": users, "next_cursor": cursor})
}

func (s *Server) handleAdminUser(w http.ResponseWriter, r *http.Request) {
	tx, err := s.Pool.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		adminDatabaseError(w, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	u, err := loadAdminUser(r.Context(), tx, chi.URLParam(r, "id"), false)
	if err != nil {
		adminDatabaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u})
}

// Serialize admin mutations, then recheck the actor inside the transaction. This
// prevents two admins concurrently removing each other's remaining access.
func (s *Server) beginAdminMutation(w http.ResponseWriter, r *http.Request, revision *int64) (pgx.Tx, adminUser, bool) {
	if revision == nil || *revision < 0 {
		writeError(w, http.StatusBadRequest, "revision is required; reload the account before saving")
		return nil, adminUser{}, false
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		adminDatabaseError(w, err)
		return nil, adminUser{}, false
	}
	ok := false
	defer func() {
		if !ok {
			_ = tx.Rollback(r.Context())
		}
	}()
	if _, err := tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(7107001)`); err != nil {
		adminDatabaseError(w, err)
		return nil, adminUser{}, false
	}
	claims, _ := r.Context().Value(ctxClaims).(*auth.Claims)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "administrator authentication required")
		return nil, adminUser{}, false
	}
	// The actor lock also serializes self-service password changes with this save.
	var actorID string
	if err := tx.QueryRow(r.Context(), `SELECT id FROM users WHERE id=$1 FOR UPDATE`, claims.UserID).Scan(&actorID); err != nil {
		adminDatabaseError(w, err)
		return nil, adminUser{}, false
	}
	session, err := auth.LoadSession(r.Context(), tx, claims.UserID, claims.PlayerID)
	if !checkSession(w, session, claims, err) {
		return nil, adminUser{}, false
	}
	if !session.IsAdmin || session.MustChangePassword {
		writeError(w, http.StatusForbidden, "administrator access required")
		return nil, adminUser{}, false
	}
	u, err := loadAdminUser(r.Context(), tx, chi.URLParam(r, "id"), true)
	if err != nil {
		adminDatabaseError(w, err)
		return nil, adminUser{}, false
	}
	if u.Revision != *revision {
		writeError(w, http.StatusConflict, "Account changed since it was opened. Reload it before saving.")
		return nil, adminUser{}, false
	}
	ok = true
	return tx, u, true
}

func protectAdminAccess(ctx context.Context, tx pgx.Tx, actor string, target adminUser) error {
	if target.ID == actor {
		return errors.New("you cannot suspend, ban or remove your own administrator role")
	}
	if !target.IsAdmin {
		return nil
	}
	var others int
	err := tx.QueryRow(ctx, `SELECT count(*) FROM users u WHERE u.is_admin AND u.id<>$1
		AND (u.account_status='active' OR (u.account_status='suspended' AND u.suspended_until<=now()))
		AND EXISTS (SELECT 1 FROM players p WHERE p.user_id=u.id)`, target.ID).Scan(&others)
	if err != nil {
		return errors.New("unable to verify remaining administrators")
	}
	if others == 0 {
		return errors.New("at least one active administrator must remain")
	}
	return nil
}

func applyAdminPlayerPatch(p *adminPlayer, patch adminPlayerPatch) ([]string, error) {
	fields := []string{}
	set := func(name string, source *int, target *int) {
		if source != nil {
			*target = *source
			fields = append(fields, "player."+name)
		}
	}
	if patch.Credits != nil {
		p.Credits = *patch.Credits
		fields = append(fields, "player.credits")
	}
	if patch.XP != nil {
		p.XP = *patch.XP
		fields = append(fields, "player.xp")
		p.Level = game.LevelForXP(p.XP)
	}
	if patch.ShipType != nil {
		p.ShipType = *patch.ShipType
		fields = append(fields, "player.ship_type")
	}
	set("ship_cargo_upgrades", patch.ShipCargoUpgrades, &p.ShipCargoUpgrades)
	set("ship_turn_upgrades", patch.ShipTurnUpgrades, &p.ShipTurnUpgrades)
	set("turns", patch.Turns, &p.Turns)
	set("turns_max", patch.TurnsMax, &p.TurnsMax)
	set("sector_id", patch.SectorID, &p.SectorID)
	set("cargo_max", patch.CargoMax, &p.CargoMax)
	set("cargo_ore", patch.CargoOre, &p.CargoOre)
	set("cargo_organics", patch.CargoOrganics, &p.CargoOrganics)
	set("cargo_equipment", patch.CargoEquipment, &p.CargoEquipment)
	set("season_id", patch.SeasonID, &p.SeasonID)
	if patch.Turns != nil {
		p.LastTurnRegen = time.Now().UTC()
	}
	if p.Credits < 0 || p.Credits > 9_000_000_000_000_000 || p.XP < 0 || p.XP > 9_000_000_000_000_000 {
		return nil, errors.New("credits and XP must be between 0 and 9000000000000000")
	}
	for name, value := range map[string]int{"turns": p.Turns, "turns_max": p.TurnsMax,
		"cargo_max": p.CargoMax, "cargo_ore": p.CargoOre, "cargo_organics": p.CargoOrganics, "cargo_equipment": p.CargoEquipment} {
		if value < 0 || value > 1_000_000 {
			return nil, fmt.Errorf("%s must be between 0 and 1000000", name)
		}
	}
	if p.Turns > p.TurnsMax {
		return nil, errors.New("turns cannot exceed maximum turns")
	}
	if p.CargoOre+p.CargoOrganics+p.CargoEquipment > p.CargoMax {
		return nil, errors.New("cargo exceeds cargo capacity")
	}
	if p.SectorID < 1 || p.SectorID > 2147483647 || p.SeasonID < 1 || p.SeasonID > 2147483647 {
		return nil, errors.New("sector and season must be valid positive IDs")
	}
	if p.ShipCargoUpgrades < 0 || p.ShipCargoUpgrades > 20 || p.ShipTurnUpgrades < 0 || p.ShipTurnUpgrades > 10 {
		return nil, errors.New("cargo upgrades must be 0-20 and turn upgrades 0-10")
	}
	switch p.ShipType {
	case "SCOUT", "TRADER", "FREIGHTER", "INTERCEPTOR":
	default:
		return nil, errors.New("invalid ship type")
	}
	return fields, nil
}

func (s *Server) handleAdminUserUpdate(w http.ResponseWriter, r *http.Request) {
	var patch adminUserPatch
	if !decodeAdminJSON(w, r, &patch) {
		return
	}
	if patch.Username != nil {
		*patch.Username = strings.TrimSpace(*patch.Username)
		if len(*patch.Username) < 3 || len(*patch.Username) > 20 || strings.ContainsFunc(*patch.Username, unicode.IsControl) {
			writeError(w, http.StatusBadRequest, "username must be 3-20 UTF-8 bytes without control characters")
			return
		}
	}
	newHash := ""
	if patch.NewPassword != nil {
		if err := auth.ValidatePasswordLength(*patch.NewPassword); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		var err error
		newHash, err = auth.HashPassword(*patch.NewPassword)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "unable to reset password")
			return
		}
	}
	tx, before, ok := s.beginAdminMutation(w, r, patch.Revision)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	after := before
	fields := []string{}
	revoke := false
	actor, _ := userIDFrom(r.Context())
	if patch.Username != nil {
		after.Username = *patch.Username
		fields = append(fields, "username")
		revoke = after.Username != before.Username
	}
	if patch.IsAdmin != nil {
		if before.IsAdmin && !*patch.IsAdmin {
			if err := protectAdminAccess(r.Context(), tx, actor, before); err != nil {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
		}
		after.IsAdmin = *patch.IsAdmin
		fields = append(fields, "is_admin")
		revoke = revoke || after.IsAdmin != before.IsAdmin
	}
	if patch.MustChangePassword != nil {
		after.MustChangePassword = *patch.MustChangePassword
		fields = append(fields, "must_change_password")
		revoke = revoke || after.MustChangePassword != before.MustChangePassword
	}
	if newHash != "" {
		fields = append(fields, "password_reset")
		revoke = true
		// Reset passwords require a change by default, unless explicitly overridden.
		if patch.MustChangePassword == nil {
			after.MustChangePassword = true
		}
	}
	if patch.Player != nil {
		if before.Player == nil {
			writeError(w, http.StatusConflict, "account has no player to edit")
			return
		}
		p := *before.Player
		changed, err := applyAdminPlayerPatch(&p, *patch.Player)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		after.Player = &p
		fields = append(fields, changed...)
		if len(changed) > 0 {
			_, err = tx.Exec(r.Context(), `UPDATE players SET credits=$2, xp=$3, level=$4,
				ship_type=$5, ship_cargo_upgrades=$6, ship_turn_upgrades=$7, turns=$8,
				turns_max=$9, sector_id=$10, cargo_max=$11, cargo_ore=$12, cargo_organics=$13,
				cargo_equipment=$14, last_turn_regen=$15, season_id=$16 WHERE id=$1`,
				p.ID, p.Credits, p.XP, p.Level, p.ShipType, p.ShipCargoUpgrades, p.ShipTurnUpgrades,
				p.Turns, p.TurnsMax, p.SectorID, p.CargoMax, p.CargoOre, p.CargoOrganics,
				p.CargoEquipment, p.LastTurnRegen, p.SeasonID)
			if err != nil {
				adminDatabaseError(w, err)
				return
			}
			if patch.Player.SectorID != nil {
				if err := game.MarkDiscovered(r.Context(), tx, p.ID, p.SectorID); err != nil {
					adminDatabaseError(w, err)
					return
				}
			}
		}
	}
	if len(fields) == 0 {
		writeError(w, http.StatusBadRequest, "no editable fields supplied")
		return
	}
	_, err := tx.Exec(r.Context(), `UPDATE users SET username=$2, is_admin=$3, must_change_password=$4,
		password_hash=CASE WHEN $5='' THEN password_hash ELSE $5 END,
		password_changed_at=CASE WHEN $5='' THEN password_changed_at ELSE now() END,
		session_version=session_version+CASE WHEN $6 THEN 1 ELSE 0 END,
		account_revision=account_revision+1 WHERE id=$1`,
		before.ID, after.Username, after.IsAdmin, after.MustChangePassword, newHash, revoke)
	if err != nil {
		adminDatabaseError(w, err)
		return
	}
	s.finishAdminMutation(w, r, tx, before, "USER_UPDATE", fields, "", revoke && actor == before.ID)
}

func (s *Server) handleAdminUserModeration(w http.ResponseWriter, r *http.Request) {
	var request moderationRequest
	if !decodeAdminJSON(w, r, &request) {
		return
	}
	request.Reason = strings.TrimSpace(request.Reason)
	if len(request.Reason) < 1 || len(request.Reason) > 1000 {
		writeError(w, http.StatusBadRequest, "a reason of 1-1000 UTF-8 bytes is required")
		return
	}
	status := ""
	switch request.Action {
	case "suspend":
		status = "suspended"
	case "ban":
		status = "banned"
	case "activate":
		status = "active"
	default:
		writeError(w, http.StatusBadRequest, "action must be suspend, ban or activate")
		return
	}
	if request.SuspendedUntil != nil && (status != "suspended" || !request.SuspendedUntil.After(time.Now().UTC()) || request.SuspendedUntil.Year() > 9999) {
		writeError(w, http.StatusBadRequest, "suspended_until must be a future RFC3339 date and is only valid for suspensions")
		return
	}
	tx, before, ok := s.beginAdminMutation(w, r, request.Revision)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	actor, _ := userIDFrom(r.Context())
	if status != "active" {
		if err := protectAdminAccess(r.Context(), tx, actor, before); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	}
	_, err := tx.Exec(r.Context(), `UPDATE users SET account_status=$2, suspended_until=$3,
		moderation_reason=$4, moderated_at=now(), session_version=session_version+1,
		account_revision=account_revision+1 WHERE id=$1`, before.ID, status, request.SuspendedUntil, request.Reason)
	if err != nil {
		adminDatabaseError(w, err)
		return
	}
	s.finishAdminMutation(w, r, tx, before, "USER_"+strings.ToUpper(request.Action),
		[]string{"account_status", "suspended_until", "moderation_reason"}, request.Reason, false)
}

func (s *Server) finishAdminMutation(w http.ResponseWriter, r *http.Request, tx pgx.Tx, before adminUser, action string, fields []string, reason string, signInAgain bool) {
	after, err := loadAdminUser(r.Context(), tx, before.ID, false)
	if err != nil {
		adminDatabaseError(w, err)
		return
	}
	actor, _ := userIDFrom(r.Context())
	// Only the explicit response model is serialized. Never log the request or hash.
	details := map[string]any{"target_user_id": before.ID, "fields": fields, "reason": reason, "before": before, "after": after}
	_, err = tx.Exec(r.Context(), `INSERT INTO admin_audit_log(actor_user_id, action, details) VALUES ($1,$2,$3)`, actor, action, details)
	if err != nil {
		adminDatabaseError(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		adminDatabaseError(w, err)
		return
	}
	slog.Info("admin_user_change", "actor_user_id", actor, "target_user_id", before.ID, "action", action, "fields", fields)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "user": after, "sign_in_again": signInAgain})
}
