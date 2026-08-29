package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ariyam/api-monitor/internal/models"
	"github.com/ariyam/api-monitor/internal/worker"
)

// CheckRepository writes check results and reads check history.
type CheckRepository struct {
	db *gorm.DB
}

func NewCheckRepository(db *gorm.DB) *CheckRepository {
	return &CheckRepository{db: db}
}

// RecordResult saves everything one check produced, in ONE transaction.
//
// Up to three tables change together:
//
//	INSERT into checks          - what happened
//	UPDATE monitors             - current status and last_checked_at
//	INSERT or UPDATE incidents  - only on a transition
//
// WHY A TRANSACTION.
//
// Without one, a crash between statements leaves the database lying. Insert
// the check, mark the monitor 'down', then crash before opening the incident:
// the dashboard now shows a monitor that is down with no incident explaining
// it, and because the status is already 'down', no future check will ever
// open one - the transition has been consumed. The outage becomes invisible
// and stays invisible.
//
// A transaction makes all of it happen or none of it. That is what ACID
// atomicity means, and this is the clearest example of why it exists.
func (r *CheckRepository) RecordResult(
	ctx context.Context,
	monitorID uuid.UUID,
	result worker.CheckResult,
	transition worker.Transition,
) error {
	// GORM's Transaction helper commits when the function returns nil and
	// rolls back on any error - including a panic. Doing this by hand means
	// remembering to roll back on every early return, and one missed path
	// leaves a transaction open holding locks.
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// ---- 1. record what happened -----------------------------------
		check := models.Check{
			MonitorID:      monitorID,
			CheckedAt:      result.CheckedAt,
			Success:        result.Success,
			StatusCode:     result.StatusCode, // nil when nothing responded
			ResponseTimeMs: result.ResponseTimeMs,
			Error:          result.Error,
		}
		if err := tx.Create(&check).Error; err != nil {
			return fmt.Errorf("inserting check: %w", err)
		}

		// ---- 2. update the monitor's current state ---------------------
		updates := map[string]any{
			"status":          transition.NewStatus,
			"last_checked_at": result.CheckedAt,
			"updated_at":      time.Now().UTC(),
		}
		if err := tx.Model(&models.Monitor{}).
			Where("id = ?", monitorID).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("updating monitor status: %w", err)
		}

		// ---- 3. open or close an incident, if this was a transition ----
		if transition.OpenIncident {
			incident := models.Incident{
				MonitorID: monitorID,
				StartedAt: result.CheckedAt,
				Cause:     result.Error,
			}
			if err := tx.Create(&incident).Error; err != nil {
				// idx_incidents_one_open is a partial unique index allowing at
				// most one unresolved incident per monitor. If our logic ever
				// tried to open a second, PostgreSQL rejects it here and the
				// whole transaction rolls back.
				//
				// The database enforcing the rule is what makes it survive bugs
				// in this code.
				if isUniqueViolation(err) {
					return fmt.Errorf("an incident is already open for this monitor: %w", err)
				}
				return fmt.Errorf("opening incident: %w", err)
			}
		}

		if transition.CloseIncident {
			// Close whichever incident is currently open for this monitor.
			// duration_seconds is computed in SQL from the two timestamps, so
			// it cannot disagree with them.
			result := tx.Model(&models.Incident{}).
				Where("monitor_id = ? AND resolved_at IS NULL", monitorID).
				Updates(map[string]any{
					"resolved_at": check.CheckedAt,
					"duration_seconds": gorm.Expr(
						"EXTRACT(EPOCH FROM (? - started_at))::int", check.CheckedAt),
				})
			if result.Error != nil {
				return fmt.Errorf("closing incident: %w", result.Error)
			}
			// Zero rows is not fatal - it can happen if the database was
			// cleared while the monitor was down - but it is worth surfacing
			// rather than silently ignoring.
			if result.RowsAffected == 0 {
				return fmt.Errorf("expected an open incident to close, found none")
			}
		}

		return nil // returning nil COMMITS
	})
}

// ListByMonitorForUser returns recent checks for a monitor the user owns.
func (r *CheckRepository) ListByMonitorForUser(
	ctx context.Context,
	monitorID, userID uuid.UUID,
	limit int,
	since time.Time,
) ([]models.Check, error) {
	// Clamp the limit. Never let a client decide how much work the server
	// does - limit=1000000 would happily try to load a million rows.
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var checks []models.Check

	// Ownership is enforced in the query itself by joining up to projects.
	// A monitor belonging to someone else simply matches no rows.
	err := r.db.WithContext(ctx).
		Model(&models.Check{}).
		Select("checks.*").
		Joins("JOIN monitors ON monitors.id = checks.monitor_id").
		Joins("JOIN projects ON projects.id = monitors.project_id").
		Where("checks.monitor_id = ? AND projects.user_id = ? AND checks.checked_at >= ?",
			monitorID, userID, since).
		// Matches idx_checks_monitor_checked_at exactly, so PostgreSQL reads
		// the rows already in order instead of sorting them.
		Order("checks.checked_at DESC").
		Limit(limit).
		Find(&checks).Error
	if err != nil {
		return nil, fmt.Errorf("listing checks: %w", err)
	}

	return checks, nil
}

// LatestForMonitor returns the most recent check, or nil if there are none.
func (r *CheckRepository) LatestForMonitor(ctx context.Context, monitorID uuid.UUID) (*models.Check, error) {
	var check models.Check

	err := r.db.WithContext(ctx).
		Where("monitor_id = ?", monitorID).
		Order("checked_at DESC").
		First(&check).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // no checks yet is not an error
		}
		return nil, fmt.Errorf("finding latest check: %w", err)
	}

	return &check, nil
}
