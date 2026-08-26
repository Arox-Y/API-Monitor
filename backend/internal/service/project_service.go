package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/ariyam/api-monitor/internal/apperr"
	"github.com/ariyam/api-monitor/internal/models"
	"github.com/ariyam/api-monitor/internal/repository"
)

// ProjectRepository is what ProjectService needs from storage.
//
// Every method takes a userID, because there is no legitimate reason to read
// or change a project without knowing whose it is.
type ProjectRepository interface {
	Create(ctx context.Context, project *models.Project) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Project, error)
	GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*models.Project, error)
	Update(ctx context.Context, project *models.Project, userID uuid.UUID) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// ProjectService holds the project rules.
type ProjectService struct {
	projects ProjectRepository
}

func NewProjectService(projects ProjectRepository) *ProjectService {
	return &ProjectService{projects: projects}
}

// CreateProjectInput is what creating a project needs.
//
// Note there is no Slug field. Slugs are server-generated: if a client could
// choose one, the first person to sign up could claim /status/stripe.
type CreateProjectInput struct {
	Name        string
	Description string
	IsPublic    bool
}

// maxSlugAttempts bounds the collision-retry loop.
const maxSlugAttempts = 5

// Create makes a new project for a user.
func (s *ProjectService) Create(ctx context.Context, userID uuid.UUID, in CreateProjectInput) (*models.Project, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)

	if err := validateProject(in.Name, in.Description); err != nil {
		return nil, err
	}

	base := slugify(in.Name)

	// Slugs are globally unique, so two users naming a project "API" collide.
	// Rather than asking the database "does this slug exist?" and then
	// inserting - a race between the two - we attempt the insert and add a
	// random suffix if it is rejected.
	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		slug := base
		if attempt > 0 {
			suffix, err := randomSuffix()
			if err != nil {
				return nil, err
			}
			slug = fmt.Sprintf("%s-%s", base, suffix)
		}

		project := &models.Project{
			UserID:      userID,
			Name:        in.Name,
			Description: in.Description,
			Slug:        slug,
			IsPublic:    in.IsPublic,
		}

		err := s.projects.Create(ctx, project)
		if err == nil {
			return project, nil
		}
		if !errors.Is(err, repository.ErrSlugTaken) {
			return nil, err
		}
		// Slug collision: loop and try again with a new suffix.
	}

	return nil, fmt.Errorf("could not generate a unique slug after %d attempts", maxSlugAttempts)
}

// List returns the user's projects.
func (s *ProjectService) List(ctx context.Context, userID uuid.UUID) ([]models.Project, error) {
	return s.projects.ListByUser(ctx, userID)
}

// Get returns one project the user owns, or apperr.ErrNotFound.
func (s *ProjectService) Get(ctx context.Context, id, userID uuid.UUID) (*models.Project, error) {
	return s.projects.GetByIDForUser(ctx, id, userID)
}

// UpdateProjectInput carries only the fields a client may change.
//
// The fields are POINTERS so we can tell "not supplied" from "set to empty".
// With a plain string, a PATCH that omits the name is indistinguishable from
// one setting the name to "" - and we would wipe it. With *string, nil means
// leave it alone.
//
// This is the classic reason Go APIs use pointers for optional input.
type UpdateProjectInput struct {
	Name        *string
	Description *string
	IsPublic    *bool
}

// Update changes a project the user owns.
func (s *ProjectService) Update(ctx context.Context, id, userID uuid.UUID, in UpdateProjectInput) (*models.Project, error) {
	// Load the current state first. This both fetches the values we are not
	// changing and proves the user owns it - a request for someone else's
	// project stops here with ErrNotFound.
	project, err := s.projects.GetByIDForUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		project.Name = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		project.Description = strings.TrimSpace(*in.Description)
	}
	if in.IsPublic != nil {
		project.IsPublic = *in.IsPublic
	}

	if err := validateProject(project.Name, project.Description); err != nil {
		return nil, err
	}

	// The slug deliberately does NOT change when the name does. Renaming a
	// project must not break a status page URL people have already shared.
	if err := s.projects.Update(ctx, project, userID); err != nil {
		return nil, err
	}

	return s.projects.GetByIDForUser(ctx, id, userID)
}

// Delete removes a project the user owns, cascading to its monitors.
func (s *ProjectService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.projects.Delete(ctx, id, userID)
}

func validateProject(name, description string) error {
	var fields []apperr.FieldError

	if name == "" {
		fields = append(fields, apperr.FieldError{Field: "name", Message: "is required"})
	} else if len(name) > 100 {
		fields = append(fields, apperr.FieldError{Field: "name", Message: "must be at most 100 characters"})
	}

	if len(description) > 500 {
		fields = append(fields, apperr.FieldError{Field: "description", Message: "must be at most 500 characters"})
	}

	if len(fields) > 0 {
		return apperr.NewValidationError(fields...)
	}
	return nil
}
