package api

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sovereignconquest/internal/config"
)

const (
	jsonRequestLimit = int64(1 << 20)
	uploadLimit      = int64(26 << 20)
)

type transportWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *transportWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *transportWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(data)
	w.bytes += n
	return n, err
}

func (w *transportWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

type requestWindow struct {
	started  time.Time
	lastSeen time.Time
	count    int
}

type requestLimiter struct {
	mu      sync.Mutex
	entries map[string]requestWindow
	checks  uint64
}

func newRequestLimiter() *requestLimiter {
	return &requestLimiter{entries: make(map[string]requestWindow)}
}

func (l *requestLimiter) allow(key string, limit int, duration time.Duration) (bool, time.Duration) {
	if limit < 1 || duration <= 0 {
		return true, 0
	}
	now := time.Now().UTC()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.checks++
	if l.checks%1024 == 0 {
		cutoff := now.Add(-10 * duration)
		for entryKey, entry := range l.entries {
			if entry.lastSeen.Before(cutoff) {
				delete(l.entries, entryKey)
			}
		}
	}

	entry, found := l.entries[key]
	if !found || now.Sub(entry.started) >= duration {
		l.entries[key] = requestWindow{started: now, lastSeen: now, count: 1}
		return true, 0
	}
	entry.lastSeen = now
	if entry.count >= limit {
		l.entries[key] = entry
		remaining := duration - now.Sub(entry.started)
		if remaining < time.Second {
			remaining = time.Second
		}
		return false, remaining
	}
	entry.count++
	l.entries[key] = entry
	return true, 0
}

type transportHandler struct {
	pool    *pgxpool.Pool
	next    http.Handler
	limiter *requestLimiter
}

// HardenHTTP adds limits, security headers, proxy sanitization, health probes,
// and structured request logs outside the existing API router.
func HardenHTTP(pool *pgxpool.Pool, next http.Handler) http.Handler {
	return &transportHandler{pool: pool, next: next, limiter: newRequestLimiter()}
}

func (h *transportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	recorder := &transportWriter{ResponseWriter: w}
	setTransportHeaders(recorder, r)
	clientIP := requestClientIP(r)
	sanitizeProxyHeaders(r, clientIP)

	defer func() {
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		slog.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"bytes", recorder.bytes,
			"duration_ms", time.Since(started).Milliseconds(),
			"client_ip", clientIP,
		)
	}()

	if h.serveProbe(recorder, r) {
		return
	}

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		limit := jsonRequestLimit
		if r.URL.Path == "/api/bug_report" || r.URL.Path == "/api/messages/send" {
			limit = uploadLimit
		}
		r.Body = http.MaxBytesReader(recorder, r.Body, limit)
	}

	limit, duration, class := requestPolicy(r)
	if limit > 0 {
		allowed, retryAfter := h.limiter.allow(clientIP+"|"+class, limit, duration)
		if !allowed {
			seconds := int(retryAfter.Round(time.Second) / time.Second)
			if seconds < 1 {
				seconds = 1
			}
			recorder.Header().Set("Retry-After", strconv.Itoa(seconds))
			writeError(recorder, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
	}

	h.next.ServeHTTP(recorder, r)
}

func (h *transportHandler) serveProbe(w http.ResponseWriter, r *http.Request) bool {
	switch r.URL.Path {
	case "/api/livez":
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"name":    config.AppName,
			"version": config.Version,
		})
		return true
	case "/api/readyz":
		if h.pool == nil {
			writeError(w, http.StatusServiceUnavailable, "database unavailable")
			return true
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := h.pool.Ping(ctx); err != nil {
			writeError(w, http.StatusServiceUnavailable, "database unavailable")
			return true
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"name":    config.AppName,
			"version": config.Version,
		})
		return true
	default:
		return false
	}
}

func requestPolicy(r *http.Request) (int, time.Duration, string) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		if strings.HasPrefix(r.URL.Path, "/api/messages/attachments/") {
			return 120, time.Minute, "attachment"
		}
		return 0, 0, ""
	}

	switch r.URL.Path {
	case "/api/register", "/api/login":
		return 10, 5 * time.Minute, "authentication"
	case "/api/change_password":
		return 10, 10 * time.Minute, "password"
	case "/api/command":
		return 120, time.Minute, "command"
	case "/api/messages/send", "/api/messages/report", "/api/bug_report":
		return 30, time.Minute, "messaging"
	case "/api/admin/soft_wipe":
		return 3, time.Hour, "administration"
	default:
		return 120, time.Minute, "mutation"
	}
}

func proxyHeadersTrusted() bool {
	trusted, err := strconv.ParseBool(strings.TrimSpace(os.Getenv("TRUST_PROXY_HEADERS")))
	return err == nil && trusted
}

func requestClientIP(r *http.Request) string {
	remote := remoteIP(r.RemoteAddr)
	if !proxyHeadersTrusted() {
		return remote
	}
	for _, name := range []string{"CF-Connecting-IP", "X-Real-IP"} {
		candidate := strings.TrimSpace(r.Header.Get(name))
		if net.ParseIP(candidate) != nil {
			return candidate
		}
	}
	return remote
}

func sanitizeProxyHeaders(r *http.Request, clientIP string) {
	for _, name := range []string{"Forwarded", "True-Client-IP", "X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"} {
		r.Header.Del(name)
	}
	if proxyHeadersTrusted() && net.ParseIP(clientIP) != nil {
		r.Header.Set("X-Real-IP", clientIP)
	}
}

func remoteIP(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err == nil && host != "" {
		return host
	}
	return address
}

func setTransportHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'; img-src 'self' data:; object-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'")
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Cache-Control", "no-store")
	}
	if r.TLS != nil {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}
}
