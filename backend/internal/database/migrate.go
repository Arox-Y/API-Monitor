package database

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // registers the "postgres://" scheme
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/ariyam/api-monitor/migrations"
)

// RunMigrations applies every pending migration, in order, using the SQL
// files embedded into the binary (see migrations/embed.go).
//
// WHY THIS IS STILL "AN EXPLICIT STEP", NOT AutoMigrate (ADR-003).
//
// ADR-003 rejected GORM's AutoMigrate because it derives schema by guessing
// from Go structs at runtime - it cannot express the partial unique index
// idx_incidents_one_open this schema depends on, and it gives no reviewable
// diff or way back. This function does something different: it runs
// hand-written, version-numbered SQL files that are checked into the repo
// and reviewed like any other code change. Running that automatically at
// startup is still explicit and versioned - it is simply no longer a manual
// psql command a developer has to remember to type.
//
// golang-migrate keeps its own bookkeeping table (schema_migrations) inside
// the target database, recording which version was last applied. Calling
// this on a database that is already up to date is a safe no-op.
func RunMigrations(databaseURL string) error {
	// iofs adapts an fs.FS (our embedded migrations.FS) into the "source"
	// abstraction golang-migrate reads migration files through. The "."
	// means "the root of that filesystem" - migrations.FS already contains
	// only the *.sql files, nothing else.
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("reading embedded migrations: %w", err)
	}

	// NewWithSourceInstance parses databaseURL's scheme ("postgres://") and
	// looks it up among the drivers registered by the blank import above.
	// It opens its OWN connection - separate from the GORM pool used for
	// everything else - which keeps this concern independent of GORM.
	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, databaseURL)
	if err != nil {
		return fmt.Errorf("initialising migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("applying migrations: %w", err)
	}

	return nil
}
