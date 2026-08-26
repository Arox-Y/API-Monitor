// Package models holds the structs that describe our data.
//
// These mirror the database tables created in backend/migrations/. This
// package imports nothing from the rest of the app — everything depends on
// models, and models depends on nothing.
package models

import (
	"time"

	"github.com/google/uuid"
)

// User is one account.
//
// The backtick text after each field is a "struct tag" — a note attached to
// the field that libraries read at runtime:
//
//	gorm:"..."  tells GORM how the field maps to a database column
//	json:"..."  tells Go's JSON encoder what to call it in API responses
type User struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// Always stored lowercase. PostgreSQL's UNIQUE is case-sensitive, so
	// without normalising first, Bob@x.com and bob@x.com would both be
	// accepted as separate accounts.
	Email string `gorm:"uniqueIndex;not null" json:"email"`

	// json:"-" means "never include this in JSON output".
	//
	// This single character is the difference between a safe API and one that
	// leaks every password hash in every response. Even though a bcrypt hash
	// cannot be reversed easily, handing attackers the hashes lets them crack
	// them offline at their leisure.
	PasswordHash string `gorm:"not null" json:"-"`

	Name string `gorm:"not null" json:"name"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName tells GORM which table this struct belongs to.
//
// GORM would guess "users" correctly here, but stating it explicitly means a
// future rename of the Go type cannot silently change which table we query.
func (User) TableName() string {
	return "users"
}
