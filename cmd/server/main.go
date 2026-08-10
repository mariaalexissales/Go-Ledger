package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"go-ledger/internal/api"
	"go-ledger/internal/config"
	"go-ledger/internal/demo"
	"go-ledger/internal/ops"
	"go-ledger/internal/seed"
	"go-ledger/internal/server"
	"go-ledger/internal/spa"
)

func main() {
	args := os.Args[1:]

	// Runs before anything else: the container healthcheck uses this, and the
	// distroless runtime image has no shell to curl with.
	if slices.Contains(args, "healthcheck") {
		runHealthcheck()
		return
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
		log.Fatalf("Invalid configuration: %v", err)
	}

	ipMode, err := ops.ParseClientIPMode(cfg.ClientIPMode)
	if err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Println("Running migrations")
	if err := server.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	connectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	pool, err := server.Connect(connectCtx, cfg.DatabaseURL)
	cancel()

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	wantsReset := slices.Contains(args, "reset")
	wantsSeed := slices.Contains(args, "seed")

	if wantsReset {
		resetCtx, resetCancel := context.WithTimeout(context.Background(), 10*time.Second)

		log.Println("Resetting accounts and transactions tables")
		if err := seed.Reset(resetCtx, pool); err != nil {
			resetCancel()
			log.Fatalf("Failed to reset database: %v", err)
		}

		resetCancel()
	}

	if wantsSeed || cfg.SeedOnStart {
		seedCtx, seedCancel := context.WithTimeout(context.Background(), 30*time.Second)

		log.Println("Seeding database with fake data")
		if err := seed.Run(seedCtx, pool, cfg.FakeSeed); err != nil {
			seedCancel()
			log.Fatalf("Failed to seed database: %v", err)
		}

		seedCancel()
	}

	// Explicit CLI invocations are one-shot; SEED_ON_START is not.
	if wantsReset || wantsSeed {
		return
	}

	// The demo token authorizes the trusted X-Demo-Client-IP channel. Generated
	// per process and never persisted, so only the in-process runner holds it.
	demoToken := ""
	if cfg.DemosEnabled {
		demoToken = randomToken()
	}

	limiter := ops.NewRateLimiter(cfg.RateLimit, cfg.RateWindow, cfg.RateBlockPeriod)
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
		log.Printf("Demo scenarios enabled (%d registered)", len(demo.All()))
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
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Listening on :%s (client IP mode: %s, rate limit: %d per %s)",
			cfg.Port, ipMode, cfg.RateLimit, cfg.RateWindow)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()
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
}

func runHealthcheck() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}

func randomToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		log.Fatalf("Failed to generate demo token: %v", err)
	}
	return hex.EncodeToString(buf)
}
