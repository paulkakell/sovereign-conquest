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
		"ARG SC_BUILD_DNS",
		"ARG SC_USE_VENDOR",
		"./scripts/build_api.sh",
		"/tmp/sc-build.log",
		"tail -n 200",
	} {
		if !strings.Contains(dockerfile, want) {
			t.Fatalf("server/Dockerfile missing %q", want)
		}
	}

	// The Dockerfile delegates module download/build logic to a script (keeps the
	// Dockerfile short enough that some build UIs don't truncate the important Go
	// error output).
	scriptPath := filepath.Join(repoRoot, "server", "scripts", "build_api.sh")
	sbs, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read %s: %v", scriptPath, err)
	}
	script := string(sbs)
	for _, want := range []string{
		"go mod download",
		"-mod=vendor",
		"SC_BUILD_DNS",
		"SC_USE_VENDOR",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("server/scripts/build_api.sh missing %q", want)
		}
	}
}
