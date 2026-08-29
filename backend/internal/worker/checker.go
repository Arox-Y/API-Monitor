// Package worker contains the background monitoring engine: the scheduler
// that decides what to check, and the checker that does the checking.
package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ariyam/api-monitor/internal/models"
)

// CheckResult is what one visit to a URL produced.
//
// It is a plain value with no database or HTTP types in it, which is what
// lets the checker be tested without either.
type CheckResult struct {
	CheckedAt      time.Time
	Success        bool
	StatusCode     *int // nil when no HTTP response arrived at all
	ResponseTimeMs int
	Error          string
}

// Checker performs a single HTTP check against a monitor.
type Checker struct {
	client *http.Client
}

// NewChecker builds a Checker with a shared HTTP client.
//
// ONE client, created once, reused for every check. This matters more than it
// looks: an http.Client holds a pool of open TCP connections and reuses them.
// Creating a client per request throws that pool away each time, so every
// check pays for a fresh TCP handshake (and a TLS handshake on https).
//
// Note there is no Timeout set on the client itself. That would be a single
// value for every monitor, and each of ours has its own timeout setting. We
// use a per-request context instead - see Check below.
func NewChecker() *Checker {
	return &Checker{
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},

			// Follow redirects, but not forever. A misconfigured site can
			// redirect in a loop, and without a limit we would follow it until
			// something gave out.
			//
			// ⚠️ SSRF gap (ADR-012 level 3): a public URL is allowed to
			// redirect to a private address like 127.0.0.1. Blocking that
			// means re-validating the destination at every hop. Documented as
			// a known gap, scheduled for Day 5.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("stopped after 5 redirects")
				}
				return nil
			},
		},
	}
}

// Check visits a monitor's URL once and reports what happened.
//
// It returns a CheckResult and no error: a failed check is a normal, expected
// outcome that we RECORD, not an error the caller must handle. The monitored
// site being down is the whole point of the product, not a fault in our code.
func (c *Checker) Check(ctx context.Context, monitor models.Monitor) CheckResult {
	// A deadline for THIS monitor, using THIS monitor's own timeout setting.
	//
	// context.WithTimeout is the correct way to bound work in Go. It cancels
	// automatically when the time runs out, and that cancellation travels down
	// into the HTTP request, aborting the connection.
	timeout := time.Duration(monitor.TimeoutSeconds) * time.Second
	reqCtx, cancel := context.WithTimeout(ctx, timeout)

	// ALWAYS defer cancel(). Even when the request finishes early, the timer
	// behind the context lives until cancel() is called. Forgetting this leaks
	// a timer per check - one of the most common bugs in Go, and one that only
	// shows up as slowly growing memory in production.
	defer cancel()

	request, err := http.NewRequestWithContext(reqCtx, monitor.Method, monitor.URL, nil)
	if err != nil {
		// A malformed URL. Validation should have caught it, but a monitor
		// created before a rule changed could still reach here.
		return CheckResult{
			CheckedAt:      time.Now().UTC(),
			Success:        false,
			ResponseTimeMs: 0,
			Error:          fmt.Sprintf("invalid request: %v", err),
		}
	}

	// Identify ourselves. Some sites block unknown clients, and an operator
	// looking at their logs deserves to know who is calling them every minute.
	request.Header.Set("User-Agent", "APIMonitor/1.0 (+https://github.com/Arox-Y/API-Monitor)")

	// Start the clock as late as possible so we measure the request, not our
	// own setup work.
	start := time.Now()
	response, err := c.client.Do(request)
	elapsed := time.Since(start)

	checkedAt := time.Now().UTC()
	responseMs := int(elapsed.Milliseconds())

	if err != nil {
		// No HTTP response at all: a timeout, DNS failure, refused connection,
		// or TLS error. There is NO status code to record, so it stays nil.
		return CheckResult{
			CheckedAt:      checkedAt,
			Success:        false,
			StatusCode:     nil,
			ResponseTimeMs: responseMs,
			Error:          describeTransportError(err, reqCtx),
		}
	}

	// Two things must happen to every response body, in this order.
	//
	// 1. Drain it. The connection cannot be returned to the pool until the
	//    body is fully read, so skipping this quietly disables connection
	//    reuse - the pool fills with unusable connections.
	// 2. Close it. Otherwise the connection leaks entirely.
	//
	// We do not care about the CONTENT, only that it arrived, so it is copied
	// to io.Discard. LimitReader caps how much we will read, so a monitor
	// pointed at a huge file cannot exhaust our memory.
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64*1024))

	statusCode := response.StatusCode
	success := statusCode == monitor.ExpectedStatus

	result := CheckResult{
		CheckedAt:      checkedAt,
		Success:        success,
		StatusCode:     &statusCode,
		ResponseTimeMs: responseMs,
	}

	if !success {
		result.Error = fmt.Sprintf("expected status %d, got %d",
			monitor.ExpectedStatus, statusCode)
	}

	return result
}

// describeTransportError turns a Go error into something a user can read.
//
// "context deadline exceeded" is accurate and means nothing to somebody
// looking at their dashboard at 3am.
func describeTransportError(err error, ctx context.Context) string {
	// Distinguish OUR timeout from other failures. ctx.Err() reports why the
	// context ended, which is more reliable than string-matching the error.
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "timed out"
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return "check cancelled during shutdown"
	}
	return err.Error()
}
