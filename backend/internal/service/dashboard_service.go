package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ariyam/api-monitor/internal/models"
	"github.com/ariyam/api-monitor/internal/repository"
)

// StatsRepository is what DashboardService needs for aggregates.
type StatsRepository interface {
	CountMonitors(ctx context.Context, userID uuid.UUID) (repository.MonitorCounts, error)
	UptimePercentage24h(ctx context.Context, userID uuid.UUID) (*float64, error)
	RecentChecksForUser(ctx context.Context, userID uuid.UUID, limit int) ([]models.Check, error)
	UptimeForMonitor(ctx context.Context, monitorID uuid.UUID, since time.Time) (*float64, error)
}

// IncidentRepository is what DashboardService needs for incident history.
type IncidentRepository interface {
	ListRecentForUser(ctx context.Context, userID uuid.UUID, limit int) ([]models.Incident, error)
	ListByMonitorForUser(ctx context.Context, monitorID, userID uuid.UUID, limit int) ([]models.Incident, error)
}

// CheckRepository is what the services need for check history.
type CheckRepository interface {
	ListByMonitorForUser(ctx context.Context, monitorID, userID uuid.UUID, limit int, since time.Time) ([]models.Check, error)
}

// DashboardStats is everything the dashboard screen needs, in one response.
type DashboardStats struct {
	TotalMonitors int64 `json:"total_monitors"`
	MonitorsUp    int64 `json:"monitors_up"`
	MonitorsDown  int64 `json:"monitors_down"`

	// Pointer so it can be null in JSON. null means "nothing measured yet",
	// which the UI must render as a dash, never as 0%.
	UptimePercentage24h *float64 `json:"uptime_percentage_24h"`

	RecentIncidents []models.Incident `json:"recent_incidents"`
	RecentChecks    []models.Check    `json:"recent_checks"`
}

// DashboardService assembles the dashboard payload.
type DashboardService struct {
	stats     StatsRepository
	incidents IncidentRepository
	checks    CheckRepository
	monitors  MonitorRepository
}

func NewDashboardService(
	stats StatsRepository,
	incidents IncidentRepository,
	checks CheckRepository,
	monitors MonitorRepository,
) *DashboardService {
	return &DashboardService{stats: stats, incidents: incidents, checks: checks, monitors: monitors}
}

// GetStats builds the whole dashboard in one call.
//
// This is an aggregate endpoint rather than pure REST (ADR-005): the dashboard
// needs six pieces of data, and five separate requests would mean five round
// trips, five loading spinners and five ways to fail on one screen.
func (s *DashboardService) GetStats(ctx context.Context, userID uuid.UUID) (*DashboardStats, error) {
	counts, err := s.stats.CountMonitors(ctx, userID)
	if err != nil {
		return nil, err
	}

	uptime, err := s.stats.UptimePercentage24h(ctx, userID)
	if err != nil {
		return nil, err
	}

	incidents, err := s.incidents.ListRecentForUser(ctx, userID, 5)
	if err != nil {
		return nil, err
	}

	recentChecks, err := s.stats.RecentChecksForUser(ctx, userID, 10)
	if err != nil {
		return nil, err
	}

	// Return empty slices rather than nil. A nil slice marshals to JSON `null`,
	// and the frontend would then have to guard every `.map()` against null.
	// An empty array is the honest representation of "none", and it keeps the
	// client code simple.
	if incidents == nil {
		incidents = []models.Incident{}
	}
	if recentChecks == nil {
		recentChecks = []models.Check{}
	}

	return &DashboardStats{
		TotalMonitors:       counts.Total,
		MonitorsUp:          counts.Up,
		MonitorsDown:        counts.Down,
		UptimePercentage24h: uptime,
		RecentIncidents:     incidents,
		RecentChecks:        recentChecks,
	}, nil
}

// MonitorDetail is one monitor plus its recent history.
type MonitorDetail struct {
	Monitor   models.Monitor    `json:"monitor"`
	Checks    []models.Check    `json:"checks"`
	Incidents []models.Incident `json:"incidents"`
	Uptime24h *float64          `json:"uptime_percentage_24h"`
}

// GetMonitorDetail returns a monitor the user owns, with its history.
func (s *DashboardService) GetMonitorDetail(
	ctx context.Context, monitorID, userID uuid.UUID, checkLimit int,
) (*MonitorDetail, error) {
	// Ownership is proved here first. Everything below is only reached if the
	// monitor genuinely belongs to this user; otherwise this returns
	// ErrNotFound and we never touch the history tables.
	monitor, err := s.monitors.GetByIDForUser(ctx, monitorID, userID)
	if err != nil {
		return nil, err
	}

	since := time.Now().UTC().Add(-24 * time.Hour)

	checks, err := s.checks.ListByMonitorForUser(ctx, monitorID, userID, checkLimit, since)
	if err != nil {
		return nil, fmt.Errorf("loading checks: %w", err)
	}

	incidents, err := s.incidents.ListByMonitorForUser(ctx, monitorID, userID, 20)
	if err != nil {
		return nil, fmt.Errorf("loading incidents: %w", err)
	}

	uptime, err := s.stats.UptimeForMonitor(ctx, monitorID, since)
	if err != nil {
		return nil, err
	}

	if checks == nil {
		checks = []models.Check{}
	}
	if incidents == nil {
		incidents = []models.Incident{}
	}

	return &MonitorDetail{
		Monitor:   *monitor,
		Checks:    checks,
		Incidents: incidents,
		Uptime24h: uptime,
	}, nil
}
