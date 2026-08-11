package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"go-ledger/internal/api"
	"go-ledger/internal/config"
	"go-ledger/internal/db"
	"go-ledger/internal/demo"
	"go-ledger/internal/ops"
	"go-ledger/internal/seed"
	"go-ledger/internal/spa"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}

// run dispatches on the subcommand, if any. Every path returns an error rather
// than calling log.Fatalf, so the deferred cleanup in each actually runs.
func run(args []string) error {
	// The healthcheck runs before configuration is even loaded: the container
	// healthcheck invokes it, the distroless runtime image has no shell to curl
	// with, and it must not depend on a valid DATABASE_URL to report liveness.
	if len(args) > 0 && args[0] == "healthcheck" {
		return runHealthcheck()
	}

	// A missing .env is normal in a container, where configuration arrives as
	// real environment variables. Only report it.
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file loaded (%v); falling back to the environment", err)
	} else {
		log.Println("Successfully loaded .env file")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Validated up here, with the rest of the configuration, so a typo fails
	// before the process touches the database.
	ipMode, err := ops.ParseClientIPMode(cfg.ClientIPMode)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Matched exactly, not searched for anywhere in the argument list. The
	// previous slices.Contains meant `server not-reset-actually-seed` reseeded
	// the database.
	var command string
	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case "", "serve", "reset", "seed":
	default:
		return fmt.Errorf("unknown command %q (want serve, seed, reset or healthcheck)", command)
	}

	log.Println("Running migrations")
	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	connectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	pool, err := pgxpool.New(connectCtx, cfg.DatabaseURL)
	cancel()

	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	if command == "reset" {
		if err := resetLedger(pool); err != nil {
			return err
		}
	}

	if command == "seed" || cfg.SeedOnStart {
		if err := seedLedger(pool, cfg.FakeSeed); err != nil {
			return err
		}
	}

	// Explicit CLI invocations are one-shot; SEED_ON_START is not.
	if command == "reset" || command == "seed" {
		return nil
	}

	return serve(cfg, ipMode, pool)
}

func resetLedger(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("Resetting accounts and transactions tables")
	if err := seed.Reset(ctx, pool); err != nil {
		return fmt.Errorf("reset database: %w", err)
	}
	return nil
}

func seedLedger(pool *pgxpool.Pool, fakeSeed uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("Seeding database with fake data")
	if err := seed.Run(ctx, pool, fakeSeed); err != nil {
		return fmt.Errorf("seed database: %w", err)
	}
	return nil
}

func serve(cfg *config.Config, ipMode ops.ClientIPMode, pool *pgxpool.Pool) error {
	var err error

	// The demo token authorizes the trusted X-Demo-Client-IP channel. Generated
	// per process and never persisted, so only the in-process runner holds it.
	demoToken := ""
	if cfg.DemosEnabled {
		demoToken, err = randomToken()
		if err != nil {
			return err
		}
	}

	limiter := ops.NewRateLimiter(cfg.RateLimit, cfg.RateWindow, cfg.RateBlockPeriod)
	defer limiter.Close()

	hub := ops.NewHub()
	recorder := ops.NewRecorder(pool, hub)
	resolver := ops.NewResolver(ipMode, demoToken)
	guard := ops.NewSecurityGuard(limiter, recorder, resolver)
	console := ops.NewConsole(pool, hub, guard, cfg.DemosEnabled)

	// The runner targets the server's own loopback address, taken from config
	// only, never from anything derived from a request.
	var demos http.Handler
	if cfg.DemosEnabled {
		runner := demo.NewRunner("http://127.0.0.1:"+cfg.Port, demoToken, guard)
		demos = demo.NewHandler(runner).Routes()
		log.Printf("Demo scenarios enabled (%d available)", len(demo.All()))
	}

	// The recorder outlives the signal context on purpose: requests still in
	// flight during shutdown record events, and those should be persisted too.
	recorderCtx, stopRecorder := context.WithCancel(context.Background())
	go recorder.Run(recorderCtx)

	spaHandler := spa.PlaceholderHandler()
	if assets, ok := spa.Resolve(cfg.SPADir); ok {
		spaHandler = spa.Handler(assets)
		log.Println("Serving the React console")
	}

	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: api.NewRouter(api.Deps{
			Cfg:     cfg,
			Pool:    pool,
			Guard:   guard,
			Console: console,
			Demos:   demos,
			SPA:     spaHandler,
		}),

		// http.Server defaults every timeout to zero, meaning none: a connection
		// dribbling one header byte per minute is held open indefinitely. That is
		// Slowloris, and it walks straight past a per-request rate limiter
		// because it never completes a request to be counted.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,

		// Safe for the open-ended SSE stream: every frame goes out through
		// writeRaw in internal/ops/stream.go, which sets a per-connection write
		// deadline, and that overrides the server-wide one.
		WriteTimeout: 30 * time.Second,

		// Between requests on a keep-alive connection only, so a stream that is
		// mid-response is unaffected.
		IdleTimeout: 60 * time.Second,

		// 64 KiB against a 1 MiB default; X-Forwarded-For is attacker-controlled
		// by design here.
		MaxHeaderBytes: 1 << 16,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Buffered so the goroutine can report and exit even though the select
	// below may already have been woken by a signal instead.
	serveErr := make(chan error, 1)

	go func() {
		log.Printf("Listening on :%s (client IP mode: %s, rate limit: %d per %s)",
			cfg.Port, ipMode, cfg.RateLimit, cfg.RateWindow)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	// A signal and a failed listen leave by the same door, so a server that
	// cannot bind still drains its queued events. Not log.Fatalf in the
	// goroutine: os.Exit from a non-main goroutine skips every defer here and
	// the drain below.
	var listenErr error
	select {
	case <-ctx.Done():
	case listenErr = <-serveErr:
		log.Printf("Server error: %v", listenErr)
	}
	stop()

	log.Println("Shutting down server")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Only now that no more requests can arrive: drain the event queue.
	stopRecorder()
	select {
	case <-recorder.Done():
	case <-time.After(5 * time.Second):
		log.Println("Timed out flushing security events")
	}

	// Reported only now, so the process still exits non-zero for a supervisor or
	// a compose restart policy, but does so after the drain rather than instead
	// of it.
	return listenErr
}

func runHealthcheck() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: /health returned %d", resp.StatusCode)
	}
	return nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate demo token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
