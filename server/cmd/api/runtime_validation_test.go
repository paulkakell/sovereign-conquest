package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sovereignconquest/internal/config"
)

func validStartupConfig(t *testing.T) config.Config {
	t.Helper()
	t.Setenv("APP_ENV", "development")
	t.Setenv("PGPASSWORD", "")
	t.Setenv("PGSERVICE", "")
	t.Setenv("PGPASSFILE", filepath.Join(t.TempDir(), "absent"))
	return config.Config{
		DatabaseURL: "postgres://fixture:database-fixture@127.0.0.1/fixture?sslmode=require",
		JWTSecret:   strings.Repeat("j", 32), AdminSecret: strings.Repeat("a", 32),
		InitialAdminPass: strings.Repeat("p", 16), UniverseSectors: 40, TurnRegenSeconds: 120,
	}
}

func TestStartupSecretLengthsInEveryEnvironment(t *testing.T) {
	cases := []struct {
		name   string
		change func(*config.Config)
		want   string
	}{
		{"minimums", func(c *config.Config) {}, ""},
		{"empty database", func(c *config.Config) { c.DatabaseURL = "postgres://fixture@127.0.0.1/fixture?sslmode=require" }, "POSTGRES_PASSWORD"},
		{"one byte database", func(c *config.Config) { c.DatabaseURL = "postgres://fixture:x@127.0.0.1/fixture?sslmode=require" }, ""},
		{"long database", func(c *config.Config) {
			c.DatabaseURL = "postgres://fixture:" + strings.Repeat("d", 256) + "@127.0.0.1/fixture?sslmode=require"
		}, ""},
		{"empty jwt", func(c *config.Config) { c.JWTSecret = "" }, "JWT_SECRET"},
		{"short jwt", func(c *config.Config) { c.JWTSecret = strings.Repeat("j", 31) }, "JWT_SECRET"},
		{"padded short jwt", func(c *config.Config) { c.JWTSecret = " " + strings.Repeat("j", 31) + " " }, "JWT_SECRET"},
		{"long jwt", func(c *config.Config) { c.JWTSecret = strings.Repeat("j", 256) }, ""},
		{"optional admin", func(c *config.Config) { c.AdminSecret = "" }, ""},
		{"blank admin", func(c *config.Config) { c.AdminSecret = " \t " }, "ADMIN_SECRET"},
		{"short admin", func(c *config.Config) { c.AdminSecret = strings.Repeat("a", 31) }, "ADMIN_SECRET"},
		{"long admin", func(c *config.Config) { c.AdminSecret = strings.Repeat("a", 256) }, ""},
		{"empty bootstrap", func(c *config.Config) { c.InitialAdminPass = "" }, "INITIAL_ADMIN_PASSWORD"},
		{"short bootstrap", func(c *config.Config) { c.InitialAdminPass = strings.Repeat("p", 15) }, "INITIAL_ADMIN_PASSWORD"},
		{"maximum bootstrap", func(c *config.Config) { c.InitialAdminPass = strings.Repeat("p", 72) }, ""},
		{"long bootstrap", func(c *config.Config) { c.InitialAdminPass = strings.Repeat("p", 73) }, "INITIAL_ADMIN_PASSWORD"},
		{"unicode maximum", func(c *config.Config) { c.InitialAdminPass = strings.Repeat("é", 36) }, ""},
		{"unicode too long", func(c *config.Config) { c.InitialAdminPass = strings.Repeat("é", 37) }, "INITIAL_ADMIN_PASSWORD"},
		{"trim bootstrap like hashing", func(c *config.Config) { c.InitialAdminPass = " " + strings.Repeat("p", 72) + "\t" }, ""},
	}
	for _, environment := range []string{"development", "production", ""} {
		for _, tc := range cases {
			t.Run(environment+"/"+tc.name, func(t *testing.T) {
				cfg := validStartupConfig(t)
				t.Setenv("APP_ENV", environment)
				tc.change(&cfg)
				err := validateRuntimeConfiguration(cfg)
				if tc.want == "" && err != nil {
					t.Fatalf("valid configuration rejected: %v", err)
				}
				if tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
					t.Fatalf("expected %s validation error, got %v", tc.want, err)
				}
			})
		}
	}
}

func TestStartupUsesEffectiveDatabasePassword(t *testing.T) {
	cfg := validStartupConfig(t)
	cfg.DatabaseURL = "postgres://fixture@127.0.0.1/fixture?sslmode=disable"
	t.Setenv("PGPASSWORD", "literal-'$@:/?#-fixture")
	if err := validateRuntimeConfiguration(cfg); err != nil {
		t.Fatal(err)
	}
	// pgx treats an empty URL password as absent, so PGPASSWORD still applies.
	cfg.DatabaseURL = "postgres://fixture:@127.0.0.1/fixture?sslmode=disable"
	if err := validateRuntimeConfiguration(cfg); err != nil {
		t.Fatal(err)
	}
	// An explicitly empty keyword password does override the environment.
	cfg.DatabaseURL = "host=127.0.0.1 user=fixture password='' dbname=fixture sslmode=disable"
	if err := validateRuntimeConfiguration(cfg); err == nil || !strings.Contains(err.Error(), "POSTGRES_PASSWORD") {
		t.Fatalf("empty effective password accepted: %v", err)
	}
	// URL credentials continue working without a duplicate environment secret.
	t.Setenv("PGPASSWORD", "")
	cfg.DatabaseURL = "postgres://fixture:url-fixture@127.0.0.1/fixture?sslmode=disable"
	if err := validateRuntimeConfiguration(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestStartupRedactsInvalidConnectionConfiguration(t *testing.T) {
	cfg := validStartupConfig(t)
	cfg.DatabaseURL = "postgres://fixture:must-not-be-logged@localhost:invalid/fixture"
	cfg.JWTSecret = "private-jwt"
	cfg.AdminSecret = "private-admin"
	cfg.InitialAdminPass = "private-pass"
	err := validateRuntimeConfiguration(cfg)
	if err == nil {
		t.Fatal("invalid configuration accepted")
	}
	for _, secret := range []string{cfg.DatabaseURL, "must-not-be-logged", cfg.JWTSecret, cfg.AdminSecret, cfg.InitialAdminPass} {
		if strings.Contains(err.Error(), secret) {
			t.Fatal("validation leaked a supplied value")
		}
	}
	for _, key := range []string{"DATABASE_URL", "JWT_SECRET", "ADMIN_SECRET", "INITIAL_ADMIN_PASSWORD"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("missing failure for %s", key)
		}
	}
}

func TestProductionStillRequiresDatabaseTLS(t *testing.T) {
	cfg := validStartupConfig(t)
	t.Setenv("APP_ENV", "production")
	cfg.DatabaseURL = "postgres://fixture:database-fixture@127.0.0.1/fixture?sslmode=disable"
	if err := validateRuntimeConfiguration(cfg); err == nil || !strings.Contains(err.Error(), "database transport") {
		t.Fatalf("production TLS requirement lost: %v", err)
	}
}

func TestStartupLogsAllSecretFailuresBeforeDatabaseAccess(t *testing.T) {
	for _, environment := range []string{"development", "production"} {
		t.Run(environment, func(t *testing.T) {
			validStartupConfig(t)
			t.Setenv("SC_API_MAIN_TEST_ROLE", "serve")
			t.Setenv("APP_ENV", environment)
			t.Setenv("DATABASE_URL", "postgres://fixture@127.0.0.1:1/fixture?sslmode=disable")
			t.Setenv("JWT_SECRET", "private-jwt")
			t.Setenv("ADMIN_SECRET", "private-admin")
			t.Setenv("INITIAL_ADMIN_PASSWORD", strings.Repeat("p", 73))
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAPIMainProcess$")
			output, err := command.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatal("startup validation timed out")
			}
			if err == nil {
				t.Fatal("invalid startup succeeded")
			}
			for _, key := range []string{"configuration validation failed", "POSTGRES_PASSWORD", "JWT_SECRET", "ADMIN_SECRET", "INITIAL_ADMIN_PASSWORD", "16-72", "32 bytes"} {
				if !strings.Contains(string(output), key) {
					t.Errorf("missing diagnostic %s: %s", key, output)
				}
			}
			for _, forbidden := range []string{"database connection failed", "private-jwt", "private-admin", strings.Repeat("p", 73), "postgres://"} {
				if strings.Contains(string(output), forbidden) {
					t.Fatal("startup accessed the database or leaked configuration")
				}
			}
		})
	}
}
