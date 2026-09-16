package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ariyam/api-monitor/internal/models"
)

// MonitorFinder supplies the monitors that are due.
//
// Declared here, in the package that USES it, rather than in the repository
// package that implements it. That is the Go convention, and it means this
// package depends on a two-line description of what it needs instead of on
// GORM and PostgreSQL.
type MonitorFinder interface {
	FindDue(ctx context.Context, limit int) ([]models.Monitor, error)
}

// ResultStore saves what a check produced.
type ResultStore interface {
	RecordResult(ctx context.Context, monitorID uuid.UUID, result CheckResult, transition Transition) error
}

// Scheduler repeatedly finds due monitors and checks them.
type Scheduler struct {
	monitors    MonitorFinder
	results     ResultStore
	checker     *Checker
	logger      *slog.Logger
	tick        time.Duration
	concurrency int
	batchSize   int
}

// SchedulerConfig configures a Scheduler.
type SchedulerConfig struct {
	// Tick is how often we LOOK for work. It is the scheduler's resolution,
	// not any monitor's interval. A 10s tick with 60s monitors means each
	// monitor is checked within 10 seconds of becoming due, which is fine.
	Tick time.Duration

	// Concurrency caps how many checks run at the same time.
	Concurrency int

	// BatchSize caps how many monitors one tick will take on.
	BatchSize int

	// Checker performs the actual HTTP checks. Optional: nil defaults to
	// NewChecker() with SSRF protection enabled - the safe default for
	// anything that forgets to set this field.
	//
	// main.go sets it explicitly, turning protection off outside production
	// (see the comment there for why). Scheduler tests set it too, the only
	// way they can reach the httptest servers they run on 127.0.0.1. See
	// scheduler_test.go.
	Checker *Checker

	// Logger receives structured log entries. Optional: nil defaults to
	// slog.Default(), which main.go configures once for the whole program.
	Logger *slog.Logger
}

// NewScheduler builds a Scheduler, applying sane defaults.
func NewScheduler(monitors MonitorFinder, results ResultStore, cfg SchedulerConfig) *Scheduler {
	if cfg.Tick <= 0 {
		cfg.Tick = 10 * time.Second
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 20
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}

	checker := cfg.Checker
	if checker == nil {
		checker = NewChecker() // SSRF protection enabled - the production default
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Scheduler{
		monitors:    monitors,
		results:     results,
		checker:     checker,
		logger:      logger,
		tick:        cfg.Tick,
		concurrency: cfg.Concurrency,
		batchSize:   cfg.BatchSize,
	}
}

// Run drives the scheduler until ctx is cancelled.
//
// It BLOCKS, so callers run it in its own goroutine. It returns only after
// every in-flight check has finished, which is what makes shutdown clean.
func (s *Scheduler) Run(ctx context.Context) {
	s.logger.Info("worker starting",
		"tick", s.tick.String(),
		"concurrency", s.concurrency,
		"batch_size", s.batchSize,
	)

	// A Ticker sends the current time down a channel every interval.
	// Stopping it releases the underlying timer.
	ticker := time.NewTicker(s.tick)
	defer ticker.Stop()

	// Check immediately on startup rather than waiting a full tick. Otherwise
	// restarting the process means up to 10 seconds of nothing happening.
	s.runOnce(ctx)

	for {
		// select waits on several channels at once and takes whichever is
		// ready first. Here that is either "time to work" or "time to stop",
		// which is the idiomatic Go shape for a background loop.
		select {
		case <-ctx.Done():
			s.logger.Info("worker stopping: shutdown requested")
			return

		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

// runOnce performs one full pass: find what is due, check it all, wait.
func (s *Scheduler) runOnce(ctx context.Context) {
	monitors, err := s.monitors.FindDue(ctx, s.batchSize)
	if err != nil {
		// A database hiccup must NOT kill the worker. Log it and try again on
		// the next tick - a monitoring system that dies when its database
		// blips is worse than useless.
		s.logger.Error("worker could not load due monitors", "error", err)
		return
	}

	if len(monitors) == 0 {
		return
	}

	s.logger.Info("worker checking monitors", "count", len(monitors))

	// A buffered channel used as a SEMAPHORE.
	//
	// A buffered channel of size N accepts N sends before it blocks. So
	// sending an empty struct is "take a slot" and receiving is "give it
	// back". When all 20 slots are taken, the next send BLOCKS until one
	// frees up - and that blocking is the entire mechanism.
	//
	// struct{}{} is used because it occupies zero bytes: we care about the
	// act of sending, never the value.
	//
	// Without this, 500 due monitors would launch 500 goroutines opening 500
	// simultaneous connections, and exhaust file descriptors.
	semaphore := make(chan struct{}, s.concurrency)

	// A WaitGroup counts outstanding work. Add before starting, Done when
	// finished, Wait blocks until the count reaches zero.
	var wg sync.WaitGroup

	for _, monitor := range monitors {
		// Stop dispatching immediately if we are shutting down.
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		default:
		}

		semaphore <- struct{}{} // acquire a slot (blocks when full)
		wg.Add(1)

		// Passing the monitor as an ARGUMENT, not capturing the loop variable.
		//
		// Go 1.22 made loop variables per-iteration, so capturing is safe now,
		// but passing explicitly makes the goroutine's input obvious and works
		// on every version. This used to be Go's most notorious bug.
		go func(m models.Monitor) {
			defer wg.Done()
			defer func() { <-semaphore }() // release the slot, always

			// PANIC RECOVERY, PER GOROUTINE.
			//
			// This is not optional. An unrecovered panic inside ANY goroutine
			// crashes the ENTIRE PROGRAM - it is not confined to that
			// goroutine, and the HTTP server's recovery middleware cannot see
			// it. One malformed response could take the whole service down.
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("worker panic while checking monitor",
						"monitor_id", m.ID, "monitor_name", m.Name, "panic", r)
				}
			}()

			s.checkOne(ctx, m)
		}(monitor)
	}

	// Wait for this batch before returning, so ticks cannot overlap and pile
	// work on top of work.
	wg.Wait()
}

// checkOne performs a single check and records the outcome.
func (s *Scheduler) checkOne(ctx context.Context, monitor models.Monitor) {
	result := s.checker.Check(ctx, monitor)

	// Decide what this means - a pure function, fully unit tested.
	transition := DecideTransition(monitor.Status, result.Success)

	// Persisting uses a SEPARATE context that is not cancelled by shutdown.
	//
	// If the parent context has just been cancelled, we have already done the
	// work of making the HTTP request. Throwing the result away because we are
	// shutting down would lose a real measurement, and worse, could leave an
	// outage unrecorded. Five seconds is plenty to write three rows.
	storeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	if err := s.results.RecordResult(storeCtx, monitor.ID, result, transition); err != nil {
		s.logger.Error("worker failed to record check",
			"monitor_id", monitor.ID, "monitor_name", monitor.Name, "error", err)
		return
	}

	// Deliberately NOT logging monitor.URL below.
	//
	// A monitor's URL is user-supplied and can legitimately contain a secret
	// in its query string (an API key used as a health-check credential, for
	// instance). Logging the full URL would put that secret into server logs
	// forever, on every incident. The monitor's own NAME is a label the user
	// chose specifically to be non-sensitive, so that is what identifies it
	// here instead.
	if transition.OpenIncident {
		s.logger.Warn("incident opened",
			"monitor_id", monitor.ID, "monitor_name", monitor.Name, "cause", result.Error)
	}
	if transition.CloseIncident {
		s.logger.Info("incident resolved",
			"monitor_id", monitor.ID, "monitor_name", monitor.Name)
	}
}
