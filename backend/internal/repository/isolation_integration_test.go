//go:build integration

// The build tag above means these tests only run with:
//
//	go test -tags=integration ./...
//
// A plain `go test ./...` skips this file entirely, so the everyday test run
// stays fast and needs no database. Slow tests that are always run get
// disabled by frustrated developers; slow tests that are opt-in get kept.

package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/ariyam/api-monitor/internal/apperr"
	"github.com/ariyam/api-monitor/internal/repository"
	"github.com/ariyam/api-monitor/internal/testutil"
)

// THE MOST IMPORTANT TEST IN THIS PROJECT.
//
// It guards the boundary between "a multi-tenant application" and "a data
// breach". Every query that touches user-owned data must be scoped by the
// user, and this proves it against a real database rather than a mock.
//
// It was verified by hand on Day 2. Hand-verification proves it worked once,
// while someone was watching. This proves it on every run, forever - which is
// what stops a refactor six weeks from now from quietly dropping a WHERE
// clause and exposing one customer's data to another.
func TestCrossUserIsolation(t *testing.T) {
	db := testutil.DB(t)

	projects := repository.NewProjectRepository(db)
	monitors := repository.NewMonitorRepository(db)
	checks := repository.NewCheckRepository(db)
	incidents := repository.NewIncidentRepository(db)

	ctx := context.Background()

	// Two separate accounts. Alice owns everything; Bob owns nothing.
	alice := testutil.CreateUser(t, db, "alice@isolation.test")
	bob := testutil.CreateUser(t, db, "bob@isolation.test")

	aliceProject := testutil.CreateProject(t, db, alice.ID, "Alice Project", "alice-project")
	aliceMonitor := testutil.CreateMonitor(t, db, aliceProject.ID, "Alice Monitor")

	t.Run("bob cannot read alice's project", func(t *testing.T) {
		_, err := projects.GetByIDForUser(ctx, aliceProject.ID, bob.ID)

		// It must be ErrNotFound, NOT a permission error. A "forbidden"
		// response would confirm the id is real, letting an attacker walk
		// through ids and map out the customer base.
		if !errors.Is(err, apperr.ErrNotFound) {
			t.Fatalf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("alice CAN read her own project", func(t *testing.T) {
		// The control case. Without it, a repository that returned ErrNotFound
		// for absolutely everything would pass every other test here.
		got, err := projects.GetByIDForUser(ctx, aliceProject.ID, alice.ID)
		if err != nil {
			t.Fatalf("alice could not read her own project: %v", err)
		}
		if got.ID != aliceProject.ID {
			t.Errorf("got project %s, want %s", got.ID, aliceProject.ID)
		}
	})

	t.Run("bob cannot update alice's project", func(t *testing.T) {
		aliceProject.Name = "Hacked"
		err := projects.Update(ctx, &aliceProject, bob.ID)
		if !errors.Is(err, apperr.ErrNotFound) {
			t.Fatalf("got %v, want ErrNotFound", err)
		}

		// And the name in the database must be unchanged.
		fresh, err := projects.GetByIDForUser(ctx, aliceProject.ID, alice.ID)
		if err != nil {
			t.Fatalf("reloading project: %v", err)
		}
		if fresh.Name == "Hacked" {
			t.Fatal("SECURITY FAILURE: bob modified alice's project")
		}
	})

	t.Run("bob cannot delete alice's project", func(t *testing.T) {
		err := projects.Delete(ctx, aliceProject.ID, bob.ID)
		if !errors.Is(err, apperr.ErrNotFound) {
			t.Fatalf("got %v, want ErrNotFound", err)
		}

		if _, err := projects.GetByIDForUser(ctx, aliceProject.ID, alice.ID); err != nil {
			t.Fatal("SECURITY FAILURE: bob deleted alice's project")
		}
	})

	t.Run("bob cannot read alice's monitor", func(t *testing.T) {
		_, err := monitors.GetByIDForUser(ctx, aliceMonitor.ID, bob.ID)
		if !errors.Is(err, apperr.ErrNotFound) {
			t.Fatalf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("bob cannot update alice's monitor", func(t *testing.T) {
		aliceMonitor.Name = "Hacked"
		err := monitors.Update(ctx, &aliceMonitor, bob.ID)
		if !errors.Is(err, apperr.ErrNotFound) {
			t.Fatalf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("bob cannot delete alice's monitor", func(t *testing.T) {
		err := monitors.Delete(ctx, aliceMonitor.ID, bob.ID)
		if !errors.Is(err, apperr.ErrNotFound) {
			t.Fatalf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("bob's listings are empty, not alice's data", func(t *testing.T) {
		// A subtle failure mode: a query might correctly refuse single lookups
		// while a LIST query forgets the filter and returns everything.
		bobProjects, err := projects.ListByUser(ctx, bob.ID)
		if err != nil {
			t.Fatalf("listing bob's projects: %v", err)
		}
		if len(bobProjects) != 0 {
			t.Errorf("SECURITY FAILURE: bob sees %d projects, want 0", len(bobProjects))
		}

		bobMonitors, err := monitors.ListByUser(ctx, bob.ID)
		if err != nil {
			t.Fatalf("listing bob's monitors: %v", err)
		}
		if len(bobMonitors) != 0 {
			t.Errorf("SECURITY FAILURE: bob sees %d monitors, want 0", len(bobMonitors))
		}
	})

	t.Run("bob cannot list monitors inside alice's project", func(t *testing.T) {
		got, err := monitors.ListByProject(ctx, aliceProject.ID, bob.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("SECURITY FAILURE: bob sees %d of alice's monitors", len(got))
		}
	})

	t.Run("bob cannot read alice's check history", func(t *testing.T) {
		got, err := checks.ListByMonitorForUser(ctx, aliceMonitor.ID, bob.ID, 50, timeZero())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("SECURITY FAILURE: bob sees %d of alice's checks", len(got))
		}
	})

	t.Run("bob cannot read alice's incidents", func(t *testing.T) {
		got, err := incidents.ListByMonitorForUser(ctx, aliceMonitor.ID, bob.ID, 50)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("SECURITY FAILURE: bob sees %d of alice's incidents", len(got))
		}

		recent, err := incidents.ListRecentForUser(ctx, bob.ID, 50)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(recent) != 0 {
			t.Errorf("SECURITY FAILURE: bob sees %d incidents, want 0", len(recent))
		}
	})

	t.Run("a random id belonging to nobody is also not found", func(t *testing.T) {
		// Not-yours and does-not-exist must be indistinguishable. If they
		// differed, the difference itself would reveal which ids are real.
		_, err := projects.GetByIDForUser(ctx, uuid.New(), alice.ID)
		if !errors.Is(err, apperr.ErrNotFound) {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})
}
