// Command api serves the ShadowCommit HTTP API.
package main
import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/cavelasquez17/shadowcommit/api/internal/auth"
	"github.com/cavelasquez17/shadowcommit/api/internal/config"
	"github.com/cavelasquez17/shadowcommit/api/internal/handlers"
	"github.com/cavelasquez17/shadowcommit/api/internal/seed"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
)

func main() {
	// Load .env if present — silently ignored if not.
	_ = godotenv.Load()

	withDemo := flag.Bool("seed-demo", false, "seed demo users and threads (no-op if users exist)")
	flag.Parse()

	cfg := config.Load()

	db, err := store.Open(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	if err := seed.Reference(db); err != nil {
		log.Fatalf("seed reference data: %v", err)
	}
	if *withDemo || os.Getenv("SEED_DEMO") == "true" {
		if err := seed.Demo(db); err != nil {
			log.Fatalf("seed demo data: %v", err)
		}
	}

	iss := auth.NewIssuer(cfg)
	gh := auth.NewGitHub(cfg)
	if gh == nil {
		log.Println("warning: GitHub OAuth not configured; only dev login is available")
	}

	h := handlers.New(db, cfg, iss, gh)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("shadowcommit api listening on :%s (env=%s driver=%s)", cfg.Port, cfg.Env, cfg.DBDriver)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	h.Close()
	log.Println("api stopped. no severance was paid.")
}
