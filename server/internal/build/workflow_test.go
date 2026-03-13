package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDockerPublishWorkflowBuildsServiceImages(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	// this file: <repo>/server/internal/build/workflow_test.go
	// workflow: <repo>/.github/workflows/docker-publish.yml
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	bs, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "docker-publish.yml"))
	if err != nil {
		t.Fatalf("read .github/workflows/docker-publish.yml: %v", err)
	}

	workflow := string(bs)
	for _, want := range []string{
		"IMAGE_BASENAME",
		"matrix:",
		"name: api",
		"context: ./server",
		"file: ./server/Dockerfile",
		"image_suffix: -api",
		"name: web",
		"context: ./web",
		"file: ./web/Dockerfile",
		"image_suffix: -web",
		"context: ${{ matrix.context }}",
		"file: ${{ matrix.file }}",
		"cache-from: type=gha,scope=${{ matrix.name }}",
		"cache-to: type=gha,mode=max,scope=${{ matrix.name }}",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("docker-publish workflow missing %q", want)
		}
	}

	for _, line := range strings.Split(workflow, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "context: ." || trimmed == `context: "."` || trimmed == `context: '.'` {
			t.Fatal("docker-publish workflow must not build the repository root; this project ships service Dockerfiles under ./server and ./web")
		}
	}
}
