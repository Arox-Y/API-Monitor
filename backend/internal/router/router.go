// Package router defines every URL this API answers.
//
// Keeping the whole route table in one file means you can see the entire API
// surface at a glance - including, importantly, which routes are public.
package router

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ariyam/api-monitor/internal/auth"
	"github.com/ariyam/api-monitor/internal/config"
	"github.com/ariyam/api-monitor/internal/database"
	"github.com/ariyam/api-monitor/internal/handler"
	"github.com/ariyam/api-monitor/internal/middleware"
)

// Dependencies is everything the router needs.
//
// Grouping them in a struct beats a long parameter list: adding one later does
// not force every call site to change.
type Dependencies struct {
	Config    config.Config
	DB        *gorm.DB
	Tokens    *auth.TokenService
	Auth      *handler.AuthHandler
	Projects  *handler.ProjectHandler
	Monitors  *handler.MonitorHandler
	Dashboard *handler.DashboardHandler
	Status    *handler.StatusHandler

	// Rate limiters, one per traffic class - the numbers match
	// API_SPEC.md section 4 exactly. Built once in main.go, so their
	// background cleanup goroutines share the app's single shutdown
	// context, and handed in here rather than constructed per request.
	AuthRateLimit    *middleware.RateLimiter // 5/min  - register & login
	DefaultRateLimit *middleware.RateLimiter // 100/min - everything else authenticated
	PublicRateLimit  *middleware.RateLimiter // 60/min  - the public status page
}

// New builds the Gin engine with all routes and middleware attached.
func New(deps Dependencies) *gin.Engine {
	engine := gin.New()

	// Middleware order matters, and these run top to bottom.
	//
	// Recovery goes FIRST so it can catch a panic from anything after it. A
	// panic in a handler would otherwise crash the entire server, taking down
	// every other user's requests along with it.
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())
	engine.Use(middleware.SecurityHeaders())
	engine.Use(middleware.CORS(deps.Config.CORSOrigin))

	// ---- Public: infrastructure -------------------------------------------

	// Deliberately outside /api/v1. Health checks are not part of the product
	// API and should not be versioned alongside it, and are not rate limited -
	// Docker's own healthcheck polls this constantly by design.
	engine.GET("/healthz", healthzHandler(deps.DB))

	// ---- API v1 -----------------------------------------------------------

	v1 := engine.Group("/api/v1")

	// Public auth routes: you cannot be signed in to sign in.
	//
	// The strictest limit in the app (5/min) lives here specifically because
	// these are the endpoints a credential-stuffing attack would hammer -
	// trying many passwords against one email, or many emails at once.
	auth1 := v1.Group("/auth")
	auth1.Use(deps.AuthRateLimit.Middleware())
	{
		auth1.POST("/register", deps.Auth.Register)
		auth1.POST("/login", deps.Auth.Login)
	}

	// The public status page. The ONLY data endpoint reachable without a
	// session, which is why it is listed here on its own rather than buried
	// among the protected routes. Its own, more generous limit (60/min)
	// keeps a status page that gets linked somewhere busy from being taken
	// down by its own popularity.
	v1.GET("/public/status/:slug", deps.PublicRateLimit.Middleware(), deps.Status.Get)

	// Everything below requires a valid session cookie.
	//
	// Applying RequireAuth to a GROUP rather than to each route is the point:
	// a new endpoint added inside this group is protected automatically.
	// Listing routes individually eventually means forgetting one, and a
	// forgotten auth check is a data breach. The default rate limit (100/min)
	// is applied the same way, for the same reason.
	authed := v1.Group("")
	authed.Use(middleware.RequireAuth(deps.Tokens, handler.SessionCookieName))
	authed.Use(deps.DefaultRateLimit.Middleware())
	{
		authed.POST("/auth/logout", deps.Auth.Logout)
		authed.GET("/auth/me", deps.Auth.Me)

		// Projects
		authed.GET("/projects", deps.Projects.List)
		authed.POST("/projects", deps.Projects.Create)
		authed.GET("/projects/:id", deps.Projects.Get)
		authed.PATCH("/projects/:id", deps.Projects.Update)
		authed.DELETE("/projects/:id", deps.Projects.Delete)

		// Monitors nested under a project: creation must name its parent,
		// because a monitor cannot exist without one.
		authed.GET("/projects/:id/monitors", deps.Monitors.ListByProject)
		authed.POST("/projects/:id/monitors", deps.Monitors.Create)

		// Monitors addressed directly. A monitor ID is globally unique, so
		// nesting every route would only make the URLs longer and brittle.
		authed.GET("/monitors", deps.Monitors.List)
		authed.GET("/monitors/:id", deps.Monitors.Get)
		authed.PATCH("/monitors/:id", deps.Monitors.Update)
		authed.DELETE("/monitors/:id", deps.Monitors.Delete)

		// Monitor history: the monitor plus its checks, incidents and uptime.
		authed.GET("/monitors/:id/checks", deps.Dashboard.MonitorDetail)

		// Incident history across every monitor the user owns.
		authed.GET("/incidents", deps.Dashboard.Incidents)

		// One aggregate endpoint for the whole dashboard screen (ADR-005).
		authed.GET("/dashboard/stats", deps.Dashboard.Stats)
	}

	return engine
}

// healthzHandler reports whether the service and its database are alive.
func healthzHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := database.Ping(ctx, db); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "degraded",
				"database": "unreachable",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok", "database": "ok"})
	}
}
