package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
}

func TestOfflineBuildArtifactsCommitted(t *testing.T) {
	root := repoRoot(t)

	for _, rel := range []string{
		filepath.Join("server", "go.sum"),
		filepath.Join("server", "vendor", "modules.txt"),
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to be committed for offline builds: %v", rel, err)
		}
	}

	bs, err := os.ReadFile(filepath.Join(root, "server", "vendor", "modules.txt"))
	if err != nil {
		t.Fatalf("read vendor/modules.txt: %v", err)
	}
	vendorModules := string(bs)
	for _, want := range []string{
		"github.com/go-chi/chi/v5",
		"github.com/jackc/pgx/v5",
		"golang.org/x/crypto",
	} {
		if !strings.Contains(vendorModules, want) {
			t.Fatalf("vendor/modules.txt missing %q", want)
		}
	}
}

func TestDockerignoreDoesNotExcludeVendoredModules(t *testing.T) {
	root := repoRoot(t)
	bs, err := os.ReadFile(filepath.Join(root, "server", ".dockerignore"))
	if err != nil {
		t.Fatalf("read server/.dockerignore: %v", err)
	}
	ignore := string(bs)

	for _, bad := range []string{"vendor", "vendor/", "**/vendor", "**/vendor/"} {
		if strings.Contains(ignore, bad) {
			t.Fatalf("server/.dockerignore must not exclude vendored modules; found %q", bad)
		}
	}
}
