package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"sovereignconquest/internal/auth"
	"sovereignconquest/internal/config"
)

type administratorGuard struct {
	cfg  config.Config
	pool *pgxpool.Pool
	next http.Handler
}

// RequireAdministrator protects the destructive administration endpoint with
// both a signed user session and the separately configured administrative key.
func RequireAdministrator(cfg config.Config, pool *pgxpool.Pool, next http.Handler) http.Handler {
	return &administratorGuard{cfg: cfg, pool: pool, next: next}
}

func (g *administratorGuard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/admin/soft_wipe" {
		g.next.ServeHTTP(w, r)
		return
	}
	if g.cfg.AdminSecret == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	provided := r.Header.Get("X-Admin-Secret")
	if len(provided) != len(g.cfg.AdminSecret) || subtle.ConstantTimeCompare([]byte(provided), []byte(g.cfg.AdminSecret)) != 1 {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	header := r.Header.Get("Authorization")
	if len(header) > 8192 || !strings.HasPrefix(header, "Bearer ") {
		writeError(w, http.StatusUnauthorized, "administrator authentication required")
		return
	}
	claims, err := auth.ParseToken(g.cfg.JWTSecret, strings.TrimPrefix(header, "Bearer "))
	if err != nil || g.pool == nil {
		writeError(w, http.StatusUnauthorized, "administrator authentication required")
		return
	}

	var isAdmin bool
	if err := g.pool.QueryRow(r.Context(), "SELECT is_admin FROM users WHERE id=$1", claims.UserID).Scan(&isAdmin); err != nil {
		writeError(w, http.StatusUnauthorized, "administrator authentication required")
		return
	}
	if !isAdmin {
		writeError(w, http.StatusForbidden, "administrator access required")
		return
	}

	g.next.ServeHTTP(w, r)
}
