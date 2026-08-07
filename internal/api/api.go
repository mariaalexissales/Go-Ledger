package api

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	DB *pgxpool.Pool
}


func (a *API) withTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := a.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err 
	}

	// Commit if no errors occurred
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}