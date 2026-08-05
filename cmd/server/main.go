package main

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Account struct {
	Name      string
	Balance   float32
	CreatedAt time.Time
}

type Transactions struct {
	AccountId	int
	Amount		float32
	CreatedAt	time.Time
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	} else {
		log.Println("Successfully loaded .env file")
	}

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("Running migrations")

	migrateURL := strings.Replace(connStr, "postgres://", "pgx5://", 1)

	m, err := migrate.New("file://internal/db/migrations", migrateURL)

	if err != nil {
		log.Fatalf("Failed to initialize migrations: %v", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Creating database")

	conn, err := pgxpool.New(ctx, connStr)

	if err != nil {
		log.Fatalf("Failed to spin up Layer 1: %v", err)
	}

	log.Println("Populating accounts with fake data")

	accountIDs := make([]int, 0, 50)

	for range 50 {
		name := gofakeit.Name()
		createdAt := gofakeit.Date()

		var id int
		err := conn.QueryRow(ctx,
			"INSERT INTO accounts (name, created_at) VALUES ($1, $2) RETURNING id",
			name, createdAt,
		).Scan(&id)

		if err != nil {
			log.Fatalf("Failed to insert fake data into accounts table: %v", err)
		}

		accountIDs = append(accountIDs, id)
	}

	log.Println("Populating transactions with fake data")

	for range 200 {
		accountID := accountIDs[gofakeit.Number(0, len(accountIDs)-1)]
		amount := gofakeit.Float32Range(-500, 500)
		createdAt := gofakeit.Date()

		_, err := conn.Exec(ctx,
			"INSERT INTO transactions (account_id, amount, timestamp) VALUES ($1, $2, $3)",
			accountID, amount, createdAt,
		)

		if err != nil {
			log.Fatalf("Failed to insert fake data into transactions table: %v", err)
		}
	}

	log.Println("Calculating account balances from transactions")

	_, err = conn.Exec(ctx, `
		UPDATE accounts
		SET balance = sub.total
		FROM (
			SELECT account_id, SUM(amount) AS total
			FROM transactions
			GROUP BY account_id
		) sub
		WHERE accounts.id = sub.account_id
	`)

	if err != nil {
		log.Fatalf("Failed to calculate account balances: %v", err)
	}

	conn.Close()
	log.Println("Layer 1 initialized successfully. Database spun.")
}
