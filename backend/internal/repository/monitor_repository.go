package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ariyam/api-monitor/internal/apperr"
	"github.com/ariyam/api-monitor/internal/models"
)

// MonitorRepository runs monitor queries.
//
// Monitors have no user_id column. Ownership is reached through the project:
//
//	monitors -> projects -> users
//
// So every scoped query here JOINs projects and filters on projects.user_id.
// That is one extra indexed join, and in exchange ownership has exactly one
// definition that cannot drift out of sync. See ADR-007.
type MonitorRepository struct {
	db *gorm.DB
}

func NewMonitorRepository(db *gorm.DB) *MonitorRepository {
	return &MonitorRepository{db: db}
}

// Create inserts a monitor. The caller must already have proved the user owns
// the target project.
func (r *MonitorRepository) Create(ctx context.Context, monitor *models.Monitor) error {
	if err := r.db.WithContext(ctx).Create(monitor).Error; err != nil {
		return fmt.Errorf("creating monitor: %w", err)
	}
	return nil
}

// ListByUser returns every monitor across all of a user's projects.
func (r *MonitorRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Monitor, error) {
	var monitors []models.Monitor

	err := r.db.WithContext(ctx).
		Model(&models.Monitor{}).
		Joins("JOIN projects ON projects.id = monitors.project_id").
		Where("projects.user_id = ?", userID).
		Order("monitors.created_at DESC").
		Find(&monitors).Error
	if err != nil {
		return nil, fmt.Errorf("listing monitors: %w", err)
	}

	return monitors, nil
}

// ListByProject returns the monitors in one project the user owns.
func (r *MonitorRepository) ListByProject(ctx context.Context, projectID, userID uuid.UUID) ([]models.Monitor, error) {
	var monitors []models.Monitor

	err := r.db.WithContext(ctx).
		Model(&models.Monitor{}).
		Joins("JOIN projects ON projects.id = monitors.project_id").
		Where("monitors.project_id = ? AND projects.user_id = ?", projectID, userID).
		Order("monitors.created_at DESC").
		Find(&monitors).Error
	if err != nil {
		return nil, fmt.Errorf("listing project monitors: %w", err)
	}

	return monitors, nil
}

// GetByIDForUser returns a monitor only if the user owns its project.
//
// Not found and not yours produce the same apperr.ErrNotFound, so the API
// cannot be used to discover which monitor IDs exist.
func (r *MonitorRepository) GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*models.Monitor, error) {
	var monitor models.Monitor

	err := r.db.WithContext(ctx).
		Model(&models.Monitor{}).
		// Select monitors.* explicitly. Without it the join's project columns
		// would also be scanned, and a column named on both tables (id,
		// created_at, name) would silently overwrite the monitor's own value.
		Select("monitors.*").
		Joins("JOIN projects ON projects.id = monitors.project_id").
		Where("monitors.id = ? AND projects.user_id = ?", id, userID).
		First(&monitor).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("finding monitor: %w", err)
	}

	return &monitor, nil
}

// ListByPublicProject returns the monitors shown on a public status page.
//
// No user ID, because this feeds the unauthenticated page. Only enabled
// monitors appear, and the caller must already have confirmed the project is
// public.
func (r *MonitorRepository) ListByPublicProject(ctx context.Context, projectID uuid.UUID) ([]models.Monitor, error) {
	var monitors []models.Monitor

	err := r.db.WithContext(ctx).
		Where("project_id = ? AND enabled = true", projectID).
		Order("created_at ASC").
		Find(&monitors).Error
	if err != nil {
		return nil, fmt.Errorf("listing public monitors: %w", err)
	}

	return monitors, nil
}

// Update saves changes to a monitor the user owns.
//
// GORM cannot express "UPDATE ... FROM a join" portably, so ownership is
// enforced with a subquery instead: the row must belong to a project owned by
// this user.
func (r *MonitorRepository) Update(ctx context.Context, monitor *models.Monitor, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&models.Monitor{}).
		Where("id = ? AND project_id IN (SELECT id FROM projects WHERE user_id = ?)", monitor.ID, userID).
		Updates(map[string]any{
			"name":             monitor.Name,
			"url":              monitor.URL,
			"method":           monitor.Method,
			"interval_seconds": monitor.IntervalSeconds,
			"timeout_seconds":  monitor.TimeoutSeconds,
			"expected_status":  monitor.ExpectedStatus,
			"enabled":          monitor.Enabled,
		})

	if result.Error != nil {
		return fmt.Errorf("updating monitor: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperr.ErrNotFound
	}

	return nil
}

// Delete removes a monitor the user owns, cascading to its checks and
// incidents.
func (r *MonitorRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND project_id IN (SELECT id FROM projects WHERE user_id = ?)", id, userID).
		Delete(&models.Monitor{})

	if result.Error != nil {
		return fmt.Errorf("deleting monitor: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperr.ErrNotFound
	}

	return nil
}
