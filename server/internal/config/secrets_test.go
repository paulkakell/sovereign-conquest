package config

import (
	"strings"
	"testing"
)

func TestMissingSecretsHaveNoBuiltInCredentialFallback(t *testing.T) {
	for _, key := range []string{"JWT_SECRET", "INITIAL_ADMIN_PASSWORD", "ADMIN_SECRET", "DATABASE_URL"} {
		t.Setenv(key, "")
	}
	cfg := Load()
	if cfg.JWTSecret != "" || cfg.InitialAdminPass != "" || cfg.AdminSecret != "" {
		t.Fatal("missing secrets received built-in defaults")
	}
	if strings.Contains(cfg.DatabaseURL, "@") {
		t.Fatal("default database URL embeds credentials")
	}
}
