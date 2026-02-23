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
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	if err := schema.Ensure(ctx, pool); err != nil {
		log.Fatalf("schema ensure failed: %v", err)
	}

	if err := game.EnsureUniverse(ctx, pool, game.UniverseConfig{Seed: cfg.UniverseSeed, Sectors: cfg.UniverseSectors}); err != nil {
		log.Fatalf("universe init failed: %v", err)
	}

	if res, err := game.EnsureInitialAdmin(ctx, pool, cfg.InitialAdminUser, cfg.InitialAdminPass); err != nil {
		log.Printf("initial admin ensure failed: %v", err)
	} else if res.Created {
		log.Printf("initial admin ensured: username=%s (password change required on first login)", res.Username)
	} else if res.Promoted && res.PasswordReset {
		log.Printf("initial admin recovered: username=%s promoted to admin and password reset (password change required on first login)", res.Username)
	}

	game.StartPortTicker(ctx, pool, cfg.PortTickSeconds)
	game.StartPlanetTicker(ctx, pool, cfg.PlanetTickSeconds)
	game.StartEventTicker(ctx, pool, cfg.EventTickSeconds)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           (&api.Server{Cfg: cfg, Pool: pool}).Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
	cancel()
}
