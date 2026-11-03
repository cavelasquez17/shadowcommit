// Command api serves the ShadowCommit HTTP API.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
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
	"github.com/cavelasquez17/shadowcommit/api/internal/jobs"
	"github.com/cavelasquez17/shadowcommit/api/internal/seed"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
)

func main() {
	// Load .env if present — silently ignored if not.
	_ = godotenv.Load()

	withDemo := flag.Bool("seed-demo", false, "seed demo users and threads (no-op if users exist)")
	healthcheck := flag.Bool("healthcheck", false, "probe /healthz on the local port and exit 0 if healthy (for container HEALTHCHECK)")
	flag.Parse()

	if *healthcheck {
		if err := runHealthcheck("http://127.0.0.1:"+config.Port()+"/healthz", 3*time.Second); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	cfg := config.Load()

	db, err := store.Open(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	if err := seed.Reference(db); err != nil {
		log.Fatalf("seed reference data: %v", err)
	}
	if err := seed.MigrateAvatars(db); err != nil {
		log.Fatalf("migrate avatars: %v", err)
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

	// Recurring work (server local time). Set JOBS_ENABLED=false on all but one replica.
	var sched *jobs.Scheduler
	if os.Getenv("JOBS_ENABLED") != "false" {
		sched = jobs.New(nil,
			jobs.Job{Name: "refresh-hot-scores", Schedule: jobs.Daily(3),
				Run: func(ctx context.Context, now time.Time) error {
					n, err := jobs.RefreshHotScores(ctx, db, now)
					log.Printf("refresh-hot-scores: %d threads", n)
					return err
				}},
			jobs.Job{Name: "weekly-digest", Schedule: jobs.Weekly(time.Monday, 9),
				Run: func(ctx context.Context, now time.Time) error {
					n, err := jobs.SendWeeklyDigest(ctx, db, now)
					log.Printf("weekly-digest: %d notifications", n)
					return err
				}},
		)
		sched.Start()
	}
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
	if sched != nil {
		sched.Stop()
	}
	h.Close()
	log.Println("api stopped. no severance was paid.")
}
