// Package service holds the business rules.
//
// This layer decides WHAT should happen. It does not know about HTTP, and it
// does not write SQL. It sits between handlers (which speak HTTP) and
// repositories (which speak SQL).
package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"

	"github.com/ariyam/api-monitor/internal/apperr"
	"github.com/ariyam/api-monitor/internal/auth"
	"github.com/ariyam/api-monitor/internal/models"
)

// UserRepository is what AuthService needs from storage.
//
// Note WHERE this interface lives: in the package that USES it, not the one
// that implements it. That is the Go convention, and it has a real benefit -
// this package depends on a small description of what it needs, not on GORM.
//
// A test can supply a fake implementation of these three methods and exercise
// every rule below with no database running.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// AuthService handles registration and login.
type AuthService struct {
	users  UserRepository
	tokens *auth.TokenService
}

// NewAuthService builds an AuthService from its dependencies.
func NewAuthService(users UserRepository, tokens *auth.TokenService) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

// RegisterInput is what registration needs. A named struct beats four string
// parameters: at a call site, Register(a, b, c) gives no clue which is which.
type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

// Register creates an account and returns the user plus a login token.
//
// Returning three values (user, token, error) is normal in Go. The caller must
// check the error before touching the others.
func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*models.User, string, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	if err := validateRegister(in); err != nil {
		return nil, "", err
	}

	// Hash before storing. The plain password exists only inside this function
	// and is never written anywhere - not the database, not a log.
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		// Translate the auth package's length errors into field errors the
		// frontend can attach to the right input box.
		if errors.Is(err, auth.ErrPasswordTooShort) || errors.Is(err, auth.ErrPasswordTooLong) {
			return nil, "", apperr.NewValidationError(
				apperr.FieldError{Field: "password", Message: err.Error()},
			)
		}
		return nil, "", fmt.Errorf("hashing password: %w", err)
	}

	user := &models.User{
		Name:         in.Name,
		Email:        in.Email,
		PasswordHash: hash,
	}

	// We do NOT check "is this email taken?" first. Between that check and the
	// insert, another request could register the same address. The database's
	// UNIQUE constraint is the only reliable arbiter, so we insert and handle
	// the rejection. This is check-then-act, and avoiding it is a real habit
	// worth forming.
	if err := s.users.Create(ctx, user); err != nil {
		return nil, "", err // already ErrEmailTaken if that was the cause
	}

	token, err := s.tokens.Generate(user.ID)
	if err != nil {
		return nil, "", fmt.Errorf("generating token: %w", err)
	}

	return user, token, nil
}

// LoginInput is what login needs.
type LoginInput struct {
	Email    string
	Password string
}

// Login verifies credentials and returns the user plus a fresh token.
func (s *AuthService) Login(ctx context.Context, in LoginInput) (*models.User, string, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	user, err := s.users.GetByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			// SECURITY: an unknown email returns the SAME error as a wrong
			// password. Distinguishing them would let anyone test which email
			// addresses have accounts here.
			//
			// We also still run a bcrypt comparison below in that case, so the
			// response takes the same amount of time either way. Without that,
			// an attacker could measure the difference - a fast reply meaning
			// "no such user", a slow one meaning "user exists, wrong password".
			auth.CheckPassword(in.Password, dummyHash)
			return nil, "", apperr.ErrInvalidCredentials
		}
		return nil, "", fmt.Errorf("looking up user: %w", err)
	}

	if !auth.CheckPassword(in.Password, user.PasswordHash) {
		return nil, "", apperr.ErrInvalidCredentials
	}

	token, err := s.tokens.Generate(user.ID)
	if err != nil {
		return nil, "", fmt.Errorf("generating token: %w", err)
	}

	return user, token, nil
}

// GetByID loads a user, for the "who am I?" endpoint.
func (s *AuthService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.users.GetByID(ctx, id)
}

// dummyHash is a real bcrypt hash of a throwaway value.
//
// Its only purpose is to burn the same ~250ms when the email does not exist,
// so login timing cannot reveal whether an account is registered. The value it
// hashes is irrelevant and it is never compared against anything real.
const dummyHash = "$2a$12$C6UzMDM.H6dfI/f/IKcEe.7Zt8dEQiVjzYIYnUvVYbLDvhWMxRVBK"

func validateRegister(in RegisterInput) error {
	var fields []apperr.FieldError

	if in.Name == "" {
		fields = append(fields, apperr.FieldError{
			Field: "name", Message: "is required",
		})
	} else if len(in.Name) > 100 {
		fields = append(fields, apperr.FieldError{
			Field: "name", Message: "must be at most 100 characters",
		})
	}

	if in.Email == "" {
		fields = append(fields, apperr.FieldError{
			Field: "email", Message: "is required",
		})
	} else if len(in.Email) > 255 {
		fields = append(fields, apperr.FieldError{
			Field: "email", Message: "must be at most 255 characters",
		})
	} else if _, err := mail.ParseAddress(in.Email); err != nil {
		// net/mail is in the standard library and understands the real address
		// format. A hand-written regex for email addresses is a classic
		// mistake - the actual specification is far more complex than it looks.
		fields = append(fields, apperr.FieldError{
			Field: "email", Message: "must be a valid email address",
		})
	}

	if in.Password == "" {
		fields = append(fields, apperr.FieldError{
			Field: "password", Message: "is required",
		})
	}

	if len(fields) > 0 {
		return apperr.NewValidationError(fields...)
	}
	return nil
}
