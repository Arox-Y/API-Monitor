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

// ProjectRepository runs project queries.
//
// EVERY method that reads or changes a project takes a userID and filters by
// it. There is deliberately no plain GetByID: if an unscoped version existed,
// somebody would eventually call it and hand one user another user's data.
//
// Making the safe thing the ONLY thing is better than making it the default.
type ProjectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create inserts a project. The caller sets UserID and Slug.
func (r *ProjectRepository) Create(ctx context.Context, project *models.Project) error {
	if err := r.db.WithContext(ctx).Create(project).Error; err != nil {
		if isUniqueViolation(err) {
			// The only unique column here is slug.
			return errSlugTaken
		}
		return fmt.Errorf("creating project: %w", err)
	}
	return nil
}

// errSlugTaken signals a slug collision so the service can retry with a suffix.
var errSlugTaken = errors.New("slug already taken")

// ErrSlugTaken is exported for the service layer's retry loop.
var ErrSlugTaken = errSlugTaken

// ListByUser returns every project owned by a user, newest first, each with
// its monitor count.
func (r *ProjectRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Project, error) {
	var projects []models.Project

	// One query with a subquery, rather than fetching projects and then
	// counting monitors for each. The naive version issues 1 + N queries -
	// the N+1 problem - and gets slower with every project a user adds.
	err := r.db.WithContext(ctx).
		Model(&models.Project{}).
		Select("projects.*, (SELECT COUNT(*) FROM monitors WHERE monitors.project_id = projects.id) AS monitor_count").
		Where("projects.user_id = ?", userID).
		Order("projects.created_at DESC").
		Find(&projects).Error
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}

	return projects, nil
}

// GetByIDForUser returns a project only if it belongs to this user.
//
// If the project does not exist, OR exists but belongs to somebody else, the
// result is the same: apperr.ErrNotFound. The caller cannot tell the cases
// apart, so neither can an attacker probing IDs. See ADR-009.
func (r *ProjectRepository) GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*models.Project, error) {
	var project models.Project

	err := r.db.WithContext(ctx).
		Model(&models.Project{}).
		Select("projects.*, (SELECT COUNT(*) FROM monitors WHERE monitors.project_id = projects.id) AS monitor_count").
		Where("projects.id = ? AND projects.user_id = ?", id, userID).
		First(&project).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("finding project: %w", err)
	}

	return &project, nil
}

// GetPublicBySlug returns a project for the public status page.
//
// No user ID: this is the one endpoint anybody may call. Note it also requires
// is_public - a private project must be invisible even to someone who guesses
// its slug.
func (r *ProjectRepository) GetPublicBySlug(ctx context.Context, slug string) (*models.Project, error) {
	var project models.Project

	err := r.db.WithContext(ctx).
		Where("slug = ? AND is_public = true", slug).
		First(&project).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("finding public project: %w", err)
	}

	return &project, nil
}

// Update saves changes to a project the user owns.
func (r *ProjectRepository) Update(ctx context.Context, project *models.Project, userID uuid.UUID) error {
	// The WHERE clause carries user_id as well as id. Even if a bug let a
	// foreign project reach this function, the UPDATE would match zero rows
	// rather than modifying someone else's data.
	result := r.db.WithContext(ctx).
		Model(&models.Project{}).
		Where("id = ? AND user_id = ?", project.ID, userID).
		Updates(map[string]any{
			"name":        project.Name,
			"description": project.Description,
			"is_public":   project.IsPublic,
		})

	if result.Error != nil {
		return fmt.Errorf("updating project: %w", result.Error)
	}
	// Zero rows changed means it did not exist, or was not theirs.
	if result.RowsAffected == 0 {
		return apperr.ErrNotFound
	}

	return nil
}

// Delete removes a project the user owns.
//
// The database cascades this to monitors, and from there to checks and
// incidents, so no orphan rows can survive.
func (r *ProjectRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Project{})

	if result.Error != nil {
		return fmt.Errorf("deleting project: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperr.ErrNotFound
	}

	return nil
}
