// Package apperr defines the errors our business logic can produce.
//
// These describe WHAT went wrong in domain terms ("email already taken"),
// never HOW it should be reported over HTTP. The service layer returns these;
// a single place in the handler layer translates them into status codes.
//
// Keeping HTTP out of the service layer means the services could be reused by
// a CLI or a background job without dragging web concepts along.
package apperr

import "errors"

// Sentinel errors: named values callers compare against with errors.Is().
//
// Comparing against a named error is reliable. Matching on error text is not:
// rewording a message would silently break every caller.
var (
	// ErrNotFound also covers "it exists but is not yours".
	//
	// We deliberately do not distinguish those two cases. Returning 403 for
	// someone else's resource would CONFIRM that it exists, letting an
	// attacker discover valid IDs by trying them. See ADR-009.
	ErrNotFound = errors.New("not found")

	ErrEmailTaken = errors.New("email already registered")

	// ErrInvalidCredentials is returned for BOTH a wrong password and an
	// unknown email. Saying "no such user" would tell an attacker which email
	// addresses have accounts.
	ErrInvalidCredentials = errors.New("invalid credentials")

	ErrUnauthorized = errors.New("unauthorized")
)

// FieldError is one problem with one input field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError collects everything wrong with a request.
//
// We report ALL the problems at once rather than stopping at the first, so the
// user fixes their form in one go instead of playing whack-a-mole.
type ValidationError struct {
	Fields []FieldError
}

// Error makes ValidationError satisfy Go's built-in error interface.
//
// An interface in Go is satisfied implicitly: there is no "implements"
// keyword. Any type with an Error() string method IS an error, automatically.
func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return "validation failed"
	}
	return "validation failed: " + e.Fields[0].Field + " " + e.Fields[0].Message
}

// NewValidationError builds a validation error from field/message pairs.
func NewValidationError(fields ...FieldError) *ValidationError {
	return &ValidationError{Fields: fields}
}
