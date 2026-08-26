package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ariyam/api-monitor/internal/middleware"
	"github.com/ariyam/api-monitor/internal/service"
)

// SessionCookieName is the cookie the login token travels in.
const SessionCookieName = "am_session"

// AuthHandler serves the /auth endpoints.
type AuthHandler struct {
	auth       *service.AuthService
	cookieTTL  int  // seconds
	secureOnly bool // require HTTPS for the cookie
}

// NewAuthHandler builds an AuthHandler.
//
// secureOnly should be false in local development (no HTTPS) and true in
// production, where a cookie sent over plain HTTP could be intercepted.
func NewAuthHandler(authService *service.AuthService, cookieTTLSeconds int, secureOnly bool) *AuthHandler {
	return &AuthHandler{
		auth:       authService,
		cookieTTL:  cookieTTLSeconds,
		secureOnly: secureOnly,
	}
}

// registerRequest is the expected JSON body.
//
// The `binding:"required"` tag makes Gin reject the request if the field is
// missing. That only checks presence; real rules live in the service, because
// validation must not depend on which transport the request arrived through.
type registerRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest

	// ShouldBindJSON parses the body into the struct. It fails on malformed
	// JSON or a missing required field.
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest,
			"VALIDATION_FAILED", "Name, email and password are required.")
		return
	}

	// c.Request.Context() - NOT c itself.
	//
	// *gin.Context is not a context.Context. Passing c downward would compile
	// but carry no cancellation, so a query would keep running after the
	// client disconnected. This is the most common Gin mistake there is.
	user, token, err := h.auth.Register(c.Request.Context(), service.RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		respondWithDomainError(c, err)
		return
	}

	h.setSessionCookie(c, token)

	// The User model carries `json:"-"` on PasswordHash, so the hash cannot
	// appear in this response even by accident.
	c.JSON(http.StatusCreated, user)
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		// Deliberately the same message a bad password gets, so a malformed
		// request cannot be used to probe anything.
		respondError(c, http.StatusBadRequest,
			"VALIDATION_FAILED", "Email and password are required.")
		return
	}

	user, token, err := h.auth.Login(c.Request.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		respondWithDomainError(c, err)
		return
	}

	h.setSessionCookie(c, token)
	c.JSON(http.StatusOK, user)
}

// Logout handles POST /api/v1/auth/logout.
func (h *AuthHandler) Logout(c *gin.Context) {
	// Clear the cookie by setting it empty with a negative lifetime.
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(SessionCookieName, "", -1, "/", "", h.secureOnly, true)

	// HONEST LIMITATION: a JWT cannot be revoked without a server-side
	// denylist. Clearing the cookie logs this browser out, but a token copied
	// beforehand stays valid until it expires. Documented in ADR-004.
	c.Status(http.StatusNoContent)
}

// Me handles GET /api/v1/auth/me - "who am I?".
//
// The frontend calls this on load to find out whether the visitor is signed in.
func (h *AuthHandler) Me(c *gin.Context) {
	// The auth middleware already verified the token and stored the user ID.
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required.")
		return
	}

	user, err := h.auth.GetByID(c.Request.Context(), userID)
	if err != nil {
		respondWithDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// setSessionCookie stores the login token in the browser.
func (h *AuthHandler) setSessionCookie(c *gin.Context, token string) {
	// SameSite=Lax: the browser will not send this cookie on cross-site POST
	// requests, which blocks the realistic CSRF attacks.
	//
	// Note localhost:3000 and localhost:8080 count as the SAME SITE - the
	// same-site rule ignores the port - so the cookie works across our two
	// dev servers.
	c.SetSameSite(http.SameSiteLaxMode)

	c.SetCookie(
		SessionCookieName,
		token,
		h.cookieTTL,
		"/",          // path: send it to every route
		"",           // domain: default to the current host
		h.secureOnly, // Secure: HTTPS only, outside development
		true,         // HttpOnly: JAVASCRIPT CANNOT READ THIS COOKIE
	)

	// That final `true` is the important one.
	//
	// HttpOnly means document.cookie cannot see it. If an attacker manages to
	// run JavaScript on our page - through a bug, or a compromised npm package
	// - they still cannot steal the session. Storing the token in
	// localStorage, as many tutorials do, would hand it straight to them.
}
