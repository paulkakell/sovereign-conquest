package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"sovereignconquest/internal/auth"
	"sovereignconquest/internal/config"
	"sovereignconquest/internal/game"
	"sovereignconquest/internal/schema"
)

const integrationPassword = "Synthetic-Account-Password-2026!"

type adminFixture struct {
	t       *testing.T
	pool    *pgxpool.Pool
	handler http.Handler
	cfg     config.Config
}

func newAdminFixture(t *testing.T) *adminFixture {
	t.Helper()
	url := os.Getenv("SC_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("SC_TEST_DATABASE_URL is required for real PostgreSQL integration")
	}
	ctx := context.Background()
	root, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	namespace := fmt.Sprintf("sc_admin_test_%d", time.Now().UnixNano())
	identifier := pgx.Identifier{namespace}.Sanitize()
	if _, err := root.Exec(ctx, `CREATE SCHEMA `+identifier); err != nil {
		root.Close()
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = namespace
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close(); _, _ = root.Exec(ctx, `DROP SCHEMA `+identifier+` CASCADE`); root.Close() })
	if err := schema.PrepareSeasonSchema(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := schema.Ensure(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// Re-applying migrations is an explicit supported startup path.
	if err := schema.Ensure(ctx, pool); err != nil {
		t.Fatal(err)
	}
	hash, err := auth.HashPassword(integrationPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO sectors(id,name) VALUES (1,'Home'),(2,'Frontier')`); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"admin", "second", "pilot"} {
		if _, err := pool.Exec(ctx, `INSERT INTO users(id,username,password_hash,is_admin) VALUES ($1,$1,$2,$3)`, id, hash, id != "pilot"); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO players(id,user_id,sector_id,season_id,credits,turns) VALUES ($1,$2,1,1,1000,50)`, id+"-player", id); err != nil {
			t.Fatal(err)
		}
	}
	appCfg := config.Config{JWTSecret: strings.Repeat("s", 64), AdminSecret: strings.Repeat("a", 64), TurnRegenSeconds: 120}
	s := &Server{Cfg: appCfg, Pool: pool}
	return &adminFixture{t: t, pool: pool, handler: RequireAdministrator(appCfg, pool, s.Router()), cfg: appCfg}
}

func (f *adminFixture) token(id string) string {
	f.t.Helper()
	var v int64
	if err := f.pool.QueryRow(context.Background(), `SELECT session_version FROM users WHERE id=$1`, id).Scan(&v); err != nil {
		f.t.Fatal(err)
	}
	token, err := auth.MintTokenForSession(f.cfg.JWTSecret, id, id+"-player", v, time.Hour)
	if err != nil {
		f.t.Fatal(err)
	}
	return token
}
func (f *adminFixture) request(method, path, token string, body any) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	r := httptest.NewRequest(method, path, bytes.NewReader(data))
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	r.Header.Set("X-Admin-Secret", f.cfg.AdminSecret)
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w
}
func (f *adminFixture) expect(status int, method, path, token string, body any) *httptest.ResponseRecorder {
	f.t.Helper()
	w := f.request(method, path, token, body)
	if w.Code != status {
		f.t.Fatalf("%s %s: status %d, want %d: %s", method, path, w.Code, status, w.Body.String())
	}
	return w
}
func (f *adminFixture) user(id string) adminUser {
	f.t.Helper()
	w := f.expect(200, "GET", "/api/admin/users/"+id, f.token("admin"), nil)
	var data struct {
		User adminUser `json:"user"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		f.t.Fatal(err)
	}
	return data.User
}
func (f *adminFixture) moderate(id, action string, until *time.Time) {
	f.t.Helper()
	u := f.user(id)
	f.expect(200, "POST", "/api/admin/users/"+id+"/moderation", f.token("admin"), moderationRequest{Revision: &u.Revision, Action: action, Reason: "Integration moderation reason", SuspendedUntil: until})
}

func TestAdminIntegrationAuthorization(t *testing.T) {
	f := newAdminFixture(t)
	for _, route := range []struct{ method, path string }{
		{"GET", "/api/admin/users"}, {"GET", "/api/admin/users/pilot"}, {"PATCH", "/api/admin/users/pilot"},
		{"POST", "/api/admin/users/pilot/moderation"}, {"GET", "/api/admin/ansi_map"}, {"POST", "/api/admin/soft_wipe"},
	} {
		f.expect(401, route.method, route.path, "", nil)
		f.expect(403, route.method, route.path, f.token("pilot"), nil)
	}
	mismatch, err := auth.MintToken(f.cfg.JWTSecret, "admin", "pilot-player", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	f.expect(401, "GET", "/api/admin/users", mismatch, nil)
	_, err = f.pool.Exec(context.Background(), `UPDATE users SET must_change_password=true WHERE id='admin'`)
	if err != nil {
		t.Fatal(err)
	}
	f.expect(403, "GET", "/api/admin/users", f.token("admin"), nil)
	f.expect(403, "POST", "/api/admin/soft_wipe", f.token("admin"), nil)
	f.expect(200, "GET", "/api/state", f.token("admin"), nil)
}

func TestAdminIntegrationEditsAndAudit(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()
	old := f.token("pilot")
	u := f.user("pilot")
	patch := map[string]any{"revision": u.Revision, "username": "renamed", "player": map[string]any{
		"credits": "0", "xp": "20000", "ship_type": "FREIGHTER", "ship_cargo_upgrades": 2, "ship_turn_upgrades": 1,
		"turns": 80, "turns_max": 200, "sector_id": 2, "cargo_max": 100, "cargo_ore": 10, "cargo_organics": 20, "cargo_equipment": 30, "season_id": 1,
	}}
	f.expect(200, "PATCH", "/api/admin/users/pilot", f.token("admin"), patch)
	u = f.user("pilot")
	if u.Username != "renamed" || u.Player.Credits != 0 || u.Player.SectorID != 2 || u.Player.Level != game.LevelForXP(20000) || u.Player.CargoEquipment != 30 {
		t.Fatalf("bad account: %+v", u)
	}
	f.expect(401, "GET", "/api/state", old, nil)
	f.expect(409, "PATCH", "/api/admin/users/pilot", f.token("admin"), patch)
	f.expect(409, "PATCH", "/api/admin/users/pilot", f.token("admin"), map[string]any{"revision": u.Revision, "username": "admin"})
	f.expect(400, "PATCH", "/api/admin/users/pilot", f.token("admin"), map[string]any{"revision": u.Revision, "username": "rollback", "player": map[string]any{"sector_id": 99999}})
	if f.user("pilot").Username != "renamed" {
		t.Fatal("invalid multi-field edit was partially committed")
	}
	var count int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM player_discoveries WHERE player_id='pilot-player' AND sector_id=2`).Scan(&count); err != nil || count != 1 {
		t.Fatal("teleport did not reveal destination", err)
	}
	var audit string
	if err := f.pool.QueryRow(ctx, `SELECT details::text FROM admin_audit_log WHERE action='USER_UPDATE'`).Scan(&audit); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(audit, "renamed") || strings.Contains(audit, "password_hash") || strings.Contains(audit, integrationPassword) {
		t.Fatal("audit details invalid")
	}
	f.expect(404, "GET", "/api/admin/users/missing", f.token("admin"), nil)
	for _, payload := range []any{map[string]any{"username": "missingversion"}, map[string]any{"revision": u.Revision, "session_version": 0}, map[string]any{"revision": u.Revision, "id": "replace"}, map[string]any{"revision": u.Revision, "player": map[string]any{"credits": 1}}} {
		f.expect(400, "PATCH", "/api/admin/users/pilot", f.token("admin"), payload)
	}
}

func TestAdminIntegrationSuspensionBanAndRestore(t *testing.T) {
	f := newAdminFixture(t)
	old := f.token("pilot")
	future := time.Now().Add(time.Hour)
	f.moderate("pilot", "suspend", &future)
	for _, route := range []struct{ method, path string }{
		{"GET", "/api/state"}, {"POST", "/api/command"}, {"POST", "/api/change_password"}, {"GET", "/api/messages/inbox"},
		{"POST", "/api/messages/send"}, {"GET", "/api/messages/attachments/1"}, {"POST", "/api/bug_report"},
	} {
		f.expect(403, route.method, route.path, old, nil)
	}
	f.expect(403, "POST", "/api/login", "", map[string]any{"username": "pilot", "password": integrationPassword})
	f.expect(401, "POST", "/api/login", "", map[string]any{"username": "pilot", "password": "wrong-password"})
	if _, err := f.pool.Exec(context.Background(), `UPDATE users SET suspended_until=now()-interval '1 second' WHERE id='pilot'`); err != nil {
		t.Fatal(err)
	}
	if f.user("pilot").EffectiveStatus != "active" {
		t.Fatal("suspension did not expire")
	}
	f.expect(401, "GET", "/api/state", old, nil)
	f.expect(200, "POST", "/api/login", "", map[string]any{"username": "pilot", "password": integrationPassword})
	f.moderate("pilot", "ban", nil)
	f.expect(403, "POST", "/api/login", "", map[string]any{"username": "pilot", "password": integrationPassword})
	f.moderate("pilot", "activate", nil)
	f.expect(401, "GET", "/api/state", old, nil)
	f.expect(200, "POST", "/api/login", "", map[string]any{"username": "pilot", "password": integrationPassword})
	f.moderate("pilot", "suspend", nil)
	f.expect(403, "GET", "/api/state", f.token("pilot"), nil)
	u := f.user("pilot")
	past := time.Now().Add(-time.Hour)
	for _, body := range []moderationRequest{
		{Revision: &u.Revision, Action: "suspend", Reason: "reason", SuspendedUntil: &past},
		{Revision: &u.Revision, Action: "ban", Reason: "reason", SuspendedUntil: &future},
		{Revision: &u.Revision, Action: "activate", Reason: ""},
		{Revision: &u.Revision, Action: "delete", Reason: "reason"},
	} {
		f.expect(400, "POST", "/api/admin/users/pilot/moderation", f.token("admin"), body)
	}
}

func TestAdminIntegrationPasswordResetAndRoleChanges(t *testing.T) {
	f := newAdminFixture(t)
	old := f.token("pilot")
	u := f.user("pilot")
	newPassword := "Reset-Password-Fixture-2026!"
	f.expect(200, "PATCH", "/api/admin/users/pilot", f.token("admin"), map[string]any{"revision": u.Revision, "new_password": newPassword})
	f.expect(401, "GET", "/api/state", old, nil)
	f.expect(401, "POST", "/api/login", "", map[string]any{"username": "pilot", "password": integrationPassword})
	login := f.expect(200, "POST", "/api/login", "", map[string]any{"username": "pilot", "password": newPassword})
	var body authResponse
	if err := json.Unmarshal(login.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	f.expect(403, "POST", "/api/command", body.Token, map[string]any{"type": "SCAN"})
	changed := f.expect(200, "POST", "/api/change_password", body.Token, map[string]any{"old_password": newPassword, "new_password": integrationPassword})
	var replacement struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(changed.Body.Bytes(), &replacement); err != nil {
		t.Fatal(err)
	}
	f.expect(401, "GET", "/api/state", body.Token, nil)
	f.expect(200, "GET", "/api/state", replacement.Token, nil)
	u = f.user("pilot")
	if u.MustChangePassword {
		t.Fatal("required change flag remained set")
	}
	f.expect(200, "PATCH", "/api/admin/users/pilot", f.token("admin"), map[string]any{"revision": u.Revision, "is_admin": true})
	f.expect(401, "GET", "/api/admin/users", replacement.Token, nil)
	f.expect(200, "GET", "/api/admin/users", f.token("pilot"), nil)
	promotedToken := f.token("pilot")
	u = f.user("pilot")
	f.expect(200, "PATCH", "/api/admin/users/pilot", f.token("admin"), map[string]any{"revision": u.Revision, "is_admin": false})
	f.expect(401, "POST", "/api/admin/soft_wipe", promotedToken, nil)
	var audit string
	if err := f.pool.QueryRow(context.Background(), `SELECT string_agg(details::text,' ') FROM admin_audit_log`).Scan(&audit); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(audit, newPassword) || strings.Contains(audit, "$2a$") || strings.Contains(audit, "password_hash") {
		t.Fatal("credential leaked to audit")
	}
}

func TestAdminIntegrationLockoutAndConcurrency(t *testing.T) {
	f := newAdminFixture(t)
	u := f.user("admin")
	f.expect(409, "PATCH", "/api/admin/users/admin", f.token("admin"), map[string]any{"revision": u.Revision, "is_admin": false})
	for _, action := range []string{"suspend", "ban"} {
		f.expect(409, "POST", "/api/admin/users/admin/moderation", f.token("admin"), moderationRequest{Revision: &u.Revision, Action: action, Reason: "self"})
	}
	adminToken, secondToken := f.token("admin"), f.token("second")
	start := make(chan struct{})
	codes := make(chan int, 2)
	var wg sync.WaitGroup
	for _, item := range []struct{ token, target string }{{adminToken, "second"}, {secondToken, "admin"}} {
		wg.Add(1)
		go func(token, target string) {
			defer wg.Done()
			<-start
			w := f.request("PATCH", "/api/admin/users/"+target, token, map[string]any{"revision": 0, "is_admin": false})
			codes <- w.Code
		}(item.token, item.target)
	}
	close(start)
	wg.Wait()
	close(codes)
	successes := 0
	for code := range codes {
		if code == 200 {
			successes++
		} else if code != 401 && code != 403 {
			t.Fatalf("unexpected concurrent result %d", code)
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent demotions succeeded %d times", successes)
	}
	var remaining int
	if err := f.pool.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE is_admin`).Scan(&remaining); err != nil || remaining != 1 {
		t.Fatal("last administrator lost", err)
	}
}

func TestAdminIntegrationSearchAndBootstrapRename(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()
	// Accounts without a player remain visible and editable.
	if _, err := f.pool.Exec(ctx, `INSERT INTO users(id,username,password_hash) SELECT 'orphan','orphan',password_hash FROM users WHERE id='pilot'`); err != nil {
		t.Fatal(err)
	}
	if f.user("orphan").Player != nil {
		t.Fatal("unexpected orphan player")
	}
	w := f.expect(200, "GET", "/api/admin/users?q=PIL", f.token("admin"), nil)
	var page struct {
		Users []adminUser `json:"users"`
		Next  string      `json:"next_cursor"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil || len(page.Users) != 1 || page.Users[0].ID != "pilot" {
		t.Fatal("search mismatch", err)
	}
	for i := 0; i < 55; i++ {
		if _, err := f.pool.Exec(ctx, `INSERT INTO users(id,username,password_hash) VALUES ($1,$1,'unused')`, fmt.Sprintf("page%03d", i)); err != nil {
			t.Fatal(err)
		}
	}
	w = f.expect(200, "GET", "/api/admin/users", f.token("admin"), nil)
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil || len(page.Users) != 50 || page.Next == "" {
		t.Fatal("pagination mismatch", err)
	}
	w = f.expect(200, "GET", "/api/admin/users?after="+page.Next, f.token("admin"), nil)
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil || len(page.Users) != 9 || page.Next != "" {
		t.Fatal("last page mismatch", err)
	}
	f.moderate("pilot", "ban", nil)
	w = f.expect(200, "GET", "/api/admin/users?status=banned", f.token("admin"), nil)
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil || len(page.Users) != 1 || page.Users[0].ID != "pilot" {
		t.Fatal("status filter mismatch", err)
	}
	u := f.user("admin")
	f.expect(200, "PATCH", "/api/admin/users/admin", f.token("admin"), map[string]any{"revision": u.Revision, "username": "newadmin"})
	result, err := game.EnsureInitialAdmin(ctx, f.pool, "admin", integrationPassword)
	if err != nil || result.Created {
		t.Fatal("renamed bootstrap account recreated", err)
	}
}

func TestAdminIntegrationMigrationRollbackAndAuditFailure(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()
	u := f.user("pilot")
	// Reverse only the additive account migration in this disposable schema.
	if _, err := f.pool.Exec(ctx, `ALTER TABLE users DROP COLUMN session_version, DROP COLUMN account_revision,
		DROP COLUMN account_status, DROP COLUMN suspended_until, DROP COLUMN moderation_reason, DROP COLUMN moderated_at`); err != nil {
		t.Fatal(err)
	}
	var name string
	if err := f.pool.QueryRow(ctx, `SELECT username FROM users WHERE id='pilot'`).Scan(&name); err != nil || name != u.Username {
		t.Fatal("legacy read failed after rollback", err)
	}
	if err := schema.Ensure(ctx, f.pool); err != nil {
		t.Fatal(err)
	}
	backfilled := f.user("pilot")
	if backfilled.EffectiveStatus != "active" || backfilled.Revision != 0 || backfilled.Username != u.Username {
		t.Fatal("migration backfill damaged account")
	}
	if _, err := f.pool.Exec(ctx, `DROP TABLE admin_audit_log`); err != nil {
		t.Fatal(err)
	}
	f.expect(500, "PATCH", "/api/admin/users/pilot", f.token("admin"), map[string]any{"revision": 0, "username": "nosave"})
	if f.user("pilot").Username != u.Username {
		t.Fatal("update committed without audit")
	}
}

func BenchmarkAuthenticatedSession(b *testing.B) {
	url := os.Getenv("SC_TEST_DATABASE_URL")
	if url == "" {
		b.Skip("SC_TEST_DATABASE_URL is required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Close()
	// A temporary schema keeps benchmark records isolated from other tests.
	namespace := fmt.Sprintf("sc_session_bench_%d", time.Now().UnixNano())
	id := pgx.Identifier{namespace}.Sanitize()
	if _, err := pool.Exec(ctx, `CREATE SCHEMA `+id); err != nil {
		b.Fatal(err)
	}
	defer func() { _, _ = pool.Exec(ctx, `DROP SCHEMA `+id+` CASCADE`) }()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		b.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = namespace
	isolated, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		b.Fatal(err)
	}
	defer isolated.Close()
	if err := schema.PrepareSeasonSchema(ctx, isolated); err != nil {
		b.Fatal(err)
	}
	if err := schema.Ensure(ctx, isolated); err != nil {
		b.Fatal(err)
	}
	if _, err := isolated.Exec(ctx, `INSERT INTO sectors(id,name) VALUES(1,'bench');
		INSERT INTO users(id,username,password_hash) VALUES('bench','bench','unused');
		INSERT INTO players(id,user_id,sector_id,season_id) VALUES('bench','bench',1,1)`); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := auth.LoadSession(ctx, isolated, "bench", "bench"); err != nil {
				b.Error(err)
			}
		}
	})
	b.StopTimer()
}
