// Package main is the entry point of the API Monitor backend.
//
// In Go, the program starts at the function called main() inside the package
// called main. That is a rule of the language, not a choice we made.
//
// This file does three things and nothing else:
//  1. load settings
//  2. connect to the database
//  3. start the web server, and stop it cleanly when asked
//
// Keeping main() to "wiring only" is deliberate. All the real logic lives in
// the internal/ packages, which makes it testable on its own.
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

	"github.com/ariyam/api-monitor/internal/config"
	"github.com/ariyam/api-monitor/internal/database"
)

func main() {
	// ---- 1. Settings ------------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		// log.Fatal prints the message and stops the program immediately.
		// If the settings are wrong there is no point continuing.
		log.Fatalf("configuration error: %v", err)
	}
	log.Printf("starting API Monitor (env=%s)", cfg.Env)

	// ---- 2. Database ------------------------------------------------------
	db, err := database.Connect(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	log.Println("connected to database")

	// Make sure the connection is closed when main() ends.
	defer func() {
		if err := database.Close(db); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	// ---- 3. Web server ----------------------------------------------------
	if !cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// GET /healthz — "are you alive, and can you reach the database?"
	//
	// It really queries the database rather than just replying "ok". A health
	// check that always says yes is worse than no health check at all: it
	// reports healthy while the app is completely broken.
	router.GET("/healthz", func(c *gin.Context) {
		// c.Request.Context() is the request's own context. If the caller
		// hangs up, it is cancelled and we stop working on a dead request.
		//
		// Careful: *gin.Context (the c here) is NOT a context.Context.
		// Mixing them up is the single most common Gin mistake.
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := database.Ping(ctx, db); err != nil {
			log.Printf("health check failed: %v", err)

			// 503 means "Service Unavailable". We report the database as down
			// but deliberately do NOT include the raw error, because those
			// messages can reveal hostnames and internal details.
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "degraded",
				"database": "unreachable",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"database": "ok",
		})
	})

	// We build the http.Server ourselves instead of calling router.Run(),
	// because we need a handle on it in order to shut it down gracefully.
	server := &http.Server{
		Addr:    announcedAddr(cfg.Port),
		Handler: router,

		// Always set timeouts. Without them a single slow or malicious client
		// can hold a connection open forever and exhaust the server.
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ---- Graceful shutdown ------------------------------------------------
	//
	// When you press Ctrl+C, or Docker stops the container, the operating
	// system sends a "signal". By default that kills the program instantly,
	// cutting off any request that is halfway through.
	//
	// Instead we catch the signal and shut down politely: stop accepting new
	// requests, let the current ones finish, then exit.
	shutdownCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,    // Ctrl+C
		syscall.SIGTERM, // what Docker sends
	)
	defer stop()

	// Start the server in the background so this function can carry on and
	// wait for the shutdown signal. "go" runs a function concurrently.
	go func() {
		log.Printf("listening on http://localhost:%s", cfg.Port)

		// ListenAndServe blocks until the server stops. When we shut down on
		// purpose it returns ErrServerClosed, which is expected, not a fault.
		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Wait here until a shutdown signal arrives.
	<-shutdownCtx.Done()
	log.Println("shutdown signal received, finishing active requests...")

	// Give in-flight requests up to 10 seconds to complete, then force the
	// issue. A fresh context is needed because the one above is already
	// cancelled — that is what woke us up.
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(timeoutCtx); err != nil {
		log.Printf("forced shutdown: %v", err)
	}

	log.Println("stopped cleanly")
}

// announcedAddr turns a port like "8080" into the ":8080" form the HTTP
// server expects. Leaving the host empty means "listen on all addresses".
func announcedAddr(port string) string {
	return ":" + port
}
