package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestComposeUsesCombinedGHCRImage(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))

	composeBytes, err := os.ReadFile(filepath.Join(root, "docker-compose.yml"))
	if err != nil {
		t.Fatalf("read docker-compose.yml: %v", err)
	}
	envBytes, err := os.ReadFile(filepath.Join(root, ".env.example"))
	if err != nil {
		t.Fatalf("read .env.example: %v", err)
	}

	compose := string(composeBytes)
	envExample := string(envBytes)

	for _, required := range []string{
		`image: "${SC_IMAGE:-ghcr.io/paulkakell/sovereign-conquest:main}"`,
		`pull_policy: "${SC_PULL_POLICY:-always}"`,
		`WEB_ROOT: "/app/web"`,
		`127.0.0.1:${WEB_PORT:-3000}:8080`,
		`127.0.0.1:${API_PORT:-8080}:8080`,
		"read_only: true",
		"no-new-privileges:true",
		"cap_drop:",
	} {
		if !strings.Contains(compose, required) {
			t.Fatalf("docker-compose.yml missing %q", required)
		}
	}

	for _, forbidden := range []string{
		"\n    build:\n",
		"\n  web:\n",
		"context: ./server",
		"context: ./web",
	} {
		if strings.Contains(compose, forbidden) {
			t.Fatalf("docker-compose.yml still contains build-era contract %q", forbidden)
		}
	}

	for _, required := range []string{
		"SC_IMAGE=ghcr.io/paulkakell/sovereign-conquest:main",
		"SC_PULL_POLICY=always",
		"WEB_PORT=3000",
		"API_PORT=8080",
	} {
		if !strings.Contains(envExample, required) {
			t.Fatalf(".env.example missing %q", required)
		}
	}
}
