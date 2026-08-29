package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ariyam/api-monitor/internal/models"
)

// IncidentRepository reads incident history.
//
// Incidents are WRITTEN by the worker, inside the check transaction in
// check_repository.go. This type only reads them.
type IncidentRepository struct {
	db *gorm.DB
}

func NewIncidentRepository(db *gorm.DB) *IncidentRepository {
	return &IncidentRepository{db: db}
}

// ListRecentForUser returns the newest incidents across all a user's monitors.
func (r *IncidentRepository) ListRecentForUser(
	ctx context.Context, userID uuid.UUID, limit int,
) ([]models.Incident, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var incidents []models.Incident

	// monitors.name is selected AS monitor_name so each incident arrives with
	// the name already attached. Without it the UI would need one extra
	// request per row - the N+1 problem, solved here in the query.
	err := r.db.WithContext(ctx).
		Model(&models.Incident{}).
		Select("incidents.*, monitors.name AS monitor_name").
		Joins("JOIN monitors ON monitors.id = incidents.monitor_id").
		Joins("JOIN projects ON projects.id = monitors.project_id").
		Where("projects.user_id = ?", userID).
		// Open incidents first (resolved_at IS NULL sorts before NOT NULL with
		// NULLS FIRST), then newest. What is broken right now matters most.
		Order("incidents.resolved_at ASC NULLS FIRST, incidents.started_at DESC").
		Limit(limit).
		Find(&incidents).Error
	if err != nil {
		return nil, fmt.Errorf("listing incidents: %w", err)
	}

	return incidents, nil
}

// ListByMonitorForUser returns incidents for one monitor the user owns.
func (r *IncidentRepository) ListByMonitorForUser(
	ctx context.Context, monitorID, userID uuid.UUID, limit int,
) ([]models.Incident, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var incidents []models.Incident

	err := r.db.WithContext(ctx).
		Model(&models.Incident{}).
		Select("incidents.*, monitors.name AS monitor_name").
		Joins("JOIN monitors ON monitors.id = incidents.monitor_id").
		Joins("JOIN projects ON projects.id = monitors.project_id").
		Where("incidents.monitor_id = ? AND projects.user_id = ?", monitorID, userID).
		Order("incidents.started_at DESC").
		Limit(limit).
		Find(&incidents).Error
	if err != nil {
		return nil, fmt.Errorf("listing monitor incidents: %w", err)
	}

	return incidents, nil
}

// ListOpenByProject returns the unresolved incidents on a public status page.
func (r *IncidentRepository) ListOpenByProject(
	ctx context.Context, projectID uuid.UUID,
) ([]models.Incident, error) {
	var incidents []models.Incident

	err := r.db.WithContext(ctx).
		Model(&models.Incident{}).
		Select("incidents.*, monitors.name AS monitor_name").
		Joins("JOIN monitors ON monitors.id = incidents.monitor_id").
		Where("monitors.project_id = ? AND incidents.resolved_at IS NULL", projectID).
		Order("incidents.started_at DESC").
		Find(&incidents).Error
	if err != nil {
		return nil, fmt.Errorf("listing open incidents: %w", err)
	}

	return incidents, nil
}
