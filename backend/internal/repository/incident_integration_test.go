//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/ariyam/api-monitor/internal/models"
	"github.com/ariyam/api-monitor/internal/repository"
	"github.com/ariyam/api-monitor/internal/testutil"
	"github.com/ariyam/api-monitor/internal/worker"
)

func timeZero() time.Time { return time.Time{} }

// Proves the database itself refuses a second open incident.
//
// The application logic already prevents this - DecideTransition only opens an
// incident on a transition. This test checks the SAFETY NET underneath it:
//
//	CREATE UNIQUE INDEX idx_incidents_one_open
//	  ON incidents (monitor_id) WHERE resolved_at IS NULL;
//
// A partial unique index applies only to rows matching its WHERE clause, so a
// monitor may accumulate unlimited CLOSED incidents but never two open ones.
//
// This matters because application logic can be wrong. A refactor, a race, a
// second worker process - any of those could try to open a duplicate. The
// database rejecting it is what keeps the invariant true even when the code
// is not.
func TestPartialUniqueIndex_RejectsSecondOpenIncident(t *testing.T) {
	// No ctx here: this test deliberately bypasses the repository and talks to
	// the database directly, because it is testing the DATABASE CONSTRAINT
	// rather than any Go code.
	db := testutil.DB(t)

	user := testutil.CreateUser(t, db, "index@test.local")
	project := testutil.CreateProject(t, db, user.ID, "P", "index-project")
	monitor := testutil.CreateMonitor(t, db, project.ID, "M")

	// First open incident: fine.
	first := models.Incident{
		MonitorID: monitor.ID,
		StartedAt: time.Now().UTC(),
		Cause:     "first outage",
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("the FIRST open incident should be allowed: %v", err)
	}

	// Second open incident for the same monitor: must be rejected.
	second := models.Incident{
		MonitorID: monitor.ID,
		StartedAt: time.Now().UTC(),
		Cause:     "duplicate",
	}
	err := db.Create(&second).Error
	if err == nil {
		t.Fatal("DATABASE CONSTRAINT FAILED: a second open incident was accepted")
	}
	t.Logf("correctly rejected: %v", err)

	// Close the first one, and a new incident becomes allowed again - because
	// the index only constrains rows where resolved_at IS NULL.
	resolved := time.Now().UTC()
	duration := 60
	if err := db.Model(&first).Updates(map[string]any{
		"resolved_at":      resolved,
		"duration_seconds": duration,
	}).Error; err != nil {
		t.Fatalf("closing the first incident: %v", err)
	}

	third := models.Incident{
		MonitorID: monitor.ID,
		StartedAt: time.Now().UTC(),
		Cause:     "a later, separate outage",
	}
	if err := db.Create(&third).Error; err != nil {
		t.Fatalf("a new incident should be allowed once the previous one closed: %v", err)
	}

	// A DIFFERENT monitor may have its own open incident simultaneously - the
	// index is per-monitor, not global.
	other := testutil.CreateMonitor(t, db, project.ID, "Other")
	if err := db.Create(&models.Incident{
		MonitorID: other.ID,
		StartedAt: time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("a different monitor should be allowed its own open incident: %v", err)
	}
}

// Proves the whole check write is atomic.
//
// RecordResult writes up to three tables. If any part fails, NONE of it may
// land - otherwise a crash could mark a monitor down with no incident, and
// since the status would already be down, no later check would ever open one.
// The outage would become permanently invisible.
func TestRecordResult_IsAtomic(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()

	repo := repository.NewCheckRepository(db)

	user := testutil.CreateUser(t, db, "atomic@test.local")
	project := testutil.CreateProject(t, db, user.ID, "P", "atomic-project")
	monitor := testutil.CreateMonitor(t, db, project.ID, "M")

	statusCode := 503
	failure := worker.CheckResult{
		CheckedAt:      time.Now().UTC(),
		Success:        false,
		StatusCode:     &statusCode,
		ResponseTimeMs: 120,
		Error:          "expected status 200, got 503",
	}
	goingDown := worker.Transition{
		NewStatus:    models.StatusDown,
		OpenIncident: true,
	}

	if err := repo.RecordResult(ctx, monitor.ID, failure, goingDown); err != nil {
		t.Fatalf("recording a failure: %v", err)
	}

	// All three effects must be present together.
	var checkCount, incidentCount int64
	db.Model(&models.Check{}).Where("monitor_id = ?", monitor.ID).Count(&checkCount)
	db.Model(&models.Incident{}).Where("monitor_id = ?", monitor.ID).Count(&incidentCount)

	var reloaded models.Monitor
	db.First(&reloaded, "id = ?", monitor.ID)

	if checkCount != 1 {
		t.Errorf("checks = %d, want 1", checkCount)
	}
	if incidentCount != 1 {
		t.Errorf("incidents = %d, want 1", incidentCount)
	}
	if reloaded.Status != models.StatusDown {
		t.Errorf("monitor status = %q, want down", reloaded.Status)
	}
	if reloaded.LastCheckedAt == nil {
		t.Error("last_checked_at was not set")
	}

	// Now force a failure: ask it to open a SECOND incident while one is
	// already open. The partial unique index rejects the insert, and the whole
	// transaction must roll back - leaving no orphan check row behind.
	before := checkCount

	err := repo.RecordResult(ctx, monitor.ID, failure, worker.Transition{
		NewStatus:    models.StatusDown,
		OpenIncident: true, // invalid: one is already open
	})
	if err == nil {
		t.Fatal("expected the duplicate incident to be rejected")
	}

	var after int64
	db.Model(&models.Check{}).Where("monitor_id = ?", monitor.ID).Count(&after)

	// THE POINT OF THIS TEST.
	//
	// The check INSERT succeeded before the incident INSERT failed. Without a
	// transaction that check row would still be sitting there, recording a
	// measurement that officially never happened.
	if after != before {
		t.Errorf("ROLLBACK FAILED: checks went from %d to %d after a failed transaction",
			before, after)
	}
}

// Recovery must close the open incident and record a sane duration.
func TestRecordResult_RecoveryClosesIncident(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()

	repo := repository.NewCheckRepository(db)

	user := testutil.CreateUser(t, db, "recovery@test.local")
	project := testutil.CreateProject(t, db, user.ID, "P", "recovery-project")
	monitor := testutil.CreateMonitor(t, db, project.ID, "M")

	// Go down.
	code := 500
	downAt := time.Now().UTC().Add(-5 * time.Minute)
	if err := repo.RecordResult(ctx, monitor.ID,
		worker.CheckResult{CheckedAt: downAt, Success: false, StatusCode: &code, Error: "boom"},
		worker.Transition{NewStatus: models.StatusDown, OpenIncident: true},
	); err != nil {
		t.Fatalf("recording the failure: %v", err)
	}

	// Recover, five minutes later.
	okCode := 200
	upAt := downAt.Add(5 * time.Minute)
	if err := repo.RecordResult(ctx, monitor.ID,
		worker.CheckResult{CheckedAt: upAt, Success: true, StatusCode: &okCode, ResponseTimeMs: 90},
		worker.Transition{NewStatus: models.StatusUp, CloseIncident: true},
	); err != nil {
		t.Fatalf("recording the recovery: %v", err)
	}

	var incident models.Incident
	if err := db.Where("monitor_id = ?", monitor.ID).First(&incident).Error; err != nil {
		t.Fatalf("loading the incident: %v", err)
	}

	if incident.ResolvedAt == nil {
		t.Fatal("the incident was not closed")
	}
	if incident.DurationSeconds == nil {
		t.Fatal("duration_seconds was not set")
	}
	// 5 minutes = 300 seconds. Allow a second of slack for rounding.
	if *incident.DurationSeconds < 299 || *incident.DurationSeconds > 301 {
		t.Errorf("duration = %ds, want about 300", *incident.DurationSeconds)
	}

	// And exactly one incident for one outage.
	var count int64
	db.Model(&models.Incident{}).Where("monitor_id = ?", monitor.ID).Count(&count)
	if count != 1 {
		t.Errorf("incidents = %d, want 1", count)
	}
}

// Deleting a user must cascade all the way down.
func TestCascadeDelete(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()

	repo := repository.NewCheckRepository(db)

	user := testutil.CreateUser(t, db, "cascade@test.local")
	project := testutil.CreateProject(t, db, user.ID, "P", "cascade-project")
	monitor := testutil.CreateMonitor(t, db, project.ID, "M")

	code := 500
	if err := repo.RecordResult(ctx, monitor.ID,
		worker.CheckResult{CheckedAt: time.Now().UTC(), Success: false, StatusCode: &code, Error: "x"},
		worker.Transition{NewStatus: models.StatusDown, OpenIncident: true},
	); err != nil {
		t.Fatalf("seeding data: %v", err)
	}

	// Delete ONE row.
	if err := db.Delete(&models.User{}, "id = ?", user.ID).Error; err != nil {
		t.Fatalf("deleting the user: %v", err)
	}

	// Everything beneath it must be gone. Orphan rows - a monitor whose owner
	// no longer exists - cause bugs that are very hard to trace.
	for _, table := range []string{"projects", "monitors", "checks", "incidents"} {
		var count int64
		if err := db.Table(table).Count(&count).Error; err != nil {
			t.Fatalf("counting %s: %v", table, err)
		}
		if count != 0 {
			t.Errorf("%s still has %d row(s) after the owner was deleted", table, count)
		}
	}
}
