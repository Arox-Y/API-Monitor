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
	engine.Use(middleware.CORS(deps.Config.CORSOrigin))

	// ---- Public: infrastructure -------------------------------------------

	// Deliberately outside /api/v1. Health checks are not part of the product
	// API and should not be versioned alongside it.
	engine.GET("/healthz", healthzHandler(deps.DB))

	// ---- API v1 -----------------------------------------------------------

	v1 := engine.Group("/api/v1")

	// Public auth routes: you cannot be signed in to sign in.
	v1.POST("/auth/register", deps.Auth.Register)
	v1.POST("/auth/login", deps.Auth.Login)

	// The public status page. The ONLY data endpoint reachable without a
	// session, which is why it is listed here on its own rather than buried
	// among the protected routes.
	v1.GET("/public/status/:slug", deps.Status.Get)

	// Everything below requires a valid session cookie.
	//
	// Applying RequireAuth to a GROUP rather than to each route is the point:
	// a new endpoint added inside this group is protected automatically.
	// Listing routes individually eventually means forgetting one, and a
	// forgotten auth check is a data breach.
	authed := v1.Group("")
	authed.Use(middleware.RequireAuth(deps.Tokens, handler.SessionCookieName))
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
