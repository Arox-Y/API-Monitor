// Package handler is the HTTP layer.
//
// Handlers parse requests, call a service, and turn the result into a
// response. They contain no business rules and no SQL.
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ariyam/api-monitor/internal/apperr"
)

// errorBody is the shape of EVERY error this API returns.
//
//	{"error": {"code": "...", "message": "...", "details": [...]}}
//
// One predictable shape means the frontend writes ONE error handler instead of
// a different one per endpoint.
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	// Code is for machines: stable, safe to switch on.
	Code string `json:"code"`
	// Message is for humans and may be reworded freely.
	//
	// Keeping these separate matters: if they were one field, improving the
	// wording would break any client that checked it.
	Message string              `json:"message"`
	Details []apperr.FieldError `json:"details,omitempty"`
}

// respondError writes an error using the standard envelope.
func respondError(c *gin.Context, status int, code, message string, details ...apperr.FieldError) {
	c.JSON(status, errorBody{Error: errorDetail{
		Code:    code,
		Message: message,
		Details: details,
	}})
}

// respondWithDomainError maps a service-layer error to an HTTP response.
//
// This mapping exists in ONE place. Repeating it per handler would guarantee
// that some endpoint eventually returns the wrong status, or leaks an internal
// message.
func respondWithDomainError(c *gin.Context, err error) {
	var validationErr *apperr.ValidationError

	switch {
	case errors.As(err, &validationErr):
		// errors.As unwraps until it finds an error of this concrete type,
		// then fills in the pointer so we can read its fields.
		respondError(c, http.StatusBadRequest,
			"VALIDATION_FAILED", "The request could not be validated.",
			validationErr.Fields...)

	case errors.Is(err, apperr.ErrEmailTaken):
		respondError(c, http.StatusConflict,
			"EMAIL_TAKEN", "That email address is already registered.")

	case errors.Is(err, apperr.ErrInvalidCredentials):
		respondError(c, http.StatusUnauthorized,
			"INVALID_CREDENTIALS", "Incorrect email or password.")

	case errors.Is(err, apperr.ErrNotFound):
		respondError(c, http.StatusNotFound,
			"NOT_FOUND", "Not found.")

	case errors.Is(err, apperr.ErrUnauthorized):
		respondError(c, http.StatusUnauthorized,
			"UNAUTHORIZED", "Authentication required.")

	default:
		// Anything unrecognised is a bug on our side.
		//
		// Log the real error for us; tell the client nothing. A raw database
		// error would reveal table names and schema details to an attacker.
		// slog.Default() rather than a per-handler logger: main.go configures
		// the default logger once at startup, and every package that has no
		// reason to carry its own logger reference can just use it.
		//
		// Only the method, path and error are logged - never headers or the
		// request body, which could carry a password on a failed login or a
		// cookie value. That is also why this is the ONLY place an internal
		// error's real text is written down anywhere: the response the
		// client receives never contains it (see the default case below).
		slog.Error("unhandled error", "method", c.Request.Method, "path", c.Request.URL.Path, "error", err)
		respondError(c, http.StatusInternalServerError,
			"INTERNAL_ERROR", "Something went wrong on our end.")
	}
}
