package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ariyam/api-monitor/internal/models"
)

// StatsRepository runs the aggregate queries the dashboard needs.
type StatsRepository struct {
	db *gorm.DB
}

func NewStatsRepository(db *gorm.DB) *StatsRepository {
	return &StatsRepository{db: db}
}

// MonitorCounts is the up/down/total summary.
type MonitorCounts struct {
	Total int64
	Up    int64
	Down  int64
}

// CountMonitors returns all three counts in ONE query.
//
// FILTER is PostgreSQL's clean way to count a subset. Three separate COUNT
// queries would mean three round trips to produce one row of dashboard.
func (r *StatsRepository) CountMonitors(ctx context.Context, userID uuid.UUID) (MonitorCounts, error) {
	var counts MonitorCounts

	err := r.db.WithContext(ctx).
		Model(&models.Monitor{}).
		Select(`COUNT(*) AS total,
		        COUNT(*) FILTER (WHERE monitors.status = 'up')   AS up,
		        COUNT(*) FILTER (WHERE monitors.status = 'down') AS down`).
		Joins("JOIN projects ON projects.id = monitors.project_id").
		Where("projects.user_id = ?", userID).
		Scan(&counts).Error
	if err != nil {
		return MonitorCounts{}, fmt.Errorf("counting monitors: %w", err)
	}

	return counts, nil
}

// UptimePercentage24h returns successful checks as a percentage of all checks
// in the last 24 hours.
//
// It returns a POINTER so it can be nil.
//
// nil means "no checks have been recorded", which is NOT the same as 0%.
// A brand-new account has measured nothing; returning 0 would tell the user
// every service is down. Distinguishing unknown from zero is the whole reason
// this returns *float64 rather than float64.
func (r *StatsRepository) UptimePercentage24h(ctx context.Context, userID uuid.UUID) (*float64, error) {
	var result struct {
		Total   int64
		Success int64
	}

	since := time.Now().UTC().Add(-24 * time.Hour)

	err := r.db.WithContext(ctx).
		Model(&models.Check{}).
		Select(`COUNT(*) AS total,
		        COUNT(*) FILTER (WHERE checks.success) AS success`).
		Joins("JOIN monitors ON monitors.id = checks.monitor_id").
		Joins("JOIN projects ON projects.id = monitors.project_id").
		Where("projects.user_id = ? AND checks.checked_at >= ?", userID, since).
		Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("calculating uptime: %w", err)
	}

	// The division guard. Doing this in Go rather than with NULLIF in SQL
	// keeps the "unknown" case explicit and impossible to misread.
	if result.Total == 0 {
		return nil, nil
	}

	percentage := float64(result.Success) / float64(result.Total) * 100
	return &percentage, nil
}

// RecentChecksForUser returns the newest checks across all a user's monitors,
// each carrying the monitor's name.
func (r *StatsRepository) RecentChecksForUser(
	ctx context.Context, userID uuid.UUID, limit int,
) ([]models.Check, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	var checks []models.Check

	err := r.db.WithContext(ctx).
		Model(&models.Check{}).
		Select("checks.*, monitors.name AS monitor_name").
		Joins("JOIN monitors ON monitors.id = checks.monitor_id").
		Joins("JOIN projects ON projects.id = monitors.project_id").
		Where("projects.user_id = ?", userID).
		Order("checks.checked_at DESC").
		Limit(limit).
		Find(&checks).Error
	if err != nil {
		return nil, fmt.Errorf("listing recent checks: %w", err)
	}

	return checks, nil
}

// UptimeForMonitor returns one monitor's uptime over a window, or nil if it
// has no checks in that period.
func (r *StatsRepository) UptimeForMonitor(
	ctx context.Context, monitorID uuid.UUID, since time.Time,
) (*float64, error) {
	var result struct {
		Total   int64
		Success int64
	}

	err := r.db.WithContext(ctx).
		Model(&models.Check{}).
		Select(`COUNT(*) AS total, COUNT(*) FILTER (WHERE success) AS success`).
		Where("monitor_id = ? AND checked_at >= ?", monitorID, since).
		Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("calculating monitor uptime: %w", err)
	}

	if result.Total == 0 {
		return nil, nil
	}

	percentage := float64(result.Success) / float64(result.Total) * 100
	return &percentage, nil
}
