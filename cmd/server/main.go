package main

import (
	"context"
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
	"go-ledger/internal/seed"
	"go-ledger/internal/server"
)

func main() {
	args := os.Args[1:]

	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	} else {
		log.Println("Successfully loaded .env file")
	}

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	log.Println("Running migrations")
	if err := server.RunMigrations(connStr); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	connectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	pool, err := server.Connect(connectCtx, connStr)
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

	if wantsSeed {
		seedCtx, seedCancel := context.WithTimeout(context.Background(), 30*time.Second)

		log.Println("Seeding database with fake data")
		if err := seed.Run(seedCtx, pool); err != nil {
			seedCancel()
			log.Fatalf("Failed to seed database: %v", err)
		}

		seedCancel()
	}

	if wantsReset || wantsSeed {
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: api.NewRouter(pool),
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Listening on :%s", port)
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
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}
