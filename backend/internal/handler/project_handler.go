package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ariyam/api-monitor/internal/middleware"
	"github.com/ariyam/api-monitor/internal/service"
)

// ProjectHandler serves the /projects endpoints.
type ProjectHandler struct {
	projects *service.ProjectService
}

func NewProjectHandler(projects *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projects: projects}
}

type createProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

// updateProjectRequest uses POINTERS for every field.
//
// With a plain string, a PATCH body of {"is_public": true} would arrive with
// Name == "" - indistinguishable from someone deliberately clearing the name,
// and we would wipe it. A nil pointer means "this field was not sent".
type updateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsPublic    *bool   `json:"is_public"`
}

// List handles GET /api/v1/projects.
func (h *ProjectHandler) List(c *gin.Context) {
	userID, ok := requireUser(c)
	if !ok {
		return
	}

	projects, err := h.projects.List(c.Request.Context(), userID)
	if err != nil {
		respondWithDomainError(c, err)
		return
	}

	// Wrapped in an object rather than returned as a bare array, so fields
	// like pagination can be added later without breaking every client.
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

// Create handles POST /api/v1/projects.
func (h *ProjectHandler) Create(c *gin.Context) {
	userID, ok := requireUser(c)
	if !ok {
		return
	}

	var req createProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_FAILED", "A project name is required.")
		return
	}

	project, err := h.projects.Create(c.Request.Context(), userID, service.CreateProjectInput{
		Name:        req.Name,
		Description: req.Description,
		IsPublic:    req.IsPublic,
	})
	if err != nil {
		respondWithDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, project)
}

// Get handles GET /api/v1/projects/:id.
func (h *ProjectHandler) Get(c *gin.Context) {
	userID, ok := requireUser(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	project, err := h.projects.Get(c.Request.Context(), id, userID)
	if err != nil {
		respondWithDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, project)
}

// Update handles PATCH /api/v1/projects/:id.
func (h *ProjectHandler) Update(c *gin.Context) {
	userID, ok := requireUser(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req updateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_FAILED", "The request body was not valid JSON.")
		return
	}

	project, err := h.projects.Update(c.Request.Context(), id, userID, service.UpdateProjectInput{
		Name:        req.Name,
		Description: req.Description,
		IsPublic:    req.IsPublic,
	})
	if err != nil {
		respondWithDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, project)
}

// Delete handles DELETE /api/v1/projects/:id.
func (h *ProjectHandler) Delete(c *gin.Context) {
	userID, ok := requireUser(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.projects.Delete(c.Request.Context(), id, userID); err != nil {
		respondWithDomainError(c, err)
		return
	}

	// 204 No Content: succeeded, and there is nothing to send back.
	c.Status(http.StatusNoContent)
}

// ---- small shared helpers -------------------------------------------------

// requireUser pulls the authenticated user out of the request context.
//
// The auth middleware guarantees it is there, so a miss means a routing
// mistake - a protected handler wired outside the protected group.
func requireUser(c *gin.Context) (uuid.UUID, bool) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required.")
		return uuid.Nil, false
	}
	return userID, true
}

// parseUUIDParam reads a UUID from the URL path.
//
// Anything the client controls must be validated. Without this, "abc" would
// reach the repository and produce a database error rather than a clean 400.
func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		// 400, not 404: the request itself is malformed.
		respondError(c, http.StatusBadRequest, "VALIDATION_FAILED", "The id in the URL is not valid.")
		return uuid.Nil, false
	}
	return id, true
}
