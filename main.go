package main

import (
	"context"
	"log"
	"os"
	"slices"
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

	// TODO: Layer 2 - wire api.NewRouter(pool) up to a net/http listener
	api.NewRouter(pool)
}
