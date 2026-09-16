// Package main is the entry point of the API Monitor backend.
//
// In Go, the program starts at the function called main() inside the package
// called main. That is a rule of the language, not a choice we made.
//
// This file is WIRING ONLY:
//  1. load settings
//  2. connect to the database and apply pending migrations
//  3. build each layer and hand it its dependencies
//  4. start the server, and stop it cleanly when asked
//
// All the real logic lives in internal/, which is what makes it testable on
// its own. Every dependency is created here and passed down, so there is one
// place to look to see how the whole program fits together - and no hidden
// global state.
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ariyam/api-monitor/internal/auth"
	"github.com/ariyam/api-monitor/internal/config"
	"github.com/ariyam/api-monitor/internal/database"
	"github.com/ariyam/api-monitor/internal/handler"
	"github.com/ariyam/api-monitor/internal/middleware"
	"github.com/ariyam/api-monitor/internal/repository"
	"github.com/ariyam/api-monitor/internal/router"
	"github.com/ariyam/api-monitor/internal/service"
	"github.com/ariyam/api-monitor/internal/worker"
)

func main() {
	// ---- 1. Settings ------------------------------------------------------
	//
	// A plain stdlib log.Fatalf here, before slog exists, is the one place
	// in this file that is not structured logging - there is nothing to
	// configure a logger's format FROM yet, since that decision itself lives
	// in cfg.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	// ---- Structured logging -----------------------------------------------
	//
	// JSON in production (machine-parseable, the format a real log
	// aggregator expects), plain text in development (a human reading a
	// terminal). SYSTEM_DESIGN.md section 7 documents this choice; slog is
	// the standard library's structured logger, so it needed no dependency.
	//
	// slog.SetDefault means any package that calls the package-level
	// slog.Info/Error functions - rather than carrying its own *slog.Logger -
	// picks up this same configuration automatically (see handler/response.go).
	var logHandler slog.Handler
	if cfg.IsDevelopment() {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	logger.Info("starting API Monitor", "env", cfg.Env)

	// ---- 2. Database --------------------------------------------------------
	db, err := database.Connect(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to database")

	defer func() {
		if err := database.Close(db); err != nil {
			logger.Error("error closing database", "error", err)
		}
	}()

	// ---- 2b. Migrations ----------------------------------------------------
	//
	// Applied here, automatically, every time the process starts - not by a
	// developer remembering to run psql by hand. See internal/database/
	// migrate.go for why this is still "explicit" in ADR-003's sense: the
	// SQL files are the single source of truth, this just runs them.
	//
	// This runs before anything else touches the database, so a fresh
	// container - or a fresh contributor's first `docker compose up` - ends
	// up with a real, versioned schema with no manual step at all.
	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		logger.Error("running migrations failed", "error", err)
		os.Exit(1)
	}
	logger.Info("migrations applied")

	// ---- Graceful shutdown --------------------------------------------------
	//
	// Created early, and shared by EVERYTHING that needs to stop cleanly when
	// asked: the HTTP server, the background worker, and the rate limiters'
	// cleanup goroutines below. One Ctrl+C stops all of it in the same way.
	shutdownCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,    // Ctrl+C
		syscall.SIGTERM, // what Docker sends
	)
	defer stop()

	// ---- 3. Wiring ----------------------------------------------------------
	//
	// Built bottom-up, because each layer needs the one below it:
	//
	//   repository  ->  service  ->  handler
	//        (database)   (rules)     (HTTP)
	//
	// Nothing reaches upward, and nothing fetches its own dependencies.

	tokens, err := auth.NewTokenService(cfg.JWTSecret, auth.DefaultTokenTTL)
	if err != nil {
		logger.Error("token service error", "error", err)
		os.Exit(1)
	}

	userRepo := repository.NewUserRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	monitorRepo := repository.NewMonitorRepository(db)
	checkRepo := repository.NewCheckRepository(db)
	incidentRepo := repository.NewIncidentRepository(db)
	statsRepo := repository.NewStatsRepository(db)

	authService := service.NewAuthService(userRepo, tokens)
	projectService := service.NewProjectService(projectRepo)
	// MonitorService needs the project repository too, so it can confirm a
	// user owns the project before letting them add a monitor to it.
	monitorService := service.NewMonitorService(monitorRepo, projectRepo)
	dashboardService := service.NewDashboardService(statsRepo, incidentRepo, checkRepo, monitorRepo)
	statusService := service.NewStatusService(projectRepo, monitorRepo, incidentRepo, statsRepo)

	// The session cookie should live exactly as long as the token inside it.
	cookieTTLSeconds := int(auth.DefaultTokenTTL.Seconds())

	// Secure cookies require HTTPS, which local development does not have.
	// In production this becomes true so the cookie is never sent in clear.
	secureCookie := !cfg.IsDevelopment()

	authHandler := handler.NewAuthHandler(authService, cookieTTLSeconds, secureCookie)
	projectHandler := handler.NewProjectHandler(projectService)
	monitorHandler := handler.NewMonitorHandler(monitorService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	statusHandler := handler.NewStatusHandler(statusService)

	// ---- Rate limiters --------------------------------------------------
	//
	// Numbers match API_SPEC.md section 4 exactly:
	//   auth endpoints        5 requests / minute / IP  - the strictest,
	//                         because these are what a credential-stuffing
	//                         attack would hammer
	//   authenticated routes  100 requests / minute / IP
	//   public status page    60 requests / minute / IP
	//
	// Each gets a small burst allowance above its sustained rate, so one
	// legitimate user opening a few tabs at once is not immediately punished.
	authRateLimit := middleware.NewRateLimiter(5, 5)
	defaultRateLimit := middleware.NewRateLimiter(100, 20)
	publicRateLimit := middleware.NewRateLimiter(60, 10)

	// Each limiter's cleanup goroutine shares the app's one shutdown
	// context, so none of them outlive the process they belong to.
	go authRateLimit.StartCleanup(shutdownCtx)
	go defaultRateLimit.StartCleanup(shutdownCtx)
	go publicRateLimit.StartCleanup(shutdownCtx)

	engine := router.New(router.Dependencies{
		Config:           cfg,
		DB:               db,
		Tokens:           tokens,
		Auth:             authHandler,
		Projects:         projectHandler,
		Monitors:         monitorHandler,
		Dashboard:        dashboardHandler,
		Status:           statusHandler,
		AuthRateLimit:    authRateLimit,
		DefaultRateLimit: defaultRateLimit,
		PublicRateLimit:  publicRateLimit,
	})

	if !cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	// ---- 4. Server ----------------------------------------------------------
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: engine,

		// Always set timeouts. Without them one slow or malicious client can
		// hold a connection open indefinitely and exhaust the server.
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ---- Background monitoring worker -------------------------------------
	//
	// Runs in the SAME process as the API (ADR-001 and ADR-006). It is a
	// package, not a service: no message queue, no second deployment.
	//
	// ⚠️ This is correct with exactly ONE backend instance. Two would both find
	// the same due monitors and check everything twice. The fix is
	// SELECT ... FOR UPDATE SKIP LOCKED, documented in SYSTEM_DESIGN.md §8 and
	// deliberately not built, because one instance is right for this scope.
	// SSRF protection (ADR-012) is a production concern: it exists to stop a
	// PUBLIC deployment from being tricked into probing its own internal
	// network via a user-supplied URL. On a developer's own machine there is
	// no untrusted user and no internal network to protect - and disabling it
	// is what lets a monitor legitimately target a service on localhost, the
	// same way ENV already relaxes the login cookie's Secure flag below.
	scheduler := worker.NewScheduler(monitorRepo, checkRepo, worker.SchedulerConfig{
		Tick:        time.Duration(cfg.WorkerTickSeconds) * time.Second,
		Concurrency: cfg.WorkerConcurrency,
		Checker:     worker.NewChecker(worker.WithSSRFProtection(!cfg.IsDevelopment())),
		Logger:      logger,
	})

	// workerDone lets us wait for in-flight checks during shutdown, rather
	// than killing a check halfway through writing its transaction.
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		scheduler.Run(shutdownCtx) // returns when the context is cancelled
	}()

	go func() {
		logger.Info("listening", "addr", "http://localhost:"+cfg.Port)
		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownCtx.Done()
	logger.Info("shutdown signal received, finishing active requests")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(timeoutCtx); err != nil {
		logger.Error("forced shutdown", "error", err)
	}

	// Wait for the worker to finish whatever it was checking. Without this the
	// process could exit mid-transaction.
	select {
	case <-workerDone:
		logger.Info("worker stopped")
	case <-time.After(15 * time.Second):
		logger.Warn("worker did not stop in time, exiting anyway")
	}

	logger.Info("stopped cleanly")
}
