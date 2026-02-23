package config

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestVersionMatchesRootVERSIONFile(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	// this file: <repo>/server/internal/config/version_test.go
	// VERSION file: <repo>/VERSION
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	bs, err := os.ReadFile(filepath.Join(repoRoot, "VERSION"))
	if err != nil {
		t.Fatalf("read VERSION file: %v", err)
	}

	fileVersion := strings.TrimSpace(string(bs))
	if fileVersion == "" {
		t.Fatal("VERSION file is empty")
	}

	// Enforce the project's xx.xx.xx versioning convention.
	if !regexp.MustCompile(`^\d{2}\.\d{2}\.\d{2}$`).MatchString(fileVersion) {
		t.Fatalf("VERSION file does not match xx.xx.xx format: %q", fileVersion)
	}

	if fileVersion != Version {
		t.Fatalf("config.Version mismatch: VERSION=%q config.Version=%q", fileVersion, Version)
	}
}
