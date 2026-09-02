// Package testutil provides shared helpers for integration tests.
//
// It lives in internal/ like everything else, so it can never be imported by
// another module — test helpers are not part of the public surface.
package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ariyam/api-monitor/internal/models"
)

// defaultTestDSN points at the SEPARATE test database.
//
// Note the database name: apimonitor_test, not apimonitor. Integration tests
// truncate tables between cases, so pointing them at the development database
// would wipe real data. Using a different database makes that mistake
// impossible rather than merely unlikely.
const defaultTestDSN = "postgres://apimonitor:devpassword@localhost:5433/apimonitor_test?sslmode=disable"

// DB connects to the test database, or SKIPS the test if it is unavailable.
//
// Skipping rather than failing is deliberate. Someone running `go test ./...`
// without Docker should see integration tests skipped, not a wall of red that
// hides a genuine unit-test failure.
func DB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// Silent, or every test prints dozens of SQL statements and the actual
		// failure scrolls off the screen.
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("integration test skipped: cannot reach the test database (%v)", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("integration test skipped: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		t.Skipf("integration test skipped: test database is not responding (%v)", err)
	}

	// Start every test from a known-empty database.
	//
	// Tests that depend on leftovers from an earlier test pass or fail
	// depending on the order they run in, which is the most miserable kind of
	// flake to debug.
	Truncate(t, db)

	// t.Cleanup runs when this test finishes, however it finishes. Leaving the
	// database dirty would break whichever test runs next.
	t.Cleanup(func() {
		Truncate(t, db)
		_ = sqlDB.Close()
	})

	return db
}

// Truncate empties every table.
//
// CASCADE follows the foreign keys, so deleting users clears everything below
// them. RESTART IDENTITY resets any sequences.
func Truncate(t *testing.T, db *gorm.DB) {
	t.Helper()

	err := db.Exec(`TRUNCATE users, projects, monitors, checks, incidents
	                RESTART IDENTITY CASCADE`).Error
	if err != nil {
		t.Fatalf("truncating test database: %v", err)
	}
}

// CreateUser inserts a user and returns it.
//
// Test helpers take *testing.T and fail the test themselves, so callers are
// not littered with error checks that obscure what is actually being tested.
func CreateUser(t *testing.T, db *gorm.DB, email string) models.User {
	t.Helper()

	user := models.User{
		Email:        email,
		Name:         "Test User",
		PasswordHash: "$2a$12$notarealhashbutlongenoughtolooklikeone123456789012345678",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("creating test user %s: %v", email, err)
	}
	return user
}

// CreateProject inserts a project owned by userID.
func CreateProject(t *testing.T, db *gorm.DB, userID uuid.UUID, name, slug string) models.Project {
	t.Helper()

	project := models.Project{
		UserID: userID,
		Name:   name,
		Slug:   slug,
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("creating test project %s: %v", name, err)
	}
	return project
}

// CreateMonitor inserts a monitor inside a project.
func CreateMonitor(t *testing.T, db *gorm.DB, projectID uuid.UUID, name string) models.Monitor {
	t.Helper()

	monitor := models.Monitor{
		ProjectID:       projectID,
		Name:            name,
		URL:             "https://example.com",
		Method:          "GET",
		IntervalSeconds: 60,
		TimeoutSeconds:  10,
		ExpectedStatus:  200,
		Enabled:         true,
		Status:          models.StatusPending,
	}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("creating test monitor %s: %v", name, err)
	}
	return monitor
}
