// Package middleware holds the code that runs around every request.
//
// Middleware is a function that wraps a handler. Gin runs them in order, and
// each may either pass the request along or stop it dead. Authentication is
// the classic example: check the token once, here, instead of repeating the
// same check at the top of forty handlers and eventually forgetting one.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ariyam/api-monitor/internal/auth"
)

// contextUserIDKey is where the authenticated user ID is stashed.
//
// Lowercase, so no other package can read or write it directly. They must go
// through UserIDFromContext below, which also checks the type. That keeps one
// typo from silently breaking authentication everywhere.
const contextUserIDKey = "auth.userID"

// RequireAuth rejects any request without a valid session cookie.
//
// It returns a gin.HandlerFunc rather than being one. That lets us configure
// it with a TokenService at wiring time - a closure capturing its dependency,
// which is dependency injection for functions.
func RequireAuth(tokens *auth.TokenService, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie(cookieName)
		if err != nil || tokenString == "" {
			// No cookie at all: not signed in.
			unauthorized(c)
			return
		}

		userID, err := tokens.Verify(tokenString)
		if err != nil {
			// Expired, tampered with, signed by another secret, or malformed.
			//
			// The client is told none of that - every case returns the same
			// 401. Explaining WHY a token was rejected helps an attacker tune
			// their forgery attempts.
			unauthorized(c)
			return
		}

		// Hand the verified user ID to the handlers further down the chain.
		//
		// Everything after this point can trust it, because a request only
		// gets here if the signature checked out.
		c.Set(contextUserIDKey, userID)

		// c.Next() runs the rest of the chain. Without it the request stops
		// here and the client waits forever - a genuinely easy mistake.
		c.Next()
	}
}

// UserIDFromContext returns the authenticated user's ID.
//
// The bool is false when there is none, following Go's "comma ok" convention.
// Returning two values forces the caller to acknowledge the missing case,
// rather than silently receiving a zero UUID and treating it as a real user.
func UserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get(contextUserIDKey)
	if !exists {
		return uuid.Nil, false
	}

	// c.Get returns `any`, so the concrete type must be asserted back. If
	// something else ever stored a different type under this key, we return
	// false instead of panicking.
	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, false
	}

	return userID, true
}

func unauthorized(c *gin.Context) {
	// AbortWithStatusJSON stops the chain AND writes the response. Using plain
	// c.JSON here would send the body but still run the remaining handlers,
	// which would then try to write a second response.
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{
			"code":    "UNAUTHORIZED",
			"message": "Authentication required.",
		},
	})
}
