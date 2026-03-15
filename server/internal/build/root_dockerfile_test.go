package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRootDockerfileBuildsCombinedImage(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	bs, err := os.ReadFile(filepath.Join(repoRoot, "Dockerfile"))
	if err != nil {
		t.Fatalf("read root Dockerfile: %v", err)
	}

	dockerfile := string(bs)
	for _, want := range []string{
		"FROM golang:1.22-alpine AS build",
		"COPY server/ /src/server/",
		"WORKDIR /src/server",
		"./scripts/build_api.sh",
		"COPY web/static/ /app/web/",
		"ENV WEB_ROOT=/app/web",
		"EXPOSE 8080",
	} {
		if !strings.Contains(dockerfile, want) {
			t.Fatalf("root Dockerfile missing %q", want)
		}
	}
}
