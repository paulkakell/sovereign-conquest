package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sovereignconquest/internal/api"
	"sovereignconquest/internal/config"
	"sovereignconquest/internal/db"
	"sovereignconquest/internal/game"
	"sovereignconquest/internal/schema"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	if err := schema.PrepareSeasonSchema(ctx, pool); err != nil {
		log.Fatalf("schema preflight failed: %v", err)
	}
	if err := schema.Ensure(ctx, pool); err != nil {
		log.Fatalf("schema ensure failed: %v", err)
	}
	if err := game.EnsureUniverse(ctx, pool, game.UniverseConfig{Seed: cfg.UniverseSeed, Sectors: cfg.UniverseSectors}); err != nil {
		log.Fatalf("universe initialization failed: %v", err)
	}
	if err := game.EnsureProtectorateSectors(ctx, pool, cfg.UniverseSeed); err != nil {
		log.Fatalf("protectorate initialization failed: %v", err)
	}
	if result, err := game.EnsureInitialAdmin(ctx, pool, cfg.InitialAdminUser, cfg.InitialAdminPass); err != nil {
		log.Printf("administrator initialization failed: %v", err)
	} else if result.Created || result.Promoted {
		log.Printf("administrator account initialized: username=%s", result.Username)
	}

	game.StartPortTicker(ctx, pool, cfg.PortTickSeconds)
	game.StartPlanetTicker(ctx, pool, cfg.PlanetTickSeconds)
	game.StartEventTicker(ctx, pool, cfg.EventTickSeconds)
	game.StartProtectorateTicker(ctx, pool, cfg.ProtectorateTickSeconds)

	baseHandler := (&api.Server{Cfg: cfg, Pool: pool}).Router()
	adminHandler := api.RequireAdministrator(cfg, pool, baseHandler)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.HardenHTTP(pool, adminHandler),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownContext)
	cancel()
}
