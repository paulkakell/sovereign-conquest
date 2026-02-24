package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestServerDockerfileSupportsRestrictedBuildEnvironments(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	// this file: <repo>/server/internal/build/dockerfile_test.go
	// dockerfile: <repo>/server/Dockerfile
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	bs, err := os.ReadFile(filepath.Join(repoRoot, "server", "Dockerfile"))
	if err != nil {
		t.Fatalf("read server/Dockerfile: %v", err)
	}

	dockerfile := string(bs)
	for _, want := range []string{
		"ARG GOPROXY",
		"proxy.golang.org|direct",
		"ARG GOSUMDB",
		"ARG SC_USE_VENDOR",
		"go mod download",
		"-mod=vendor",
	} {
		if !strings.Contains(dockerfile, want) {
			t.Fatalf("server/Dockerfile missing %q", want)
		}
	}
}
