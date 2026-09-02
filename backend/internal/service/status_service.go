package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ariyam/api-monitor/internal/models"
)

// PublicStatus is what an anonymous visitor sees on a status page.
//
// ⚠️ THIS TYPE IS THE SECURITY BOUNDARY. ⚠️
//
// It is written out by hand instead of reusing models.Monitor or
// models.Project, and that is deliberate. Reusing an internal model means
// every future field added to it is exposed to the public internet by default,
// and somebody eventually adds one without thinking about this file.
//
// Here, exposure is opt-IN: a field appears publicly only because someone
// typed it into this struct.
//
// Deliberately ABSENT, and each for a reason:
//   - monitor URLs        reveal internal infrastructure and hostnames
//   - user email / user id  personal data, and useless to a visitor
//   - project / monitor ids internal identifiers invite probing
//   - error strings       leak stack traces, hostnames, library versions
//   - intervals, timeouts configuration is nobody else's business
type PublicStatus struct {
	ProjectName string `json:"project_name"`

	// OverallStatus is up, degraded, or down.
	OverallStatus string `json:"overall_status"`

	// Null when nothing has been measured. Never 0 - see the note in
	// DashboardStats for why that distinction matters.
	UptimePercentage24h *float64 `json:"uptime_percentage_24h"`

	Services        []PublicService  `json:"services"`
	ActiveIncidents []PublicIncident `json:"active_incidents"`
}

// PublicService is one monitored service, stripped of everything internal.
type PublicService struct {
	// Name only. NOT the URL.
	Name                string   `json:"name"`
	Status              string   `json:"status"`
	UptimePercentage24h *float64 `json:"uptime_percentage_24h"`
}

// PublicIncident is an ongoing outage, with no cause string.
//
// The cause is omitted on purpose: internally it holds text like
// "expected status 200, got 503", which reveals what we probe and how.
type PublicIncident struct {
	ServiceName string    `json:"service_name"`
	StartedAt   time.Time `json:"started_at"`
}

// PublicProjectRepository is what StatusService needs to find a public project.
type PublicProjectRepository interface {
	GetPublicBySlug(ctx context.Context, slug string) (*models.Project, error)
}

// PublicMonitorRepository lists the monitors shown on a status page.
type PublicMonitorRepository interface {
	ListByPublicProject(ctx context.Context, projectID uuid.UUID) ([]models.Monitor, error)
}

// PublicIncidentRepository lists unresolved incidents for a project.
type PublicIncidentRepository interface {
	ListOpenByProject(ctx context.Context, projectID uuid.UUID) ([]models.Incident, error)
}

// PublicUptimeRepository provides per-monitor uptime.
type PublicUptimeRepository interface {
	UptimeForMonitor(ctx context.Context, monitorID uuid.UUID, since time.Time) (*float64, error)
}

// StatusService builds the public status page.
type StatusService struct {
	projects  PublicProjectRepository
	monitors  PublicMonitorRepository
	incidents PublicIncidentRepository
	uptime    PublicUptimeRepository
}

func NewStatusService(
	projects PublicProjectRepository,
	monitors PublicMonitorRepository,
	incidents PublicIncidentRepository,
	uptime PublicUptimeRepository,
) *StatusService {
	return &StatusService{projects: projects, monitors: monitors, incidents: incidents, uptime: uptime}
}

// GetBySlug returns the public status for a project, or apperr.ErrNotFound.
//
// There is no userID parameter, because this endpoint is genuinely public.
// The access rule is enforced instead by the repository query, which requires
// is_public = true - so a private project is invisible even to somebody who
// correctly guesses its slug.
func (s *StatusService) GetBySlug(ctx context.Context, slug string) (*PublicStatus, error) {
	project, err := s.projects.GetPublicBySlug(ctx, slug)
	if err != nil {
		return nil, err // ErrNotFound covers both "no such slug" and "not public"
	}

	monitors, err := s.monitors.ListByPublicProject(ctx, project.ID)
	if err != nil {
		return nil, err
	}

	incidents, err := s.incidents.ListOpenByProject(ctx, project.ID)
	if err != nil {
		return nil, err
	}

	since := time.Now().UTC().Add(-24 * time.Hour)

	// Build the public view field by field. Nothing is copied wholesale from
	// an internal model, so nothing can be exposed by accident.
	services := make([]PublicService, 0, len(monitors))
	for _, monitor := range monitors {
		monitorUptime, err := s.uptime.UptimeForMonitor(ctx, monitor.ID, since)
		if err != nil {
			return nil, err
		}

		services = append(services, PublicService{
			Name:                monitor.Name,
			Status:              string(monitor.Status),
			UptimePercentage24h: monitorUptime,
		})
	}

	// Map incidents to their service names, so the page can say WHICH service
	// is affected without exposing any identifier.
	namesByID := make(map[uuid.UUID]string, len(monitors))
	for _, monitor := range monitors {
		namesByID[monitor.ID] = monitor.Name
	}

	activeIncidents := make([]PublicIncident, 0, len(incidents))
	for _, incident := range incidents {
		name, ok := namesByID[incident.MonitorID]
		if !ok {
			// The incident belongs to a disabled monitor, which is not listed
			// on the page. Skipping it keeps the page consistent.
			continue
		}
		activeIncidents = append(activeIncidents, PublicIncident{
			ServiceName: name,
			StartedAt:   incident.StartedAt,
		})
	}

	return &PublicStatus{
		ProjectName:         project.Name,
		OverallStatus:       overallStatus(services),
		UptimePercentage24h: averageUptime(services),
		Services:            services,
		ActiveIncidents:     activeIncidents,
	}, nil
}

// overallStatus summarises many services into one word.
//
//	all up (or none)  -> "up"
//	all down          -> "down"
//	anything mixed    -> "degraded"
//
// "degraded" exists because "down" would be alarming and wrong when nine of
// ten services are fine, while "up" would be a lie about the tenth.
func overallStatus(services []PublicService) string {
	if len(services) == 0 {
		return "up" // nothing monitored is not an outage
	}

	down := 0
	for _, service := range services {
		if service.Status == string(models.StatusDown) {
			down++
		}
	}

	switch {
	case down == 0:
		return "up"
	case down == len(services):
		return "down"
	default:
		return "degraded"
	}
}

// averageUptime is the mean of the services that actually have data.
//
// Services with nil uptime are EXCLUDED rather than counted as zero. Treating
// "not yet measured" as 0% would drag a healthy page's number down for no
// reason. If nothing has been measured at all, the result is nil.
func averageUptime(services []PublicService) *float64 {
	var sum float64
	var counted int

	for _, service := range services {
		if service.UptimePercentage24h != nil {
			sum += *service.UptimePercentage24h
			counted++
		}
	}

	if counted == 0 {
		return nil
	}

	average := sum / float64(counted)
	return &average
}
