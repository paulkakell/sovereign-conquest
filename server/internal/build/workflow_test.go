package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGHCRPublishWorkflowBuildsAndValidatesReleaseImage(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "publish-ghcr.yml")
	bs, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read .github/workflows/publish-ghcr.yml: %v", err)
	}

	workflow := string(bs)
	for _, want := range []string{
		"name: Publish GHCR Image",
		"workflow_dispatch:",
		"branches: [main]",
		"version=\"$(tr -d '\\r\\n' < VERSION)\"",
		"image=\"ghcr.io/${GITHUB_REPOSITORY,,}\"",
		"actions/checkout@v5",
		"docker/setup-buildx-action@v3",
		"docker/login-action@v3",
		"docker/build-push-action@v6",
		"context: .",
		"file: ./Dockerfile",
		"${{ steps.release.outputs.image }}:main",
		"${{ steps.release.outputs.image }}:${{ steps.release.outputs.version }}",
		"${{ steps.release.outputs.image }}:sha-${{ steps.release.outputs.short_sha }}",
		"actions/attest-build-provenance@v3",
		"aquasecurity/trivy-action@0.35.0",
		"bash scripts/release-smoke.sh",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("GHCR publish workflow missing %q", want)
		}
	}

	legacyPath := filepath.Join(repoRoot, ".github", "workflows", "docker-image.yml")
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy workflow should be absent; stat error=%v", err)
	}
}
