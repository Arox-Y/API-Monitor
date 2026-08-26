package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Errors a caller can react to specifically.
//
// The handler turns all of these into the same 401 response. We keep them
// separate anyway so logs can say WHY a token was rejected, which is the
// difference between "someone's session expired" and "someone is forging
// tokens".
var (
	ErrInvalidToken = errors.New("token is invalid")
	ErrExpiredToken = errors.New("token has expired")
)

// DefaultTokenTTL is how long a login lasts before the user must sign in again.
//
// A trade-off: shorter is safer (a stolen token stops working sooner) but more
// annoying. Real systems solve this with a short access token plus a long
// refresh token. We deliberately use one 24-hour token for the MVP and say so
// out loud rather than pretending it is ideal.
const DefaultTokenTTL = 24 * time.Hour

// TokenService creates and verifies login tokens.
//
// The secret lives in this struct rather than in a package-level variable, so
// a test can build its own service with its own secret and nothing is shared
// or hidden. This is dependency injection: give a thing what it needs instead
// of letting it reach out for globals.
type TokenService struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenService builds a TokenService.
//
// It returns an error rather than panicking on a bad secret, so the caller
// (main) can fail to start with a clear message.
func NewTokenService(secret string, ttl time.Duration) (*TokenService, error) {
	// A short secret can be brute-forced offline: an attacker with any token
	// from our system can guess secrets until the signature matches, then
	// forge a token for any user they like.
	if len(secret) < 32 {
		return nil, fmt.Errorf("jwt secret must be at least 32 characters, got %d", len(secret))
	}
	// JWT timestamps have ONE-SECOND precision: the spec stores exp and iat as
	// whole seconds since the epoch. Anything under a second rounds away, so a
	// 500ms TTL silently produces tokens that are already expired when issued.
	//
	// Found by a test: a 50ms TTL made Verify fail immediately after Generate.
	// Rather than let callers configure something the format cannot express,
	// refuse it here with a clear message.
	if ttl < time.Second {
		return nil, fmt.Errorf(
			"token ttl must be at least 1 second (jwt timestamps are whole seconds), got %v", ttl)
	}

	return &TokenService{secret: []byte(secret), ttl: ttl}, nil
}

// Generate creates a signed token for a user.
//
// The payload contains ONLY the user ID. Remember: a JWT is signed, not
// encrypted — anyone holding it can read the payload. Nothing private goes in.
func (s *TokenService) Generate(userID uuid.UUID) (string, error) {
	now := time.Now()

	// RegisteredClaims are the standard fields defined by the JWT spec.
	//   Subject   ("sub") — who this token is about
	//   IssuedAt  ("iat") — when it was created
	//   ExpiresAt ("exp") — when it stops being valid
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
	}

	// HS256 signs with one shared secret: the same key both creates and
	// verifies. That suits us because one service does both.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return signed, nil
}

// Verify checks a token and returns the user ID inside it.
//
// It fails if the token was tampered with, signed by a different secret, has
// expired, or is simply malformed.
func (s *TokenService) Verify(tokenString string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}

	_, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		// This callback supplies the key to verify against. It also gets the
		// chance to inspect the token's own header first.
		func(token *jwt.Token) (any, error) {
			// CRITICAL CHECK.
			//
			// The token itself says which algorithm it used, in a header the
			// attacker controls. If we blindly trusted it, someone could send
			// a token saying "alg: none" and the library might accept it with
			// no signature at all. That is a real, famous JWT vulnerability.
			//
			// So we insist on the algorithm WE chose, ignoring what the token
			// claims about itself.
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.secret, nil
		},
		// Belt and braces: restrict accepted algorithms at the parser level too.
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)

	if err != nil {
		// Distinguish "expired" from "invalid" for logging. errors.Is walks
		// the whole wrapped error chain rather than comparing just the outer
		// error, which is why wrapping with %w matters.
		if errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.Nil, ErrExpiredToken
		}
		return uuid.Nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// The signature is valid, but the contents still have to make sense.
	// A correctly signed token could still carry a subject that is not a UUID.
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: subject is not a valid uuid", ErrInvalidToken)
	}

	return userID, nil
}
