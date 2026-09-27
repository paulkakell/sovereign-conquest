package healthcheck

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestEndpoint(t *testing.T) {
	for _, tt := range []struct {
		address string
		want    string
	}{
		{"", "http://127.0.0.1:8080/api/livez"},
		{":9090", "http://127.0.0.1:9090/api/livez"},
		{"0.0.0.0:9090", "http://127.0.0.1:9090/api/livez"},
		{"[::]:9090", "http://[::1]:9090/api/livez"},
		{"[::ffff:0.0.0.0]:9090", "http://127.0.0.1:9090/api/livez"},
		{"127.0.0.1:8081", "http://127.0.0.1:8081/api/livez"},
		{"192.0.2.1:8081", "http://192.0.2.1:8081/api/livez"},
		{"[::1]:8081", "http://[::1]:8081/api/livez"},
		{"localhost:8081", "http://localhost:8081/api/livez"},
	} {
		t.Run(tt.address, func(t *testing.T) {
			got, err := endpoint(tt.address)
			if err != nil || got != tt.want {
				t.Fatalf("endpoint(%q) = %q, %v; want %q", tt.address, got, err, tt.want)
			}
		})
	}
}

func TestInvalidAddressDoesNotExposeInput(t *testing.T) {
	for _, address := range []string{"localhost", ":", ":0", ":65536", ":-1", "http://secret@example.com:8080"} {
		t.Run(address, func(t *testing.T) {
			err := Run(address)
			if err == nil || strings.Contains(err.Error(), "secret") {
				t.Fatalf("expected safe validation error, got %v", err)
			}
		})
	}
}

func TestRunRequiresExactOK(t *testing.T) {
	for _, status := range []int{200, 201, 204, 301, 401, 500, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/livez" || r.URL.RawQuery != "" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				if r.Header.Get("Authorization") != "" {
					t.Error("probe sent authentication")
				}
				w.WriteHeader(status)
				_, _ = w.Write([]byte("response-secret"))
			}))
			defer server.Close()
			err := Run(server.Listener.Addr().String())
			if (err == nil) != (status == http.StatusOK) {
				t.Fatalf("status %d: unexpected error %v", status, err)
			}
			if err != nil && strings.Contains(err.Error(), "response-secret") {
				t.Fatalf("response content leaked: %v", err)
			}
		})
	}
}

func TestRunWithoutApplicationConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "not-a-database-url")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ADMIN_SECRET", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	// Verify that the wildcard bind address can reach the local process.
	if err := Run("0.0.0.0:" + port); err != nil {
		t.Fatal(err)
	}
}

func TestRunIPv6Wildcard(t *testing.T) {
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 unavailable: %v", err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	_ = server.Listener.Close()
	server.Listener = listener
	server.Start()
	defer server.Close()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(net.JoinHostPort("::", port)); err != nil {
		t.Fatal(err)
	}
}

func TestRunRejectsRedirects(t *testing.T) {
	var targetRequests atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetRequests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/secret", http.StatusFound)
	}))
	defer server.Close()
	if err := Run(server.Listener.Addr().String()); err == nil {
		t.Fatal("redirect reported healthy")
	}
	if targetRequests.Load() != 0 {
		t.Fatal("probe followed redirect")
	}
}

func TestRunTimeout(t *testing.T) {
	if timeout >= 5*time.Second {
		t.Fatal("probe deadline must be shorter than the Docker health timeout")
	}
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	start := time.Now()
	if err := check(server.Listener.Addr().String(), 50*time.Millisecond); err == nil {
		t.Fatal("stalled endpoint reported healthy")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("timeout did not bound probe duration: %v", elapsed)
	}
}

func TestRunConnectionFailure(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	address := server.Listener.Addr().String()
	server.Close()
	if err := Run(address); err == nil || err.Error() != "liveness request failed" {
		t.Fatalf("expected safe connection failure, got %v", err)
	}
}

func TestRunIgnoresProxy(t *testing.T) {
	// Isolate net/http's cached proxy environment from other tests. The hostname
	// case variant resolves locally without matching Go's literal localhost
	// proxy exemption. Check that premise before testing the actual probe.
	if os.Getenv("SC_HEALTHCHECK_PROXY_TEST") == "1" {
		address := os.Getenv("SC_HEALTHCHECK_TEST_ADDR")
		target, err := endpoint(address)
		if err != nil {
			t.Fatal(err)
		}
		request, err := http.NewRequest(http.MethodGet, target, nil)
		if err != nil {
			t.Fatal(err)
		}
		proxyURL, err := http.ProxyFromEnvironment(request)
		if err != nil || proxyURL == nil {
			t.Fatalf("test requires a target normally sent through the proxy: %v", err)
		}
		if err := Run(address); err != nil {
			t.Fatal(err)
		}
		return
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	var proxyRequests atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		proxyRequests.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer proxy.Close()
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		t.Setenv(key, proxy.URL)
	}
	for _, key := range []string{"NO_PROXY", "no_proxy"} {
		t.Setenv(key, "")
	}
	t.Setenv("REQUEST_METHOD", "")
	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SC_HEALTHCHECK_PROXY_TEST", "1")
	t.Setenv("SC_HEALTHCHECK_TEST_ADDR", net.JoinHostPort("LOCALHOST", port))
	command := exec.Command(os.Args[0], "-test.run=^TestRunIgnoresProxy$")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("proxy isolation probe failed: %v\n%s", err, output)
	}
	if proxyRequests.Load() != 0 {
		t.Fatal("probe used the configured HTTP proxy")
	}
}
