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
		"docker/setup-buildx-action@v4",
		"docker/login-action@v3",
		"docker/build-push-action@v6",
		"context: .",
		"file: ./Dockerfile",
		"docker tag \"$candidate\" \"$IMAGE:main\"",
		"docker tag \"$candidate\" \"$IMAGE:$VERSION\"",
		"${{ steps.release.outputs.image }}:sha-${{ steps.release.outputs.short_sha }}",
		"actions/attest-build-provenance@v4",
		"aquasecurity/trivy-action@v0.36.0",
		"bash scripts/release-smoke.sh",
		"SC_IMAGE=sovereign-conquest-api:release-validation SC_EXPECT_WEB=false",
		"python3 scripts/container-security-check.py inventory",
		"python3 scripts/container-security-check.py vulnerabilities",
		"severity: UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL",
		"ignore-unfixed: false",
		"TRIVY_IGNOREFILE: /dev/null",
		"load: true",
		"pull: true",
		"trivy convert --format cyclonedx",
		"actions/upload-artifact@v4",
		"docker buildx imagetools inspect \"$candidate\" --format '{{json .Manifest}}'",
		"digest=\"$(jq -er '.digest' container-security/published-descriptor.json)\"",
		"docker buildx imagetools inspect \"$IMAGE@$digest\" --raw",
		"'.config.digest == $validated_id' container-security/published-manifest.json",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("GHCR publish workflow missing %q", want)
		}
	}

	for _, forbidden := range []string{"push: true", "ignore-unfixed: true", "continue-on-error: true", "docker push --quiet"} {
		if strings.Contains(workflow, forbidden) {
			t.Fatalf("publish workflow permits unsafe pre-validation behavior: %q", forbidden)
		}
	}

	// Publication must consume the validated image instead of rebuilding after checks.
	previous := -1
	for _, step := range []string{
		"- name: Build combined release candidate",
		"- name: Build standalone API release candidate",
		"- name: Verify both API runtime inventories",
		"- name: Scan combined release candidate",
		"- name: Scan standalone API release candidate",
		"- name: Export SBOMs and enforce container vulnerability policy",
		"- name: Smoke-test both release candidates",
		"- name: Push validated release image",
		"- name: Attest validated image provenance",
	} {
		index := strings.Index(workflow, step)
		if index <= previous {
			t.Fatalf("publish workflow step missing or out of order: %q", step)
		}
		previous = index
	}
	if !strings.Contains(workflow, "validated_id=\"$(jq -er '.image_id' container-security/combined-inventory.json)\"") ||
		!strings.Contains(workflow, "[[ \"$(docker image inspect --format '{{.Id}}' \"$candidate\")\" == \"$validated_id\" ]]") {
		t.Fatal("published image must match the image whose inventory was validated")
	}
	previous = -1
	for _, command := range []string{
		"docker push \"$candidate\"",
		"docker buildx imagetools inspect \"$candidate\"",
		"docker buildx imagetools inspect \"$IMAGE@$digest\" --raw",
		"'.config.digest == $validated_id'",
		"docker tag \"$candidate\" \"$IMAGE:$VERSION\"",
		"docker push \"$IMAGE:$VERSION\"",
		"docker push \"$IMAGE:main\"",
	} {
		index := strings.Index(workflow, command)
		if index <= previous {
			t.Fatalf("release aliases must follow registry verification; missing or out of order: %q", command)
		}
		previous = index
	}

	legacyPath := filepath.Join(repoRoot, ".github", "workflows", "docker-image.yml")
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy workflow should be absent; stat error=%v", err)
	}
}
