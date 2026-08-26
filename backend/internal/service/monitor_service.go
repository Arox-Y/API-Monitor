package service

import (
	"context"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/ariyam/api-monitor/internal/apperr"
	"github.com/ariyam/api-monitor/internal/models"
)

// Bounds for monitor settings. These mirror the CHECK constraints in
// migration 000001 - the application rejects bad values with a helpful
// message, and the database rejects them again as a last line of defence.
const (
	MinIntervalSeconds = 30
	MaxIntervalSeconds = 86400 // 24 hours
	MinTimeoutSeconds  = 1
	MaxTimeoutSeconds  = 30
	MinExpectedStatus  = 100
	MaxExpectedStatus  = 599
)

// MonitorRepository is what MonitorService needs from storage.
type MonitorRepository interface {
	Create(ctx context.Context, monitor *models.Monitor) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Monitor, error)
	ListByProject(ctx context.Context, projectID, userID uuid.UUID) ([]models.Monitor, error)
	GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*models.Monitor, error)
	Update(ctx context.Context, monitor *models.Monitor, userID uuid.UUID) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// MonitorService holds the monitor rules.
type MonitorService struct {
	monitors MonitorRepository
	projects ProjectRepository
}

func NewMonitorService(monitors MonitorRepository, projects ProjectRepository) *MonitorService {
	return &MonitorService{monitors: monitors, projects: projects}
}

// CreateMonitorInput is what creating a monitor needs.
//
// The numeric fields are pointers so "not supplied" can be told from "zero",
// letting us apply sensible defaults rather than treating 0 as a real choice.
type CreateMonitorInput struct {
	Name            string
	URL             string
	Method          string
	IntervalSeconds *int
	TimeoutSeconds  *int
	ExpectedStatus  *int
	Enabled         *bool
}

// Create adds a monitor to a project the user owns.
func (s *MonitorService) Create(ctx context.Context, projectID, userID uuid.UUID, in CreateMonitorInput) (*models.Monitor, error) {
	// AUTHORISATION FIRST.
	//
	// Confirm the target project belongs to this user before anything else.
	// Without this check, anyone could post a monitor into someone else's
	// project just by knowing its ID.
	if _, err := s.projects.GetByIDForUser(ctx, projectID, userID); err != nil {
		return nil, err // ErrNotFound for both "no such project" and "not yours"
	}

	monitor := &models.Monitor{
		ProjectID:       projectID,
		Name:            strings.TrimSpace(in.Name),
		URL:             strings.TrimSpace(in.URL),
		Method:          strings.ToUpper(strings.TrimSpace(in.Method)),
		IntervalSeconds: valueOr(in.IntervalSeconds, 60),
		TimeoutSeconds:  valueOr(in.TimeoutSeconds, 10),
		ExpectedStatus:  valueOr(in.ExpectedStatus, 200),
		Enabled:         valueOr(in.Enabled, true),
		Status:          models.StatusPending,
	}

	if monitor.Method == "" {
		monitor.Method = "GET"
	}

	if err := validateMonitor(monitor); err != nil {
		return nil, err
	}

	if err := s.monitors.Create(ctx, monitor); err != nil {
		return nil, err
	}

	return monitor, nil
}

// List returns every monitor the user owns.
func (s *MonitorService) List(ctx context.Context, userID uuid.UUID) ([]models.Monitor, error) {
	return s.monitors.ListByUser(ctx, userID)
}

// ListByProject returns the monitors in one of the user's projects.
func (s *MonitorService) ListByProject(ctx context.Context, projectID, userID uuid.UUID) ([]models.Monitor, error) {
	if _, err := s.projects.GetByIDForUser(ctx, projectID, userID); err != nil {
		return nil, err
	}
	return s.monitors.ListByProject(ctx, projectID, userID)
}

// Get returns one monitor the user owns.
func (s *MonitorService) Get(ctx context.Context, id, userID uuid.UUID) (*models.Monitor, error) {
	return s.monitors.GetByIDForUser(ctx, id, userID)
}

// UpdateMonitorInput carries only the fields a client may change.
// All pointers, so an omitted field means "leave it alone".
type UpdateMonitorInput struct {
	Name            *string
	URL             *string
	Method          *string
	IntervalSeconds *int
	TimeoutSeconds  *int
	ExpectedStatus  *int
	Enabled         *bool
}

// Update changes a monitor the user owns.
//
// Enabling and disabling goes through here too, with {"enabled": false}. It is
// a field update, and REST already has a verb for that - a separate /pause
// endpoint would be inventing a second way to do the same thing.
func (s *MonitorService) Update(ctx context.Context, id, userID uuid.UUID, in UpdateMonitorInput) (*models.Monitor, error) {
	monitor, err := s.monitors.GetByIDForUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		monitor.Name = strings.TrimSpace(*in.Name)
	}
	if in.URL != nil {
		monitor.URL = strings.TrimSpace(*in.URL)
	}
	if in.Method != nil {
		monitor.Method = strings.ToUpper(strings.TrimSpace(*in.Method))
	}
	if in.IntervalSeconds != nil {
		monitor.IntervalSeconds = *in.IntervalSeconds
	}
	if in.TimeoutSeconds != nil {
		monitor.TimeoutSeconds = *in.TimeoutSeconds
	}
	if in.ExpectedStatus != nil {
		monitor.ExpectedStatus = *in.ExpectedStatus
	}
	if in.Enabled != nil {
		monitor.Enabled = *in.Enabled
	}

	if err := validateMonitor(monitor); err != nil {
		return nil, err
	}

	if err := s.monitors.Update(ctx, monitor, userID); err != nil {
		return nil, err
	}

	return s.monitors.GetByIDForUser(ctx, id, userID)
}

// Delete removes a monitor the user owns.
func (s *MonitorService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.monitors.Delete(ctx, id, userID)
}

func validateMonitor(m *models.Monitor) error {
	var fields []apperr.FieldError

	if m.Name == "" {
		fields = append(fields, apperr.FieldError{Field: "name", Message: "is required"})
	} else if len(m.Name) > 100 {
		fields = append(fields, apperr.FieldError{Field: "name", Message: "must be at most 100 characters"})
	}

	fields = append(fields, validateMonitorURL(m.URL)...)

	if !models.IsMethodAllowed(m.Method) {
		fields = append(fields, apperr.FieldError{
			Field: "method", Message: "must be one of GET, POST, HEAD",
		})
	}

	if m.IntervalSeconds < MinIntervalSeconds || m.IntervalSeconds > MaxIntervalSeconds {
		fields = append(fields, apperr.FieldError{
			Field:   "interval_seconds",
			Message: "must be between 30 and 86400",
		})
	}

	if m.TimeoutSeconds < MinTimeoutSeconds || m.TimeoutSeconds > MaxTimeoutSeconds {
		fields = append(fields, apperr.FieldError{
			Field:   "timeout_seconds",
			Message: "must be between 1 and 30",
		})
	}

	if m.ExpectedStatus < MinExpectedStatus || m.ExpectedStatus > MaxExpectedStatus {
		fields = append(fields, apperr.FieldError{
			Field:   "expected_status",
			Message: "must be between 100 and 599",
		})
	}

	// CROSS-FIELD RULE.
	//
	// Each field above can be valid on its own while the pair is nonsense: a
	// 30-second timeout on a 30-second interval means the next check starts
	// before the previous one has given up.
	//
	// No struct tag can express this - it needs real code, which is exactly
	// why cross-field validation is so often forgotten.
	if m.TimeoutSeconds >= m.IntervalSeconds {
		fields = append(fields, apperr.FieldError{
			Field:   "timeout_seconds",
			Message: "must be less than interval_seconds",
		})
	}

	if len(fields) > 0 {
		return apperr.NewValidationError(fields...)
	}
	return nil
}

func validateMonitorURL(raw string) []apperr.FieldError {
	if raw == "" {
		return []apperr.FieldError{{Field: "url", Message: "is required"}}
	}
	if len(raw) > 2000 {
		return []apperr.FieldError{{Field: "url", Message: "must be at most 2000 characters"}}
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return []apperr.FieldError{{Field: "url", Message: "is not a valid URL"}}
	}

	// SSRF mitigation, level 1 (ADR-012).
	//
	// Our server fetches whatever URL the user supplies, which is the textbook
	// setup for Server-Side Request Forgery. Restricting the scheme blocks the
	// worst of it: file:// would read our own disk, and gopher:// can be used
	// to talk to internal services.
	//
	// Level 2 - rejecting private and loopback IP addresses like 127.0.0.1 and
	// 169.254.169.254 - is scheduled for Day 5 and is the more important half.
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return []apperr.FieldError{{Field: "url", Message: "must start with http:// or https://"}}
	}

	if parsed.Host == "" {
		return []apperr.FieldError{{Field: "url", Message: "must include a hostname"}}
	}

	return nil
}

// valueOr returns *p, or fallback when p is nil.
//
// A tiny generic helper. [T any] means it works for any type, so one function
// covers *int, *bool and *string instead of three near-identical copies.
func valueOr[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}
