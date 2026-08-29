package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ariyam/api-monitor/internal/models"
)

// httptest starts a REAL HTTP server on a random free port, in this process.
//
// That means these tests exercise the actual networking code - the client, the
// timeout, the body handling - without depending on any site being up. A test
// that hits a real external URL fails when your wifi drops, which trains you
// to ignore red.

// monitorFor builds a monitor pointing at a test server.
func monitorFor(url string, expected int, timeoutSeconds int) models.Monitor {
	return models.Monitor{
		Name:            "Test Monitor",
		URL:             url,
		Method:          http.MethodGet,
		ExpectedStatus:  expected,
		TimeoutSeconds:  timeoutSeconds,
		IntervalSeconds: 60,
		Enabled:         true,
	}
}

func TestCheck_HealthyEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	result := NewChecker().Check(context.Background(), monitorFor(server.URL, 200, 5))

	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.Error)
	}
	if result.StatusCode == nil || *result.StatusCode != 200 {
		t.Errorf("expected status 200, got %v", result.StatusCode)
	}
	if result.Error != "" {
		t.Errorf("expected no error message, got %q", result.Error)
	}
	if result.CheckedAt.IsZero() {
		t.Error("CheckedAt was never set")
	}
}

func TestCheck_WrongStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable) // 503
	}))
	defer server.Close()

	result := NewChecker().Check(context.Background(), monitorFor(server.URL, 200, 5))

	if result.Success {
		t.Error("a 503 was treated as success when 200 was expected")
	}
	// The status code IS recorded: the server answered, it just answered wrong.
	if result.StatusCode == nil || *result.StatusCode != 503 {
		t.Errorf("expected status 503 to be recorded, got %v", result.StatusCode)
	}
	if result.Error == "" {
		t.Error("expected an error message explaining the mismatch")
	}
}

// Expecting something other than 200 must work - a webhook endpoint might
// legitimately return 202, and a redirect checker might expect 301.
func TestCheck_NonDefaultExpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted) // 202
	}))
	defer server.Close()

	result := NewChecker().Check(context.Background(), monitorFor(server.URL, 202, 5))

	if !result.Success {
		t.Errorf("202 should be success when 202 is expected, got: %s", result.Error)
	}
}

// THE MOST IMPORTANT TEST IN THIS FILE.
//
// A timeout must produce a failure with NO status code - not 0, not 200.
func TestCheck_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the monitor's timeout allows.
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 1-second timeout against a 2-second response.
	result := NewChecker().Check(context.Background(), monitorFor(server.URL, 200, 1))

	if result.Success {
		t.Error("a timed-out check was reported as successful")
	}
	if result.StatusCode != nil {
		t.Errorf("StatusCode must be nil when nothing responded, got %d", *result.StatusCode)
	}
	if result.Error != "timed out" {
		t.Errorf("expected a readable %q message, got %q", "timed out", result.Error)
	}
	// It must give up at roughly the timeout, not wait for the full response.
	if result.ResponseTimeMs > 1800 {
		t.Errorf("took %dms; should have aborted at about 1000ms", result.ResponseTimeMs)
	}
}

func TestCheck_ConnectionRefused(t *testing.T) {
	// Start a server, then immediately close it. Its port is now dead, which
	// is a reliable way to get a refused connection.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := server.URL
	server.Close()

	result := NewChecker().Check(context.Background(), monitorFor(deadURL, 200, 5))

	if result.Success {
		t.Error("a refused connection was reported as successful")
	}
	if result.StatusCode != nil {
		t.Errorf("StatusCode must be nil when the connection failed, got %d", *result.StatusCode)
	}
	if result.Error == "" {
		t.Error("expected an error message")
	}
}

func TestCheck_InvalidURL(t *testing.T) {
	monitor := monitorFor("http://", 200, 5)
	result := NewChecker().Check(context.Background(), monitor)

	if result.Success {
		t.Error("an unusable URL was reported as successful")
	}
	if result.StatusCode != nil {
		t.Error("StatusCode must be nil when no request was made")
	}
}

func TestCheck_RespectsHTTPMethod(t *testing.T) {
	var gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	monitor := monitorFor(server.URL, 200, 5)
	monitor.Method = http.MethodHead

	NewChecker().Check(context.Background(), monitor)

	if gotMethod != http.MethodHead {
		t.Errorf("server saw method %q, want HEAD", gotMethod)
	}
}

func TestCheck_SendsUserAgent(t *testing.T) {
	var gotAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	NewChecker().Check(context.Background(), monitorFor(server.URL, 200, 5))

	if gotAgent == "" {
		t.Error("no User-Agent sent; operators cannot tell who is calling them")
	}
}

// Shutdown must abort in-flight checks rather than waiting them out.
func TestCheck_ParentContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
	}))
	defer server.Close()

	// A generous per-monitor timeout, but the PARENT context is cancelled
	// almost immediately - as happens when the process is shutting down.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	result := NewChecker().Check(ctx, monitorFor(server.URL, 200, 30))
	elapsed := time.Since(start)

	if result.Success {
		t.Error("a cancelled check was reported as successful")
	}
	// It must stop when the parent was cancelled, not sit for 30 seconds.
	if elapsed > 2*time.Second {
		t.Errorf("took %v; should have aborted on cancellation", elapsed)
	}
}

// A large response body must not be read into memory in full.
func TestCheck_LargeBodyIsBounded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// 1MB, far beyond the 64KB we are willing to drain.
		big := make([]byte, 1024*1024)
		_, _ = w.Write(big)
	}))
	defer server.Close()

	result := NewChecker().Check(context.Background(), monitorFor(server.URL, 200, 5))

	if !result.Success {
		t.Errorf("expected success, got: %s", result.Error)
	}
}
