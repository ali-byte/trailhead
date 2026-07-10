// Package postgres implements internal/adapter.BookmarkRepository against a
// real Postgres database using github.com/jackc/pgx/v5 directly (not
// database/sql + lib/pq) — see ARCHITECTURE_RFC.md "Postgres Driver".
// RunMigrations is the one exception: golang-migrate operates over
// database/sql, so migrations run through the pgx/v5 stdlib compatibility
// driver rather than the pgxpool used by Repository itself.
package postgres

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations applies every pending migration in migrations/ to the
// database at dsn. A no-op run (schema already current) is treated as
// success, not an error — the go:embed glob above must resolve to the
// real migration files, or m.Up() on an empty source also reports
// migrate.ErrNoChange while creating nothing
// (TestRunMigrations_FreshDB_CreatesSchema guards this).
func RunMigrations(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("RunMigrations: open db: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	driver, err := migratepostgres.WithInstance(db, &migratepostgres.Config{})
	if err != nil {
		return fmt.Errorf("RunMigrations: new postgres driver: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("RunMigrations: load embedded migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "pgx", driver)
	if err != nil {
		return fmt.Errorf("RunMigrations: new migrate instance: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("RunMigrations: apply migrations: %w", err)
	}

	return nil
}
