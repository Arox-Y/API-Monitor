package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ariyam/api-monitor/internal/service"
)

// StatusHandler serves the PUBLIC status page endpoint.
//
// This is the only data endpoint anyone on the internet can call without
// signing in, so it is kept in its own file - the boundary should be obvious
// to anybody reading the codebase.
type StatusHandler struct {
	status *service.StatusService
}

func NewStatusHandler(status *service.StatusService) *StatusHandler {
	return &StatusHandler{status: status}
}

// Get handles GET /api/v1/public/status/:slug.
//
// No authentication, and therefore no user id anywhere in this handler.
// The access rule lives in the query, which requires is_public = true.
func (h *StatusHandler) Get(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "Status page not found.")
		return
	}

	status, err := h.status.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		// A private project and a non-existent one return the same 404, so a
		// stranger cannot use this endpoint to discover which slugs are real.
		respondWithDomainError(c, err)
		return
	}

	// Cache briefly. A status page can be linked from somewhere busy, and this
	// caps how often a traffic spike reaches the database.
	c.Header("Cache-Control", "public, max-age=30")

	c.JSON(http.StatusOK, status)
}
