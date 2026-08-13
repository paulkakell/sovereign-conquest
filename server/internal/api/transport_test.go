package api

import (
	"net/http"
	"net/http/httptest"
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
