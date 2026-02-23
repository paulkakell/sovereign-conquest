package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"sovereignconquest/internal/auth"
	"sovereignconquest/internal/config"
	"sovereignconquest/internal/game"
	"sovereignconquest/internal/util"
)

type Server struct {
	Cfg  config.Config
	Pool *pgxpool.Pool
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	r.Get("/api/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"name":    config.AppName,
			"version": config.Version,
		})
	})

	r.Post("/api/register", s.handleRegister)
	r.Post("/api/login", s.handleLogin)

	if s.Cfg.AdminSecret != "" {
		r.Post("/api/admin/soft_wipe", s.handleAdminSoftWipe)
	}

	r.Group(func(protected chi.Router) {
		protected.Use(s.authMiddleware)
		protected.Get("/api/state", s.handleState)
		protected.Post("/api/command", s.handleCommand)
		protected.Post("/api/change_password", s.handleChangePassword)
	})

	return r
}

type ctxKey string

const ctxPlayerID ctxKey = "player_id"
const ctxUserID ctxKey = "user_id"

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		token := strings.TrimPrefix(h, "Bearer ")
		claims, err := auth.ParseToken(s.Cfg.JWTSecret, token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), ctxPlayerID, claims.PlayerID)
		ctx = context.WithValue(ctx, ctxUserID, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func playerIDFrom(ctx context.Context) (string, bool) {
	v := ctx.Value(ctxPlayerID)
	id, ok := v.(string)
	return id, ok
}

func userIDFrom(ctx context.Context) (string, bool) {
	v := ctx.Value(ctxUserID)
	id, ok := v.(string)
	return id, ok
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Token  string           `json:"token"`
	State  game.PlayerState `json:"state"`
	Sector game.SectorView  `json:"sector"`
	Logs   []game.LogEntry  `json:"logs,omitempty"`
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 3 || len(req.Username) > 20 {
		writeError(w, http.StatusBadRequest, "username must be 3-20 chars")
		return
	}
	if len(req.Password) < 8 || len(req.Password) > 100 {
		writeError(w, http.StatusBadRequest, "password must be 8-100 chars")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	now := time.Now().UTC()

	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	// Pick a valid starting sector (do not assume sector 1 exists).
	var startSector int
	if err := tx.QueryRow(r.Context(), "SELECT id FROM sectors ORDER BY id LIMIT 1").Scan(&startSector); err != nil {
		writeError(w, http.StatusInternalServerError, "universe not initialized")
		return
	}

	// Ensure there is an active season.
	var seasonID int
	err = tx.QueryRow(r.Context(), "SELECT id FROM seasons WHERE active=true ORDER BY id DESC LIMIT 1").Scan(&seasonID)
	if errors.Is(err, pgx.ErrNoRows) {
		var next int
		_ = tx.QueryRow(r.Context(), "SELECT COALESCE(MAX(id),0)+1 FROM seasons").Scan(&next)
		name := fmt.Sprintf("Season %d", next)
		err = tx.QueryRow(r.Context(), "INSERT INTO seasons(name, active, started_at) VALUES ($1,true,now()) RETURNING id", name).Scan(&seasonID)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	var userID, playerID string
	userID, err = util.NewID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}
	playerID, err = util.NewID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	err = tx.QueryRow(r.Context(), `
		INSERT INTO users(id, username, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id
	`, userID, req.Username, hash).Scan(&userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusBadRequest, "username unavailable")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	// Player starts in a valid seed sector with basic stats.
	err = tx.QueryRow(r.Context(), `
		INSERT INTO players(id, user_id, credits, turns, turns_max, sector_id, cargo_max, last_turn_regen, season_id)
		VALUES (
			$1, $2,
			$3, $4, $5,
			$6, $7,
			$8,
			$9
		)
		RETURNING id
	`, playerID, userID, int64(1000), 100, 100, startSector, 30, now, seasonID).Scan(&playerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	_ = game.MarkDiscovered(r.Context(), tx, playerID, startSector)
	_ = game.InsertLog(r.Context(), tx, playerID, "SYSTEM", "Welcome to Sovereign Conquest. Start with SCAN, then MOVE and TRADE.")

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	token, err := auth.MintToken(s.Cfg.JWTSecret, userID, playerID, 7*24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mint token")
		return
	}

	state, sector, logs, err := s.loadState(r.Context(), playerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{Token: token, State: state, Sector: sector, Logs: logs})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Username = strings.TrimSpace(req.Username)

	var userID, playerID, hash string
	err := s.Pool.QueryRow(r.Context(), `
		SELECT u.id, p.id, u.password_hash
		FROM users u
		JOIN players p ON p.user_id = u.id
		WHERE u.username = $1
	`, req.Username).Scan(&userID, &playerID, &hash)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !auth.CheckPassword(hash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := auth.MintToken(s.Cfg.JWTSecret, userID, playerID, 7*24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mint token")
		return
	}

	state, sector, logs, err := s.loadState(r.Context(), playerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{Token: token, State: state, Sector: sector, Logs: logs})
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFrom(r.Context())
	if !ok || uid == "" {
		writeError(w, http.StatusUnauthorized, "missing user context")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(req.NewPassword) < 8 || len(req.NewPassword) > 100 {
		writeError(w, http.StatusBadRequest, "password must be 8-100 chars")
		return
	}

	var hash string
	if err := s.Pool.QueryRow(r.Context(), "SELECT password_hash FROM users WHERE id=$1", uid).Scan(&hash); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if !auth.CheckPassword(hash, req.OldPassword) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	_, err = s.Pool.Exec(r.Context(), `
		UPDATE users
		SET password_hash=$2, must_change_password=false, password_changed_at=now()
		WHERE id=$1
	`, uid, newHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	pid, ok := playerIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing player context")
		return
	}

	state, sector, logs, err := s.loadState(r.Context(), pid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"state":  state,
		"sector": sector,
		"logs":   logs,
	})
}

func (s *Server) loadState(ctx context.Context, playerID string) (game.PlayerState, game.SectorView, []game.LogEntry, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return game.PlayerState{}, game.SectorView{}, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	p, err := game.LoadPlayerForUpdate(ctx, tx, playerID)
	if err != nil {
		return game.PlayerState{}, game.SectorView{}, nil, err
	}

	// Turns regenerate on demand, but the last regen timestamp must be persisted to avoid double counting.
	game.RegenTurns(&p, s.Cfg.TurnRegenSeconds, time.Now().UTC())
	if err := game.SavePlayer(ctx, tx, p); err != nil {
		return game.PlayerState{}, game.SectorView{}, nil, err
	}

	sector, err := game.LoadSectorView(ctx, tx, p.SectorID)
	if err != nil {
		return game.PlayerState{}, game.SectorView{}, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return game.PlayerState{}, game.SectorView{}, nil, err
	}

	logs, _ := game.LoadRecentLogs(ctx, s.Pool, playerID, 20)
	return p.ToState(), sector, logs, nil
}

func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	pid, ok := playerIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing player context")
		return
	}

	var cmd game.CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	resp, err := game.ExecuteCommand(r.Context(), s.Pool, pid, cmd, s.Cfg.TurnRegenSeconds)
	if err != nil {
		var ce game.CommandError
		if errors.As(err, &ce) {
			writeJSON(w, http.StatusBadRequest, resp)
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusBadRequest, "invalid command")
			return
		}
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

type softWipeRequest struct {
	SeasonName string `json:"season_name"`
	ResetCorps bool   `json:"reset_corps"`
}

func (s *Server) handleAdminSoftWipe(w http.ResponseWriter, r *http.Request) {
	if s.Cfg.AdminSecret == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	secret := r.Header.Get("X-Admin-Secret")
	if secret == "" || secret != s.Cfg.AdminSecret {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req softWipeRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	ctx := r.Context()
	res, err := game.SoftWipe(ctx, s.Pool, game.SoftWipeRequest{
		SeasonName: strings.TrimSpace(req.SeasonName),
		ResetCorps: req.ResetCorps,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"result": res,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"ok": false, "error": msg})
}
