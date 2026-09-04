package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ariyam/api-monitor/internal/service"
)

// DashboardHandler serves dashboard aggregates and monitor history.
type DashboardHandler struct {
	dashboard *service.DashboardService
}

func NewDashboardHandler(dashboard *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard}
}

// Stats handles GET /api/v1/dashboard/stats.
func (h *DashboardHandler) Stats(c *gin.Context) {
	userID, ok := requireUser(c)
	if !ok {
		return
	}

	stats, err := h.dashboard.GetStats(c.Request.Context(), userID)
	if err != nil {
		respondWithDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// MonitorDetail handles GET /api/v1/monitors/:id/checks.
//
// It returns the monitor together with its recent checks, incidents and
// uptime, because the detail screen needs all four and one request beats four.
func (h *DashboardHandler) MonitorDetail(c *gin.Context) {
	userID, ok := requireUser(c)
	if !ok {
		return
	}
	monitorID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	detail, err := h.dashboard.GetMonitorDetail(
		c.Request.Context(), monitorID, userID, parseLimit(c, 100, 200),
	)
	if err != nil {
		respondWithDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, detail)
}

// Incidents handles GET /api/v1/incidents.
//
// The dashboard already receives the five most recent incidents inside
// /dashboard/stats, which is what that screen needs. This endpoint exists for
// the case that aggregate cannot serve: browsing further back than five.
func (h *DashboardHandler) Incidents(c *gin.Context) {
	userID, ok := requireUser(c)
	if !ok {
		return
	}

	incidents, err := h.dashboard.ListIncidents(
		c.Request.Context(), userID, parseLimit(c, 20, 100),
	)
	if err != nil {
		respondWithDomainError(c, err)
		return
	}

	// Wrapped in an object rather than returned as a bare array, so pagination
	// can be added later without breaking every client.
	c.JSON(http.StatusOK, gin.H{"incidents": incidents})
}

// parseLimit reads a ?limit= query parameter and CLAMPS it.
//
// Never let a client decide how much work the server does. A request for
// limit=1000000 would otherwise try to load a million rows and could take the
// service down - which is a denial of service anyone could trigger by editing
// a URL.
func parseLimit(c *gin.Context, fallback, max int) int {
	raw := c.Query("limit")
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}
