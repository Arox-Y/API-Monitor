// Package main is the entry point of the API Monitor backend.
//
// In Go, the program starts at the function called main() inside the package
// called main. That is a rule of the language, not a choice we made.
//
// This file is WIRING ONLY:
//  1. load settings
//  2. connect to the database
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
	"github.com/ariyam/api-monitor/internal/repository"
	"github.com/ariyam/api-monitor/internal/router"
	"github.com/ariyam/api-monitor/internal/service"
	"github.com/ariyam/api-monitor/internal/worker"
)

func main() {
	// ---- 1. Settings ------------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}
	log.Printf("starting API Monitor (env=%s)", cfg.Env)

	// ---- 2. Database ------------------------------------------------------
	db, err := database.Connect(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	log.Println("connected to database")

	defer func() {
		if err := database.Close(db); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	// ---- 3. Wiring --------------------------------------------------------
	//
	// Built bottom-up, because each layer needs the one below it:
	//
	//   repository  ->  service  ->  handler
	//        (database)   (rules)     (HTTP)
	//
	// Nothing reaches upward, and nothing fetches its own dependencies.

	tokens, err := auth.NewTokenService(cfg.JWTSecret, auth.DefaultTokenTTL)
	if err != nil {
		log.Fatalf("token service error: %v", err)
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

	// The session cookie should live exactly as long as the token inside it.
	cookieTTLSeconds := int(auth.DefaultTokenTTL.Seconds())

	// Secure cookies require HTTPS, which local development does not have.
	// In production this becomes true so the cookie is never sent in clear.
	secureCookie := !cfg.IsDevelopment()

	authHandler := handler.NewAuthHandler(authService, cookieTTLSeconds, secureCookie)
	projectHandler := handler.NewProjectHandler(projectService)
	monitorHandler := handler.NewMonitorHandler(monitorService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)

	engine := router.New(router.Dependencies{
		Config:    cfg,
		DB:        db,
		Tokens:    tokens,
		Auth:      authHandler,
		Projects:  projectHandler,
		Monitors:  monitorHandler,
		Dashboard: dashboardHandler,
	})

	if !cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	// ---- 4. Server --------------------------------------------------------
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: engine,

		// Always set timeouts. Without them one slow or malicious client can
		// hold a connection open indefinitely and exhaust the server.
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ---- Graceful shutdown ------------------------------------------------
	//
	// One context governs BOTH the HTTP server and the background worker, so a
	// single Ctrl+C stops everything in an orderly way.
	shutdownCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,    // Ctrl+C
		syscall.SIGTERM, // what Docker sends
	)
	defer stop()

	// ---- Background monitoring worker -------------------------------------
	//
	// Runs in the SAME process as the API (ADR-001 and ADR-006). It is a
	// package, not a service: no message queue, no second deployment.
	//
	// ⚠️ This is correct with exactly ONE backend instance. Two would both find
	// the same due monitors and check everything twice. The fix is
	// SELECT ... FOR UPDATE SKIP LOCKED, documented in SYSTEM_DESIGN.md §8 and
	// deliberately not built, because one instance is right for this scope.
	scheduler := worker.NewScheduler(monitorRepo, checkRepo, worker.SchedulerConfig{
		Tick:        time.Duration(cfg.WorkerTickSeconds) * time.Second,
		Concurrency: cfg.WorkerConcurrency,
	})

	// workerDone lets us wait for in-flight checks during shutdown, rather
	// than killing a check halfway through writing its transaction.
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		scheduler.Run(shutdownCtx) // returns when the context is cancelled
	}()

	go func() {
		log.Printf("listening on http://localhost:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	<-shutdownCtx.Done()
	log.Println("shutdown signal received, finishing active requests...")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(timeoutCtx); err != nil {
		log.Printf("forced shutdown: %v", err)
	}

	// Wait for the worker to finish whatever it was checking. Without this the
	// process could exit mid-transaction.
	select {
	case <-workerDone:
		log.Println("worker stopped")
	case <-time.After(15 * time.Second):
		log.Println("worker did not stop in time; exiting anyway")
	}

	log.Println("stopped cleanly")
}
