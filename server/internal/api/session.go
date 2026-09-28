package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"sovereignconquest/internal/auth"
)

func writeAccessError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"ok": false, "code": code, "error": message})
}

func checkAccountAccess(w http.ResponseWriter, session auth.Session) bool {
	switch session.EffectiveStatus(time.Now().UTC()) {
	case "active":
		return true
	case "suspended":
		message := "Your account is suspended. Contact an administrator."
		if session.SuspendedUntil != nil {
			message = "Your account is suspended until " + session.SuspendedUntil.UTC().Format(time.RFC3339) + "."
		}
		writeAccessError(w, http.StatusForbidden, "account_suspended", message)
	case "banned":
		writeAccessError(w, http.StatusForbidden, "account_banned", "Your account is banned. Contact an administrator.")
	default:
		writeAccessError(w, http.StatusForbidden, "account_unavailable", "Account unavailable. Contact an administrator.")
	}
	return false
}

func checkSession(w http.ResponseWriter, session auth.Session, claims *auth.Claims, err error) bool {
	if errors.Is(err, pgx.ErrNoRows) {
		writeAccessError(w, http.StatusUnauthorized, "session_revoked", "Account unavailable. Please sign in again.")
		return false
	}
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return false
	}
	if !checkAccountAccess(w, session) {
		return false
	}
	if claims == nil || session.Version != claims.SessionVersion {
		writeAccessError(w, http.StatusUnauthorized, "session_revoked", "Your session has ended. Please sign in again.")
		return false
	}
	return true
}

func (s *Server) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := r.Context().Value(ctxSession).(auth.Session)
		if !ok || !session.IsAdmin || session.MustChangePassword {
			writeError(w, http.StatusForbidden, "administrator access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
