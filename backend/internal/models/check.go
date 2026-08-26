package models

import (
	"time"

	"github.com/google/uuid"
)

// Check is the result of ONE visit to a monitored URL.
//
// This is the highest-volume table in the system: one monitor on a 60-second
// interval produces about 1,440 rows a day.
//
// A check is an immutable historical fact, which is why there is no UpdatedAt.
// It records what happened at a moment in time and is never revised.
type Check struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MonitorID uuid.UUID `gorm:"type:uuid;not null;index" json:"-"`

	CheckedAt time.Time `gorm:"not null" json:"checked_at"`
	Success   bool      `gorm:"not null" json:"success"`

	// StatusCode is a POINTER so it can be nil.
	//
	// A timeout, a DNS failure or a refused connection produces no HTTP
	// response at all, so there is no status code to record. nil says that
	// honestly. Storing 0 would be a lie, and storing 200 would be a disaster.
	//
	// This is the clearest example in the codebase of why Go uses pointers for
	// optional values: an int cannot express "no value", it can only be 0.
	StatusCode *int `json:"status_code"`

	ResponseTimeMs int    `gorm:"not null" json:"response_time_ms"`
	Error          string `gorm:"not null;default:''" json:"error"`

	// MonitorName is populated by a join for dashboard listings, not stored.
	// `gorm:"->"` = read-only. `gorm:"-"` would make GORM discard the joined
	// value entirely, leaving this empty.
	MonitorName string `gorm:"->" json:"monitor_name,omitempty"`
}

func (Check) TableName() string {
	return "checks"
}
