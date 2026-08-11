// Package db owns schema setup. The migration files themselves live in the
// migrations subpackage, which exists so //go:embed can be rooted in the
// directory holding the .sql files.
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

// Querier is the query surface Collect needs. Satisfied by *pgxpool.Pool,
// pgx.Tx and *pgx.Conn alike, so a helper does not force a caller to decide
// whether it is inside a transaction.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Collect runs a query and gathers every row through scan. pgx.CollectRows
// closes the rows and folds rows.Err into its return, so callers get one error
// path instead of three.
//
// For a paginated query, select COUNT(*) OVER() as the last column and let scan
// write it into a variable captured from the caller -- see listAccounts. That is
// also why RowToStructByPos is unusable here: the row is one column wider than
// the struct.
func Collect[T any](ctx context.Context, q Querier, sql string, scan func(pgx.CollectableRow) (T, error), args ...any) ([]T, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, scan)
}

// RunMigrations applies every pending migration. Migrations are embedded rather
// than read from disk so the distroless image, which ships nothing but the
// binary, can still bring up a fresh database.
func RunMigrations(connStr string) error {
	u, err := url.Parse(connStr)
	if err != nil {
		return err
	}
	// golang-migrate registers the pgx v5 driver under its own scheme, not the
	// postgres:// that the rest of the application connects with.
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
