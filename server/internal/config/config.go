package config

import (
	"os"
	"strconv"
)

const (
	AppName = "Sovereign Conquest"
	Version = "01.05.04"
)

type Config struct {
	DatabaseURL             string
	JWTSecret               string
	AdminSecret             string
	InitialAdminUser        string
	InitialAdminPass        string
	UniverseSeed            int64
	UniverseSectors         int
	TurnRegenSeconds        int
	PortTickSeconds         int
	PlanetTickSeconds       int
	EventTickSeconds        int
	ProtectorateTickSeconds int
	HTTPAddr                string
}

func Load() Config {
	return Config{
		DatabaseURL:             env("DATABASE_URL", "postgres://sovereign:sovereign@db:5432/sovereign_conquest?sslmode=disable"),
		JWTSecret:               env("JWT_SECRET", "dev-secret-change-me"),
		AdminSecret:             env("ADMIN_SECRET", ""),
		InitialAdminUser:        env("INITIAL_ADMIN_USERNAME", "admin"),
		InitialAdminPass:        env("INITIAL_ADMIN_PASSWORD", "ChangeMeNow!"),
		UniverseSeed:            envInt64("UNIVERSE_SEED", 2002),
		UniverseSectors:         envInt("UNIVERSE_SECTORS", 200),
		TurnRegenSeconds:        envInt("TURN_REGEN_SECONDS", 120),
		PortTickSeconds:         envInt("PORT_TICK_SECONDS", 60),
		PlanetTickSeconds:       envInt("PLANET_TICK_SECONDS", 60),
		EventTickSeconds:        envInt("EVENT_TICK_SECONDS", 60),
		ProtectorateTickSeconds: envInt("PROTECTORATE_TICK_SECONDS", 60),
		HTTPAddr:                env("HTTP_ADDR", ":8080"),
	}
}

func env(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func envInt64(key string, def int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return i
}
