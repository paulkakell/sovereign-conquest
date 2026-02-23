package sqlcheck

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadPlayerForUpdate_LocksOnlyPlayersTable(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	// Regression test: Postgres forbids FOR UPDATE on the nullable side of an OUTER JOIN.
	// LoadPlayerForUpdate joins corp_members/corporations via LEFT JOIN, so it must lock only
	// the players table (OF p).
	storeGo := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "game", "store.go"))
	bs, err := os.ReadFile(storeGo)
	if err != nil {
		t.Fatalf("read store.go: %v", err)
	}

	if !strings.Contains(string(bs), "FOR UPDATE OF p") {
		t.Fatalf("LoadPlayerForUpdate query must use 'FOR UPDATE OF p' to avoid outer-join locking errors")
	}
}
