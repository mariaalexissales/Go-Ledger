// Package db owns schema setup and the shared row-collecting query helper.
package db

import (
	"context"
	"errors"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"

	"go-ledger/internal/db/migrations"
)

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Collect runs a query and gathers every row through scan. RowToStructByPos is
// unusable here: paginated queries select COUNT(*) OVER() as a trailing column,
// so the row is one column wider than the struct.
func Collect[T any](ctx context.Context, q Querier, sql string, scan func(pgx.CollectableRow) (T, error), args ...any) ([]T, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, scan)
}

func RunMigrations(connStr string) error {
	u, err := url.Parse(connStr)
	if err != nil {
		return err
	}
	u.Scheme = "pgx5"

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, u.String())
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
