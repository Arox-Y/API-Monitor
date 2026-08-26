package models

import (
	"time"

	"github.com/google/uuid"
)

// Incident is a continuous period during which a monitor was failing.
//
// Incidents are created by TRANSITIONS, not by states. Only the moment a
// monitor goes from healthy to failing opens one. An hour of downtime checked
// every minute therefore produces ONE incident, not sixty.
//
// The database enforces this too: idx_incidents_one_open is a partial unique
// index guaranteeing at most one row per monitor where resolved_at IS NULL.
type Incident struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MonitorID uuid.UUID `gorm:"type:uuid;not null;index" json:"monitor_id"`

	StartedAt time.Time `gorm:"not null" json:"started_at"`

	// ResolvedAt is nil while the incident is STILL OPEN - the monitor is
	// down right now.
	ResolvedAt *time.Time `json:"resolved_at"`

	// DurationSeconds is nil until the incident closes.
	//
	// It is stored rather than computed from the timestamps because it is read
	// on every incident list and written exactly once. A closed incident never
	// changes, so this denormalisation cannot drift out of sync.
	DurationSeconds *int `json:"duration_seconds"`

	Cause string `gorm:"not null;default:''" json:"cause"`

	// MonitorName is populated by a join so an incident list renders without
	// a follow-up request per row. Read-only (`->`), not ignored (`-`).
	MonitorName string `gorm:"->" json:"monitor_name,omitempty"`
}

func (Incident) TableName() string {
	return "incidents"
}

// IsOpen reports whether the incident is still ongoing.
func (i Incident) IsOpen() bool {
	return i.ResolvedAt == nil
}
