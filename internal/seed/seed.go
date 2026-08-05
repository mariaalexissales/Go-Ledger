package seed

import (
	"context"
	"fmt"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/jackc/pgx/v5/pgxpool"
)


func Reset(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, "TRUNCATE accounts, transactions RESTART IDENTITY CASCADE"); err != nil {
		return fmt.Errorf("failed to reset accounts/transactions: %w", err)
	}

	return nil
}


func Run(ctx context.Context, pool *pgxpool.Pool) error {
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM accounts").Scan(&count); err != nil {
		return fmt.Errorf("failed to check existing account count: %w", err)
	}

	if count > 0 {
		return nil
	}

	accountIDs := make([]int, 0, 50)

	for range 50 {
		name := gofakeit.Name()
		createdAt := gofakeit.Date()

		var id int
		err := pool.QueryRow(ctx,
			"INSERT INTO accounts (name, created_at) VALUES ($1, $2) RETURNING id",
			name, createdAt,
		).Scan(&id)

		if err != nil {
			return fmt.Errorf("failed to insert account: %w", err)
		}

		accountIDs = append(accountIDs, id)
	}

	for range 200 {
		accountID := accountIDs[gofakeit.Number(0, len(accountIDs)-1)]
		amount := gofakeit.Float32Range(-500, 500)
		createdAt := gofakeit.Date()

		_, err := pool.Exec(ctx,
			"INSERT INTO transactions (account_id, amount, timestamp) VALUES ($1, $2, $3)",
			accountID, amount, createdAt,
		)

		if err != nil {
			return fmt.Errorf("failed to insert transaction: %w", err)
		}
	}

	_, err := pool.Exec(ctx, `
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
		return fmt.Errorf("failed to calculate account balances: %w", err)
	}

	return nil
}
