// Package database opens and manages the connection to PostgreSQL.
package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens a connection to PostgreSQL and checks that it actually works.
//
// It returns a *gorm.DB. The * means "a pointer": rather than copying the whole
// connection object around, we pass its address. Everyone then shares the one
// real connection instead of each getting their own copy.
func Connect(databaseURL string, isDevelopment bool) (*gorm.DB, error) {
	// In development we print every SQL statement to the terminal, which is
	// great for learning — you can see exactly what GORM sends to Postgres.
	// In production that would be noisy and could leak data into logs.
	logLevel := logger.Silent
	if isDevelopment {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		// %w wraps the original error inside ours. It keeps the underlying
		// cause attached, so nothing is lost, while adding context about
		// WHERE it happened. Using %v instead would flatten it to plain text
		// and throw that detail away.
		return nil, fmt.Errorf("opening database connection: %w", err)
	}

	// GORM sits on top of Go's built-in database tools. We reach through to
	// those to configure the connection pool.
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("accessing underlying sql.DB: %w", err)
	}

	// ---- Connection pool -------------------------------------------------
	//
	// Opening a database connection is slow (a network round trip plus login).
	// Doing it per request would be wasteful, so Go keeps a POOL of open
	// connections and hands them out as needed.
	//
	// Think of a small team of workers: a request borrows one, uses it,
	// returns it. These numbers control the size of that team.
	sqlDB.SetMaxOpenConns(25)                 // never more than 25 at once
	sqlDB.SetMaxIdleConns(5)                  // keep 5 warm and ready
	sqlDB.SetConnMaxLifetime(5 * time.Minute) // recycle after 5 minutes

	// Opening a connection does not prove the database is reachable — GORM is
	// lazy and may not have contacted the server yet. Ping forces a real
	// round trip, so we find out NOW rather than on a user's first request.
	//
	// context.WithTimeout says "give up after 5 seconds". Without it, a
	// network problem could hang the program forever at startup.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer means "run this when the surrounding function finishes, no matter
	// how it finishes". Forgetting cancel() leaks a timer, and it is one of
	// the most common mistakes in Go.
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return db, nil
}

// Ping checks the database is still reachable. The /healthz endpoint uses it.
func Ping(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("accessing underlying sql.DB: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

// Close shuts the connection pool down cleanly when the program exits.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("accessing underlying sql.DB: %w", err)
	}
	return sqlDB.Close()
}
