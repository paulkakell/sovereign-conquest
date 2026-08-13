package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"sovereignconquest/internal/config"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *responseRecorder) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseRecorder) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(data)
	w.bytes += n
	return n, err
}

func (w *responseRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

type rateEntry struct {
	WindowStart time.Time
	LastSeen    time.Time
	Count       int
}

type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateEntry
	now     func() time.Time
	calls   uint64
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		entries: make(map[string]rateEntry),
		now:     time.Now,
	}
}

func (l *RateLimiter) Allow(key string, limit int, window time.Duration) (bool, time.Duration) {
	if limit < 1 || window <= 0 {
		return true, 0
	}
	if l == nil {
		return true, 0
	}

	now := l.now().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()

	l.calls++
	if l.calls%1024 == 0 {
		cutoff := now.Add(-10 * window)
		for entryKey, entry := range l.entries {
			if entry.LastSeen.Before(cutoff) {
				delete(l.entries, entryKey)
			}
		}
	}

	entry, found := l.entries[key]
	if !found || now.Sub(entry.WindowStart) >= window {
		l.entries[key] = rateEntry{WindowStart: now, LastSeen: now, Count: 1}
		return true, 0
	}

	entry.LastSeen = now
	if entry.Count >= limit {
		l.entries[key] = entry
		retryAfter := window - now.Sub(entry.WindowStart)
		if retryAfter < time.Second {
			retryAfter = time.Second
		}
		return false, retryAfter
	}

	entry.Count++
	l.entries[key] = entry
	return true, 0
}

func (s *Server) clientIPMiddleware() func(http.Handler) http.Handler {
	if strings.EqualFold(s.Cfg.ClientIPMode, "header") {
		return middleware.ClientIPFromHeader(s.Cfg.ClientIPHeader)
	}
	return middleware.ClientIPFromRemoteAddr
}

func (s *Server) securityMiddleware(next http.Handler) http.Handler {
	if s.Limiter == nil {
		s.Limiter = NewRateLimiter()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &responseRecorder{ResponseWriter: w}
		setSecurityHeaders(recorder, r)

		clientIP := middleware.GetClientIP(r.Context())
		if clientIP == "" {
			clientIP = remoteHost(r.RemoteAddr)
		}
		defer func() {
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}
			slog.Info("http request",
				"request_id", middleware.GetReqID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"bytes", recorder.bytes,
				"duration_ms", time.Since(started).Milliseconds(),
				"client_ip", clientIP,
			)
		}()

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			limit := s.Cfg.RequestBodyLimitBytes
			if r.URL.Path == "/api/bug_report" || r.URL.Path == "/api/messages/send" {
				limit = maxTotalUploadBytes + (1 << 20)
			}
			if limit > 0 {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
		}

		limit, window, class := requestRatePolicy(r)
		if limit > 0 {
			key := clientIP + "|" + class
			allowed, retryAfter := s.Limiter.Allow(key, limit, window)
			if !allowed {
				seconds := int(retryAfter.Round(time.Second) / time.Second)
				if seconds < 1 {
					seconds = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(seconds))
				writeError(recorder, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
		}

		next.ServeHTTP(recorder, r)
	})
}

func requestRatePolicy(r *http.Request) (int, time.Duration, string) {
	path := r.URL.Path
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		if strings.HasPrefix(path, "/api/messages/attachments/") {
			return 60, time.Minute, "attachment"
		}
		return 0, 0, ""
	}

	switch path {
	case "/api/register", "/api/login":
		return 10, time.Minute, "authentication"
	case "/api/change_password":
		return 10, time.Minute, "password"
	case "/api/command":
		return 120, time.Minute, "command"
	case "/api/messages/send", "/api/messages/report", "/api/bug_report":
		return 30, time.Minute, "messaging"
	case "/api/admin/soft_wipe":
		return 5, 10 * time.Minute, "administration"
	default:
		return 60, time.Minute, "mutation"
	}
}

func setSecurityHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'; img-src 'self' data:; object-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'")
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Cache-Control", "no-store")
	}
}

func remoteHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil && host != "" {
		return host
	}
	return remoteAddr
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxErr):
			return fmt.Errorf("request body exceeds %d bytes", maxErr.Limit)
		case errors.Is(err, io.EOF):
			return errors.New("request body is required")
		default:
			return fmt.Errorf("invalid json: %w", err)
		}
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func normalizedAttachmentContentType(value string, data []byte) string {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	if err == nil && mediaType != "" && !strings.ContainsAny(mediaType, "\r\n") {
		return mediaType
	}
	if len(data) > 0 {
		return http.DetectContentType(data)
	}
	return "application/octet-stream"
}

func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"name":    config.AppName,
		"version": config.Version,
	})
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if s.Pool == nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.Pool.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"name":    config.AppName,
		"version": config.Version,
	})
}
