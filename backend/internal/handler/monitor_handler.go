package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ariyam/api-monitor/internal/service"
)

// MonitorHandler serves the /monitors endpoints.
type MonitorHandler struct {
	monitors *service.MonitorService
}

func NewMonitorHandler(monitors *service.MonitorService) *MonitorHandler {
	return &MonitorHandler{monitors: monitors}
}

type createMonitorRequest struct {
	Name            string `json:"name" binding:"required"`
	URL             string `json:"url" binding:"required"`
	Method          string `json:"method"`
	IntervalSeconds *int   `json:"interval_seconds"`
	TimeoutSeconds  *int   `json:"timeout_seconds"`
	ExpectedStatus  *int   `json:"expected_status"`
	Enabled         *bool  `json:"enabled"`
}

type updateMonitorRequest struct {
	Name            *string `json:"name"`
	URL             *string `json:"url"`
	Method          *string `json:"method"`
	IntervalSeconds *int    `json:"interval_seconds"`
	TimeoutSeconds  *int    `json:"timeout_seconds"`
	ExpectedStatus  *int    `json:"expected_status"`
	Enabled         *bool   `json:"enabled"`
}

// List handles GET /api/v1/monitors - every monitor the user owns.
func (h *MonitorHandler) List(c *gin.Context) {
	userID, ok := requireUser(c)
	if !ok {
		return
	}

	monitors, err := h.monitors.List(c.Request.Context(), userID)
	if err != nil {
		respondWithDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"monitors": monitors})
}

// ListByProject handles GET /api/v1/projects/:id/monitors.
func (h *MonitorHandler) ListByProject(c *gin.Context) {
	userID, ok := requireUser(c)
	if !ok {
		return
	}
	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	monitors, err := h.monitors.ListByProject(c.Request.Context(), projectID, userID)
	if err != nil {
		respondWithDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"monitors": monitors})
}

// Create handles POST /api/v1/projects/:id/monitors.
//
// Creation is nested under a project because a monitor cannot exist without
// one - the URL makes the parent impossible to forget. Reads and updates use
// the flat /monitors/:id, since a monitor ID is globally unique on its own.
func (h *MonitorHandler) Create(c *gin.Context) {
	userID, ok := requireUser(c)
	if !ok {
		return
	}
	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req createMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_FAILED", "A monitor name and url are required.")
		return
	}

	monitor, err := h.monitors.Create(c.Request.Context(), projectID, userID, service.CreateMonitorInput{
		Name:            req.Name,
		URL:             req.URL,
		Method:          req.Method,
		IntervalSeconds: req.IntervalSeconds,
		TimeoutSeconds:  req.TimeoutSeconds,
		ExpectedStatus:  req.ExpectedStatus,
		Enabled:         req.Enabled,
	})
	if err != nil {
		respondWithDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, monitor)
}

// Get handles GET /api/v1/monitors/:id.
func (h *MonitorHandler) Get(c *gin.Context) {
	userID, ok := requireUser(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	monitor, err := h.monitors.Get(c.Request.Context(), id, userID)
	if err != nil {
		respondWithDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, monitor)
}

// Update handles PATCH /api/v1/monitors/:id.
//
// Enable and disable go through here as {"enabled": false} rather than a
// separate /pause endpoint. It is a field change, and PATCH is the verb for
// changing fields.
func (h *MonitorHandler) Update(c *gin.Context) {
	userID, ok := requireUser(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req updateMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_FAILED", "The request body was not valid JSON.")
		return
	}

	monitor, err := h.monitors.Update(c.Request.Context(), id, userID, service.UpdateMonitorInput{
		Name:            req.Name,
		URL:             req.URL,
		Method:          req.Method,
		IntervalSeconds: req.IntervalSeconds,
		TimeoutSeconds:  req.TimeoutSeconds,
		ExpectedStatus:  req.ExpectedStatus,
		Enabled:         req.Enabled,
	})
	if err != nil {
		respondWithDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, monitor)
}

// Delete handles DELETE /api/v1/monitors/:id.
func (h *MonitorHandler) Delete(c *gin.Context) {
	userID, ok := requireUser(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.monitors.Delete(c.Request.Context(), id, userID); err != nil {
		respondWithDomainError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
