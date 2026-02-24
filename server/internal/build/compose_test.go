package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestComposeDefaultGoProxyFallsBackOnProxyErrors(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	// this file: <repo>/server/internal/build/compose_test.go
	// compose: <repo>/docker-compose.yml
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	bs, err := os.ReadFile(filepath.Join(repoRoot, "docker-compose.yml"))
	if err != nil {
		t.Fatalf("read docker-compose.yml: %v", err)
	}

	compose := string(bs)
	if !strings.Contains(compose, "proxy.golang.org|direct") {
		t.Fatalf("docker-compose.yml should default GOPROXY to 'https://proxy.golang.org|direct' so Go can fall back when the proxy is unreachable")
	}
}

func TestComposeSupportsBuildNetworkOverride(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	// this file: <repo>/server/internal/build/compose_test.go
	// compose: <repo>/docker-compose.yml
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	bs, err := os.ReadFile(filepath.Join(repoRoot, "docker-compose.yml"))
	if err != nil {
		t.Fatalf("read docker-compose.yml: %v", err)
	}

	compose := string(bs)
	if !strings.Contains(compose, "SC_BUILD_NETWORK") {
		t.Fatalf("docker-compose.yml should support SC_BUILD_NETWORK to override build.network for restricted builder environments")
	}
}

func TestComposeSupportsBuildDNSOverride(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	// this file: <repo>/server/internal/build/compose_test.go
	// compose: <repo>/docker-compose.yml
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	bs, err := os.ReadFile(filepath.Join(repoRoot, "docker-compose.yml"))
	if err != nil {
		t.Fatalf("read docker-compose.yml: %v", err)
	}

	compose := string(bs)
	if !strings.Contains(compose, "SC_BUILD_DNS") {
		t.Fatalf("docker-compose.yml should support SC_BUILD_DNS to override build-time DNS in restricted builder environments")
	}
}
