package main

import (
	"errors"
	"net/url"
	"os"
	"strings"

	"sovereignconquest/internal/config"
)

func validateRuntimeConfiguration(cfg config.Config) error {
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return errors.New("database connection is required")
	}
	if cfg.UniverseSectors < 2 || cfg.UniverseSectors > 1_000_000 {
		return errors.New("universe sector count is outside the supported range")
	}
	if cfg.TurnRegenSeconds < 10 {
		return errors.New("turn regeneration interval must be at least 10 seconds")
	}
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		return nil
	}

	if len(strings.TrimSpace(cfg.JWTSecret)) < 32 {
		return errors.New("session signing material must contain at least 32 characters")
	}
	if len(strings.TrimSpace(cfg.InitialAdminPass)) < 16 {
		return errors.New("bootstrap administrator value must contain at least 16 characters")
	}
	if cfg.AdminSecret != "" && len(strings.TrimSpace(cfg.AdminSecret)) < 32 {
		return errors.New("administration key must contain at least 32 characters")
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
