package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"sovereignconquest/internal/auth"
)

func ref[T any](v T) *T { return &v }

func TestSessionRestrictions(t *testing.T) {
	past, future := time.Now().Add(-time.Minute), time.Now().Add(time.Hour)
	for _, tc := range []struct {
		name    string
		session auth.Session
		version int64
		err     error
		status  int
	}{
		{"active", auth.Session{Status: "active", Version: 4}, 4, nil, 200},
		{"revoked", auth.Session{Status: "active", Version: 4}, 3, nil, 401},
		{"banned", auth.Session{Status: "banned"}, 0, nil, 403},
		{"indefinite suspension", auth.Session{Status: "suspended"}, 0, nil, 403},
		{"timed suspension", auth.Session{Status: "suspended", SuspendedUntil: &future}, 0, nil, 403},
		{"expired suspension", auth.Session{Status: "suspended", SuspendedUntil: &past, Version: 1}, 1, nil, 200},
		{"expired does not restore token", auth.Session{Status: "suspended", SuspendedUntil: &past, Version: 1}, 0, nil, 401},
		{"deleted account", auth.Session{}, 0, pgx.ErrNoRows, 401},
		{"unknown status fails closed", auth.Session{Status: "unknown"}, 0, nil, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ok := checkSession(w, tc.session, &auth.Claims{SessionVersion: tc.version}, tc.err)
			if w.Code != tc.status || ok != (tc.status == 200) {
				t.Fatalf("status=%d allowed=%t", w.Code, ok)
			}
		})
	}
}

func TestAdminPatchValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		patch adminPlayerPatch
	}{
		{"negative balance", adminPlayerPatch{Credits: ref(int64(-1))}},
		{"overflow balance", adminPlayerPatch{Credits: ref(int64(9000000000000001))}},
		{"negative XP", adminPlayerPatch{XP: ref(int64(-1))}},
		{"too many turns", adminPlayerPatch{Turns: ref(101)}},
		{"invalid capacity", adminPlayerPatch{CargoMax: ref(1000001)}},
		{"overfilled hold", adminPlayerPatch{CargoOre: ref(31)}},
		{"bad ship", adminPlayerPatch{ShipType: ref("CHEAT")}},
		{"bad upgrade", adminPlayerPatch{ShipCargoUpgrades: ref(21)}},
		{"bad turn upgrade", adminPlayerPatch{ShipTurnUpgrades: ref(11)}},
		{"bad sector", adminPlayerPatch{SectorID: ref(0)}},
		{"overflow sector", adminPlayerPatch{SectorID: ref(2147483648)}},
		{"bad season", adminPlayerPatch{SeasonID: ref(-1)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := adminPlayer{ShipType: "SCOUT", TurnsMax: 100, CargoMax: 30, SectorID: 1, SeasonID: 1}
			if _, err := applyAdminPlayerPatch(&p, tc.patch); err == nil {
				t.Fatal("invalid patch accepted")
			}
		})
	}
	p := adminPlayer{ShipType: "SCOUT", TurnsMax: 100, CargoMax: 30, SectorID: 1, SeasonID: 1}
	if _, err := applyAdminPlayerPatch(&p, adminPlayerPatch{Credits: ref(int64(0)), XP: ref(int64(20000)), CargoOre: ref(30)}); err != nil || p.Level <= 1 || p.CargoOre != 30 {
		t.Fatalf("valid patch: %+v, %v", p, err)
	}
}

func TestAdminStrictDecodingAndPasswordValidation(t *testing.T) {
	for _, body := range []string{`{"revision":0,"password_hash":"x"}`, `{"revision":0,"id":"other"}`, `{"revision":0,"created_at":"x"}`, `{"revision":0,"player":{"level":5}}`, `{"revision":0,"is_admin":"true"}`, `{} {}`, `[]`, strings.Repeat(" ", 33000) + `{}`} {
		w := httptest.NewRecorder()
		var patch adminUserPatch
		if decodeAdminJSON(w, httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(body)), &patch) {
			t.Fatal("unsafe payload accepted")
		}
	}
	for _, body := range []string{`{"revision":0,"new_password":"short"}`, `{"revision":0,"username":"x"}`, `{"revision":0,"username":"bad\nname"}`} {
		w := httptest.NewRecorder()
		(&Server{}).handleAdminUserUpdate(w, httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(body)))
		if w.Code != 400 {
			t.Fatalf("status=%d", w.Code)
		}
	}
}
