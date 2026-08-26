// Package repository talks to the database. Nothing else does.
//
// Every function here does one job: run a query. No business rules, no
// permission checks, no HTTP. That separation is what lets the service layer
// be tested with a fake repository and no database at all.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ariyam/api-monitor/internal/apperr"
	"github.com/ariyam/api-monitor/internal/models"
)

// UserRepository runs user queries against PostgreSQL.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository builds a UserRepository.
//
// The database handle is passed IN rather than fetched from a global. That is
// dependency injection, and it is what makes this replaceable in a test.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user.
//
// ctx is the first parameter, as it is for every function that does I/O. If
// the caller gives up - the browser disconnects, a timeout fires - the
// database query is abandoned rather than finishing for nobody.
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	// Normalise before storing. PostgreSQL's UNIQUE is case-sensitive, so
	// without this "Bob@x.com" and "bob@x.com" would be two separate accounts.
	user.Email = normalizeEmail(user.Email)

	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		// The database has the final say on uniqueness. Checking "does this
		// email exist?" first and then inserting looks fine but is a race:
		// two simultaneous signups can both pass the check, and only the
		// UNIQUE constraint stops the duplicate.
		//
		// So we attempt the insert and interpret the failure.
		if isUniqueViolation(err) {
			return apperr.ErrEmailTaken
		}
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

// GetByEmail finds a user by email address, or returns apperr.ErrNotFound.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User

	err := r.db.WithContext(ctx).
		Where("email = ?", normalizeEmail(email)).
		First(&user).Error

	if err != nil {
		// GORM reports "no rows" as its own error. We translate it into our
		// domain error so the service layer never needs to know GORM exists.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("finding user by email: %w", err)
	}

	// Return a pointer so callers share one user rather than copying it.
	return &user, nil
}

// GetByID finds a user by ID, or returns apperr.ErrNotFound.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User

	// The ? is a placeholder. GORM sends the value separately from the SQL
	// text, so a malicious value can never be executed as SQL. This is how
	// SQL injection is prevented - never build queries by joining strings.
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("finding user by id: %w", err)
	}

	return &user, nil
}

// normalizeEmail lowercases and trims an address so comparisons are reliable.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// isUniqueViolation reports whether an error is PostgreSQL's "duplicate key".
//
// PostgreSQL error code 23505 is unique_violation. Matching the code would be
// more precise than matching text, but that needs a driver-specific error type;
// the driver surfaces the code inside the message, so we check for both the
// code and the wording.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "23505") ||
		strings.Contains(msg, "duplicate key value violates unique constraint")
}
