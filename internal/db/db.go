// Package db owns schema setup. The migration files themselves live in the
// migrations subpackage, which exists so //go:embed can be rooted in the
// directory holding the .sql files.
package db

import (
	"errors"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"go-ledger/internal/db/migrations"
)

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
