package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
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
		// Health/version checks should never be cached (used by the UI for live version badging).
		w.Header().Set("Cache-Control", "no-store")
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
		// Direct messages / bug reporting
		protected.Get("/api/messages/inbox", s.handleInboxMessages)
		protected.Get("/api/messages/sent", s.handleSentMessages)
		protected.Get("/api/messages/unread_count", s.handleUnreadMessageCount)
		protected.Post("/api/messages/mark_read", s.handleMarkMessagesRead)
		protected.Post("/api/messages/delete", s.handleDeleteMessage)
		protected.Post("/api/messages/send", s.handleSendMessage)
		protected.Post("/api/messages/report", s.handleReportMessage)
		protected.Get("/api/messages/attachments/{id}", s.handleDownloadMessageAttachment)
		protected.Get("/api/admin/ansi_map", s.handleAdminAnsiMap)
		protected.Post("/api/bug_report", s.handleBugReport)
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

// ---- Direct messaging / bug reporting ----

const (
	maxDirectMessageSubjectLen = 120
	maxDirectMessageBodyLen    = 4000
	maxBugReportBodyLen        = 8000

	maxAttachmentCount      = 5
	maxAttachmentBytes      = 5 << 20  // 5 MiB each
	maxTotalUploadBytes     = 25 << 20 // 25 MiB total request cap
	maxMultipartMemory      = 8 << 20  // 8 MiB in-memory before spilling to disk
	maxAttachmentNameLen    = 200
	defaultMessageListLimit = 20
	defaultMessageListMax   = 50
)

type sendMessageJSONRequest struct {
	ToUsername       string `json:"to_username"`
	Subject          string `json:"subject"`
	Body             string `json:"body"`
	RelatedMessageID *int64 `json:"related_message_id,omitempty"`
}

type reportMessageRequest struct {
	MessageID int64 `json:"message_id"`
}

type attachmentInput struct {
	Filename    string
	ContentType string
	Data        []byte
}

func (s *Server) handleInboxMessages(w http.ResponseWriter, r *http.Request) {
	pid, ok := playerIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing player context")
		return
	}

	limit := parseLimit(r, defaultMessageListLimit, defaultMessageListMax)
	msgs, err := game.LoadInboxDirectMessages(r.Context(), s.Pool, pid, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"messages": msgs,
	})
}

func (s *Server) handleSentMessages(w http.ResponseWriter, r *http.Request) {
	pid, ok := playerIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing player context")
		return
	}

	limit := parseLimit(r, defaultMessageListLimit, defaultMessageListMax)
	msgs, err := game.LoadSentDirectMessages(r.Context(), s.Pool, pid, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"messages": msgs,
	})
}

type markReadMessagesRequest struct {
	MessageIDs []int64 `json:"message_ids"`
}

type deleteMessageRequest struct {
	MessageID int64 `json:"message_id"`
}

func (s *Server) handleUnreadMessageCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pid := mustPlayerID(ctx)

	c, err := game.CountUnreadDirectMessages(ctx, s.Pool, pid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "unread": c})
}

func (s *Server) handleMarkMessagesRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pid := mustPlayerID(ctx)

	var req markReadMessagesRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(req.MessageIDs) == 0 {
		writeError(w, http.StatusBadRequest, "message_ids required")
		return
	}

	updated, err := game.MarkDirectMessagesRead(ctx, s.Pool, pid, req.MessageIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "updated": updated})
}

func (s *Server) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pid := mustPlayerID(ctx)

	var req deleteMessageRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.MessageID <= 0 {
		writeError(w, http.StatusBadRequest, "message_id required")
		return
	}

	if err := game.DeleteDirectMessage(ctx, s.Pool, pid, req.MessageID); err != nil {
		if errors.Is(err, game.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	pid, ok := playerIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing player context")
		return
	}

	toUsername, subject, body, relatedMessageID, atts, parseErr := parseMessagePayload(w, r, maxDirectMessageBodyLen)
	if parseErr != nil {
		writeError(w, http.StatusBadRequest, parseErr.Error())
		return
	}

	// Validate
	toUsername = strings.TrimSpace(toUsername)
	if len(toUsername) < 3 || len(toUsername) > 20 {
		writeError(w, http.StatusBadRequest, "recipient username must be 3-20 chars")
		return
	}
	if len(subject) > maxDirectMessageSubjectLen {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("subject too long (max %d)", maxDirectMessageSubjectLen))
		return
	}
	body = strings.TrimSpace(body)
	if body == "" {
		writeError(w, http.StatusBadRequest, "message body cannot be empty")
		return
	}
	if len(body) > maxDirectMessageBodyLen {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("message too long (max %d)", maxDirectMessageBodyLen))
		return
	}

	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	toPID, err := game.LookupPlayerIDByUsername(ctx, tx, toUsername)
	if errors.Is(err, game.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "unknown recipient username")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if toPID == pid {
		writeError(w, http.StatusBadRequest, "cannot send a message to yourself")
		return
	}

	if relatedMessageID != nil {
		// Related message must be visible to the sender (sender or recipient of the original).
		var okRel bool
		err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM direct_messages WHERE id=$1 AND (from_player_id=$2 OR to_player_id=$2))`, *relatedMessageID, pid).Scan(&okRel)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
		if !okRel {
			writeError(w, http.StatusBadRequest, "invalid related_message_id")
			return
		}
	}

	msgID, err := game.InsertDirectMessage(ctx, tx, pid, toPID, game.MessageKindUser, strings.TrimSpace(subject), body, relatedMessageID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	for _, a := range atts {
		if err := game.InsertDirectMessageAttachment(ctx, tx, msgID, a.Filename, a.ContentType, a.Data); err != nil {
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "Message sent.",
		"id":      msgID,
	})
}

func (s *Server) handleReportMessage(w http.ResponseWriter, r *http.Request) {
	pid, ok := playerIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing player context")
		return
	}

	var req reportMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.MessageID < 1 {
		writeError(w, http.StatusBadRequest, "invalid message_id")
		return
	}

	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	reporterUsername, err := game.LookupUsernameByPlayerID(ctx, tx, pid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	reported, atts, err := game.LoadDirectMessageForReport(ctx, tx, pid, req.MessageID)
	if errors.Is(err, game.ErrNotFound) {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if reported.Kind != game.MessageKindUser {
		writeError(w, http.StatusBadRequest, "only direct user messages can be reported")
		return
	}

	adminPID, err := game.LookupAdminPlayerID(ctx, tx, s.Cfg.InitialAdminUser)
	if errors.Is(err, game.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "admin account not available")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	subject := fmt.Sprintf("Spam/Abuse Report: message #%d", reported.ID)
	body := game.FormatSpamReportBody(reporterUsername, reported)
	related := reported.ID

	reportMsgID, err := game.InsertDirectMessage(ctx, tx, pid, adminPID, game.MessageKindSpamReport, subject, body, &related)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	for _, a := range atts {
		if err := game.InsertDirectMessageAttachment(ctx, tx, reportMsgID, a.Filename, a.ContentType, a.Data); err != nil {
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "Reported to admin.",
	})
}

func (s *Server) handleBugReport(w http.ResponseWriter, r *http.Request) {
	pid, ok := playerIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing player context")
		return
	}

	// Only multipart is supported here (attachments).
	r.Body = http.MaxBytesReader(w, r.Body, maxTotalUploadBytes)
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("upload too large (max %d bytes)", maxTotalUploadBytes))
			return
		}
		writeError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	subject := strings.TrimSpace(r.FormValue("subject"))
	if subject == "" {
		subject = "Bug report"
	}
	if len(subject) > maxDirectMessageSubjectLen {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("title too long (max %d)", maxDirectMessageSubjectLen))
		return
	}

	body := strings.TrimSpace(r.FormValue("body"))
	if body == "" {
		writeError(w, http.StatusBadRequest, "description cannot be empty")
		return
	}
	if len(body) > maxBugReportBodyLen {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("description too long (max %d)", maxBugReportBodyLen))
		return
	}

	files := []*multipart.FileHeader{}
	if r.MultipartForm != nil {
		files = r.MultipartForm.File["attachments"]
	}
	attachments, err := readAttachments(files)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	reporterUsername, err := game.LookupUsernameByPlayerID(ctx, tx, pid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	adminPID, err := game.LookupAdminPlayerID(ctx, tx, s.Cfg.InitialAdminUser)
	if errors.Is(err, game.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "admin account not available")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	fullSubject := fmt.Sprintf("Bug Report: %s", subject)
	fullBody := fmt.Sprintf("Bug report submitted by %s.\n\n%s", reporterUsername, body)

	msgID, err := game.InsertDirectMessage(ctx, tx, pid, adminPID, game.MessageKindBugReport, fullSubject, fullBody, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	for _, a := range attachments {
		if err := game.InsertDirectMessageAttachment(ctx, tx, msgID, a.Filename, a.ContentType, a.Data); err != nil {
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "Bug report sent to admin.",
		"id":      msgID,
	})
}

func (s *Server) handleDownloadMessageAttachment(w http.ResponseWriter, r *http.Request) {
	pid, ok := playerIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing player context")
		return
	}
	uid, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user context")
		return
	}

	idStr := chi.URLParam(r, "id")
	attID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || attID < 1 {
		writeError(w, http.StatusBadRequest, "invalid attachment id")
		return
	}

	// Determine admin status from users table.
	isAdmin := false
	var adminOK bool
	if err := s.Pool.QueryRow(r.Context(), "SELECT is_admin FROM users WHERE id=$1", uid).Scan(&adminOK); err == nil {
		isAdmin = adminOK
	}

	var filename, contentType string
	var data []byte
	if isAdmin {
		filename, contentType, data, err = game.LoadDirectMessageAttachmentForAdmin(r.Context(), s.Pool, attID)
	} else {
		filename, contentType, data, err = game.LoadDirectMessageAttachmentForPlayer(r.Context(), s.Pool, attID, pid)
	}
	if errors.Is(err, game.ErrNotFound) {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	// Avoid caching attachments.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", sanitizeFilename(filename)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func parseLimit(r *http.Request, def, max int) int {
	q := strings.TrimSpace(r.URL.Query().Get("limit"))
	if q == "" {
		return def
	}
	n, err := strconv.Atoi(q)
	if err != nil {
		return def
	}
	if n < 1 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

func parseMessagePayload(w http.ResponseWriter, r *http.Request, maxLen int) (toUsername, subject, body string, relatedMessageID *int64, atts []attachmentInput, err error) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		mr, e := r.MultipartReader()
		if e != nil {
			return "", "", "", nil, nil, fmt.Errorf("invalid multipart: %w", e)
		}
		fields := map[string]string{}
		var attachments []attachmentInput
		for {
			part, e := mr.NextPart()
			if e == io.EOF {
				break
			}
			if e != nil {
				return "", "", "", nil, nil, fmt.Errorf("multipart read error: %w", e)
			}
			name := part.FormName()
			if name == "attachment" {
				fn := part.FileName()
				ct := part.Header.Get("Content-Type")
				data, e := io.ReadAll(io.LimitReader(part, int64(maxAttachmentBytes)+1))
				if e != nil {
					return "", "", "", nil, nil, fmt.Errorf("attachment read error: %w", e)
				}
				if len(data) > maxAttachmentBytes {
					return "", "", "", nil, nil, fmt.Errorf("attachment too large (max %d bytes)", maxAttachmentBytes)
				}
				attachments = append(attachments, attachmentInput{Filename: fn, ContentType: ct, Data: data})
				continue
			}

			val, e := io.ReadAll(io.LimitReader(part, int64(maxLen)+1))
			if e != nil {
				return "", "", "", nil, nil, fmt.Errorf("read error: %w", e)
			}
			if len(val) > maxLen {
				return "", "", "", nil, nil, fmt.Errorf("field too long")
			}
			fields[name] = string(val)
		}

		var rel *int64
		if raw := strings.TrimSpace(fields["related_message_id"]); raw != "" {
			id, e := strconv.ParseInt(raw, 10, 64)
			if e != nil {
				return "", "", "", nil, nil, fmt.Errorf("invalid related_message_id")
			}
			rel = &id
		}

		return fields["to_username"], fields["subject"], fields["body"], rel, attachments, nil
	}

	// JSON
	var payload sendMessageJSONRequest
	dec := json.NewDecoder(io.LimitReader(r.Body, int64(maxLen)+1024))
	if e := dec.Decode(&payload); e != nil {
		return "", "", "", nil, nil, fmt.Errorf("invalid json")
	}
	return payload.ToUsername, payload.Subject, payload.Body, payload.RelatedMessageID, nil, nil
}

func readAttachments(files []*multipart.FileHeader) ([]attachmentInput, error) {
	if len(files) == 0 {
		return nil, nil
	}
	if len(files) > maxAttachmentCount {
		return nil, fmt.Errorf("too many attachments (max %d)", maxAttachmentCount)
	}

	total := int64(0)
	out := make([]attachmentInput, 0, len(files))
	for _, fh := range files {
		if fh == nil {
			continue
		}
		if fh.Size > maxAttachmentBytes {
			return nil, fmt.Errorf("attachment too large: %s (max %d bytes)", sanitizeFilename(fh.Filename), maxAttachmentBytes)
		}
		total += fh.Size
		if total > maxTotalUploadBytes {
			return nil, fmt.Errorf("attachments exceed total upload limit (%d bytes)", maxTotalUploadBytes)
		}

		f, err := fh.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to read attachment")
		}
		bs, rerr := io.ReadAll(io.LimitReader(f, maxAttachmentBytes+1))
		_ = f.Close()
		if rerr != nil {
			return nil, fmt.Errorf("failed to read attachment")
		}
		if int64(len(bs)) > maxAttachmentBytes {
			return nil, fmt.Errorf("attachment too large: %s (max %d bytes)", sanitizeFilename(fh.Filename), maxAttachmentBytes)
		}

		ct := fh.Header.Get("Content-Type")
		if ct == "" {
			ct = http.DetectContentType(bs)
		}

		name := sanitizeFilename(fh.Filename)
		out = append(out, attachmentInput{Filename: name, ContentType: ct, Data: bs})
	}
	return out, nil
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "attachment"
	}
	name = filepath.Base(name)
	if len(name) > maxAttachmentNameLen {
		name = name[:maxAttachmentNameLen]
	}
	return name
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
