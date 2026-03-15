package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sovereignconquest/internal/config"
)

func TestVersionAndHelpEndpoints(t *testing.T) {
	srv := (&Server{Cfg: config.Load()}).Router()

	t.Run("version", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		if got := rr.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("Cache-Control=%q want no-store", got)
		}
		var payload map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if payload["version"] != config.Version {
			t.Fatalf("version=%v want %s", payload["version"], config.Version)
		}
	})

	t.Run("help", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/help", nil)
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		var payload struct {
			Help []string `json:"help"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if len(payload.Help) == 0 {
			t.Fatal("expected help lines")
		}
	})
}

func TestRouterServesBundledWebUIWhenConfigured(t *testing.T) {
	webRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(webRoot, "index.html"), []byte("<html><body>home</body></html>"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webRoot, "app.js"), []byte("console.log('ok');"), 0o644); err != nil {
		t.Fatalf("write app.js: %v", err)
	}

	srv := (&Server{Cfg: config.Config{WebRoot: webRoot}}).Router()

	t.Run("root falls back to index", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		if got := rr.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("Cache-Control=%q want no-store", got)
		}
		if !strings.Contains(rr.Body.String(), "home") {
			t.Fatalf("body=%q missing home", rr.Body.String())
		}
	})

	t.Run("asset serves directly", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		if rr.Header().Get("Cache-Control") == "no-store" {
			t.Fatal("static asset should not be forced to no-store")
		}
		if !strings.Contains(rr.Body.String(), "console.log") {
			t.Fatalf("body=%q missing js payload", rr.Body.String())
		}
	})

	t.Run("unknown api path stays api 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/does-not-exist", nil)
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), "not found") {
			t.Fatalf("body=%q missing not found", rr.Body.String())
		}
	})

	t.Run("missing asset with extension returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/missing.js", nil)
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})
}
