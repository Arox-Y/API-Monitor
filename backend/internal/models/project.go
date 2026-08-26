package models

import (
	"time"

	"github.com/google/uuid"
)

// Project groups monitors together, e.g. "Acme API".
//
// A project may expose a public status page, which is why it carries a slug
// and an is_public flag.
type Project struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"-"`

	Name        string `gorm:"not null" json:"name"`
	Description string `gorm:"not null;default:''" json:"description"`

	// Slug is the readable identifier in the public URL: /status/acme-api.
	// Server-generated and globally unique - a client must never choose it,
	// or one user could claim /status/stripe.
	Slug string `gorm:"uniqueIndex;not null" json:"slug"`

	IsPublic bool `gorm:"not null;default:false" json:"is_public"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// MonitorCount is computed by a subquery, not stored as a column.
	//
	// `gorm:"->"` means READ-ONLY: GORM will populate it when a query selects
	// a matching column, but never try to write it back.
	//
	// It must NOT be `gorm:"-"`, which means "ignore this field completely" -
	// GORM then discards the value and the count silently arrives as 0.
	// That exact bug shipped here and was caught by testing.
	MonitorCount int `gorm:"->" json:"monitor_count"`
}

// TableName pins the table, so renaming the Go type cannot silently change
// which table we query.
func (Project) TableName() string {
	return "projects"
}

// UserID is json:"-" on purpose.
//
// The owner's ID is of no use to the owner, and exposing internal identifiers
// gives an attacker information for free. The API only ever returns a user's
// own projects, so the field would be redundant even if it were harmless.
