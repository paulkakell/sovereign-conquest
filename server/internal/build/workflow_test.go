package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDockerImageWorkflowBuildsRootImage(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	bs, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "docker-image.yml"))
	if err != nil {
		t.Fatalf("read .github/workflows/docker-image.yml: %v", err)
	}

	workflow := string(bs)
	for _, want := range []string{
		"REGISTRY: ghcr.io",
		"IMAGE_NAME=${GITHUB_REPOSITORY,,}",
		"actions/checkout@v5",
		"docker/setup-buildx-action@v3",
		"docker/login-action@v3",
		"docker/metadata-action@v5",
		"docker/build-push-action@v6",
		"context: .",
		"file: ./Dockerfile",
		"cache-from: type=gha,scope=root",
		"cache-to: type=gha,mode=max,scope=root",
		"actions/attest-build-provenance@v3",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("docker-image workflow missing %q", want)
		}
	}
}
