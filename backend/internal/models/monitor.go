package models

import (
	"time"

	"github.com/google/uuid"
)

// MonitorStatus is a monitor's current health.
//
// A named string type rather than a plain string: the compiler then stops you
// passing an arbitrary string where a status is expected.
type MonitorStatus string

const (
	// StatusPending means the monitor exists but has never been checked.
	StatusPending MonitorStatus = "pending"
	StatusUp      MonitorStatus = "up"
	StatusDown    MonitorStatus = "down"
)

// Monitor is one URL being watched.
type Monitor struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// Ownership runs Monitor -> Project -> User. There is deliberately no
	// UserID here: two places recording the owner could disagree, and a
	// disagreement about ownership is a cross-tenant data leak. See ADR-007.
	ProjectID uuid.UUID `gorm:"type:uuid;not null;index" json:"project_id"`

	// ---- the seven settings the user controls ----
	Name            string `gorm:"not null" json:"name"`
	URL             string `gorm:"not null" json:"url"`
	Method          string `gorm:"not null;default:GET" json:"method"`
	IntervalSeconds int    `gorm:"not null;default:60" json:"interval_seconds"`
	TimeoutSeconds  int    `gorm:"not null;default:10" json:"timeout_seconds"`
	ExpectedStatus  int    `gorm:"not null;default:200" json:"expected_status"`
	Enabled         bool   `gorm:"not null;default:true" json:"enabled"`

	// ---- state the app maintains, not the user ----

	// Status lets the dashboard count up/down instantly instead of scanning
	// every row in checks.
	Status MonitorStatus `gorm:"not null;default:pending" json:"status"`

	// LastCheckedAt is the scheduler's memory. It is a pointer so it can be
	// nil, meaning "never checked" - a plain time.Time would default to the
	// year 1 and pretend the monitor had been checked long ago.
	LastCheckedAt *time.Time `json:"last_checked_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Monitor) TableName() string {
	return "monitors"
}

// AllowedMethods is the HTTP method set a monitor may use.
// It mirrors the CHECK constraint in migration 000001.
var AllowedMethods = []string{"GET", "POST", "HEAD"}

// IsMethodAllowed reports whether method is one we accept.
func IsMethodAllowed(method string) bool {
	for _, m := range AllowedMethods {
		if m == method {
			return true
		}
	}
	return false
}
