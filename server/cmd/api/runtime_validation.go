package main

import (
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"sovereignconquest/internal/auth"
	"sovereignconquest/internal/config"
)

func validateRuntimeConfiguration(cfg config.Config) error {
	// Validate the effective database credential, including URL, PGPASSWORD,
	// service-file and password-file precedence, exactly as db.Connect does.
	// Parser errors can contain credentials, so never include them in logs.
	var problems []string
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		problems = append(problems, "DATABASE_URL: database connection is required")
	} else if connection, err := pgxpool.ParseConfig(cfg.DatabaseURL); err != nil {
		problems = append(problems, "DATABASE_URL: invalid PostgreSQL connection configuration; check its format and referenced files")
	} else if connection.ConnConfig.Password == "" {
		problems = append(problems, "POSTGRES_PASSWORD: database password must not be empty (supply it to the API through PGPASSWORD or DATABASE_URL)")
	}
	if len(strings.TrimSpace(cfg.JWTSecret)) < 32 {
		problems = append(problems, "JWT_SECRET: must contain at least 32 bytes after trimming surrounding whitespace (32 ASCII characters)")
	}
	if cfg.AdminSecret != "" && len(strings.TrimSpace(cfg.AdminSecret)) < 32 {
		problems = append(problems, "ADMIN_SECRET: must be empty to disable season resets, or contain at least 32 bytes after trimming surrounding whitespace (32 ASCII characters)")
	}
	passwordBytes := len(strings.TrimSpace(cfg.InitialAdminPass))
	if passwordBytes < 16 || passwordBytes > auth.MaxPasswordBytes {
		problems = append(problems, "INITIAL_ADMIN_PASSWORD: must contain 16-72 bytes after trimming surrounding whitespace (16-72 ASCII characters; bcrypt limit)")
	}
	if cfg.UniverseSectors < 2 || cfg.UniverseSectors > 1_000_000 {
		problems = append(problems, "UNIVERSE_SECTORS: universe sector count is outside the supported range")
	}
	if cfg.TurnRegenSeconds < 10 {
		problems = append(problems, "TURN_REGEN_SECONDS: turn regeneration interval must be at least 10 seconds")
	}
	if len(problems) != 0 {
		// One line reports every problem without values, hashes or connection URLs.
		return errors.New(strings.Join(problems, "; "))
	}
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		return nil
	}

	parsed, err := url.Parse(cfg.DatabaseURL)
	if err != nil {
		return errors.New("database connection is invalid")
	}
	switch strings.ToLower(parsed.Query().Get("sslmode")) {
	case "verify-full", "verify-ca", "require":
		return nil
	default:
		return errors.New("database transport verification is required in production")
	}
}
