package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCompatibilityAssetsAreShipped(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))

	index, err := os.ReadFile(filepath.Join(root, "web", "static", "index.html"))
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}
	compat, err := os.ReadFile(filepath.Join(root, "web", "static", "compat-010601.js"))
	if err != nil {
		t.Fatalf("read compatibility script: %v", err)
	}

	shell := string(index)
	client := string(compat)
	for _, required := range []string{
		"/compat-010601.js?v=01.06.02",
		"/app.js?v=01.06.02",
		"/style.css?v=01.06.02",
		"/bug.html",
	} {
		if !strings.Contains(shell, required) {
			t.Fatalf("index.html missing %q", required)
		}
	}
	for _, required := range []string{
		"correctedCommandPayload",
		"normalizePayload",
		"showPasswordChange",
	} {
		if !strings.Contains(client, required) {
			t.Fatalf("compatibility script missing %q", required)
		}
	}
	if strings.Contains(shell, "/static/app.js") || strings.Contains(shell, "/static/style.css") {
		t.Fatal("obsolete static asset prefix remains")
	}
}
