package server

import (
	"context"
	"errors"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, connStr)
}

func RunMigrations(connStr string) error {
	migrateURL := strings.Replace(connStr, "postgres://", "pgx5://", 1)

	m, err := migrate.New("file://internal/db/migrations", migrateURL)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
