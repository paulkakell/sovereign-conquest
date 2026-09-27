package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Run the real main entrypoint in a subprocess so configuration validation,
// package initialization and exit codes are exercised together.
func TestAPIMainProcess(t *testing.T) {
	switch os.Getenv("SC_API_MAIN_TEST_ROLE") {
	case "healthcheck":
		os.Args = []string{"sovereign-api", "healthcheck"}
	case "serve":
		os.Args = []string{"sovereign-api"}
	default:
		return
	}
	main()
}

func TestHealthcheckCommandDoesNotRequireApplicationConfiguration(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusServiceUnavailable} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/livez" {
				t.Errorf("unexpected healthcheck request: %s", r.URL.Path)
			}
			w.WriteHeader(status)
			_, _ = w.Write([]byte("response-secret"))
		}))
		output, err := runAPIProcess(t, "healthcheck", server.Listener.Addr().String())
		server.Close()
		if (err == nil) != (status == http.StatusOK) {
			t.Fatalf("HTTP %d: command returned %v\n%s", status, err, output)
		}
		for _, forbidden := range []string{"configuration validation failed", "database connection failed", "response-secret", "environment-secret"} {
			if strings.Contains(output, forbidden) {
				t.Fatalf("healthcheck initialized configuration or leaked content: %s", output)
			}
		}
	}
}

func TestServerCommandStillValidatesProductionConfiguration(t *testing.T) {
	output, err := runAPIProcess(t, "serve", ":8080")
	if err == nil {
		t.Fatal("normal server startup accepted invalid production configuration")
	}
	if !strings.Contains(output, "configuration validation failed") {
		t.Fatalf("expected configuration validation failure, got: %s", output)
	}
	if strings.Contains(output, "database connection failed") {
		t.Fatalf("server connected to database before validating configuration: %s", output)
	}
}

func runAPIProcess(t *testing.T, role, address string) (string, error) {
	t.Helper()
	t.Setenv("SC_API_MAIN_TEST_ROLE", role)
	t.Setenv("HTTP_ADDR", address)
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "invalid-environment-secret")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("INITIAL_ADMIN_PASSWORD", "")
	t.Setenv("ADMIN_SECRET", "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAPIMainProcess$")
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("API command exceeded timeout: %s", output)
	}
	return string(output), err
}
