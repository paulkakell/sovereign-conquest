package api

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTransportProbes(t *testing.T) {
	handler := HardenHTTP(nil, http.NotFoundHandler())
	for _, test := range []struct {
		path   string
		status int
	}{
		{path: "/api/livez", status: http.StatusOK},
		{path: "/api/readyz", status: http.StatusServiceUnavailable},
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
		if recorder.Code != test.status {
			t.Fatalf("%s status=%d want=%d", test.path, recorder.Code, test.status)
		}
	}
}

func TestTransportLogsDoNotExposeTrustedClientAddress(t *testing.T) {
	t.Setenv("TRUST_PROXY_HEADERS", "true")

	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	handler := HardenHTTP(nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("CF-Connecting-IP", "203.0.113.77")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	logged := output.String()
	if strings.Contains(logged, "203.0.113.77") {
		t.Fatal("request log exposed a trusted client address")
	}
	if !strings.Contains(logged, "http_request") {
		t.Fatal("request log entry was not written")
	}
}

func TestRequestLimiter(t *testing.T) {
	limiter := newRequestLimiter()
	allowed, _ := limiter.allow("test", 1, time.Minute)
	if !allowed {
		t.Fatal("first request should be allowed")
	}
	allowed, retry := limiter.allow("test", 1, time.Minute)
	if allowed || retry <= 0 {
		t.Fatal("second request should be delayed")
	}
}

func BenchmarkRequestLimiterAllow(b *testing.B) {
	limiter := newRequestLimiter()
	for i := 0; i < b.N; i++ {
		_, _ = limiter.allow("benchmark", b.N+1, time.Minute)
	}
}
