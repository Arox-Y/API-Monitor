package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// newTestContext builds a *gin.Context whose ClientIP() reports remoteAddr,
// without starting a real server.
func newTestContext(remoteAddr string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = remoteAddr
	return c, w
}

func TestRateLimiter_AllowsUpToBurstThenBlocks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 60/minute = 1/second, burst 3: the first 3 requests are free (the
	// burst), the 4th must be refused because no time has passed to refill.
	rl := NewRateLimiter(60, 3)
	middleware := rl.Middleware()

	for i := 1; i <= 3; i++ {
		c, w := newTestContext("203.0.113.5:1234")
		middleware(c)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: got status %d, want 200 (default gin status before a handler writes one is 200)", i, w.Code)
		}
	}

	c, w := newTestContext("203.0.113.5:1234")
	middleware(c)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("4th request in the same instant: got %d, want 429", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("a 429 without Retry-After leaves the client guessing when to try again")
	}
}

func TestRateLimiter_DifferentIPsAreIndependent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := NewRateLimiter(60, 1) // burst of exactly 1
	middleware := rl.Middleware()

	// Exhaust client A's single token.
	cA1, wA1 := newTestContext("203.0.113.10:1")
	middleware(cA1)
	if wA1.Code != http.StatusOK {
		t.Fatalf("client A's first request should succeed, got %d", wA1.Code)
	}

	cA2, wA2 := newTestContext("203.0.113.10:1")
	middleware(cA2)
	if wA2.Code != http.StatusTooManyRequests {
		t.Fatalf("client A's second request should be limited, got %d", wA2.Code)
	}

	// Client B has never been seen, and must NOT inherit A's exhausted bucket
	// - that would mean one noisy client locks out everyone else.
	cB, wB := newTestContext("203.0.113.20:1")
	middleware(cB)
	if wB.Code != http.StatusOK {
		t.Errorf("SECURITY/CORRECTNESS FAILURE: client B was rate-limited by client A's traffic, got %d", wB.Code)
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter(60, 5)

	// Touch three visitors so they exist in the map.
	rl.get("1.1.1.1")
	rl.get("2.2.2.2")
	rl.get("3.3.3.3")

	rl.mu.Lock()
	before := len(rl.visitors)
	rl.mu.Unlock()
	if before != 3 {
		t.Fatalf("expected 3 visitors before cleanup, got %d", before)
	}

	// A maxAge of 0 means "anything not touched in the last instant is
	// stale" - which, by the time cleanup runs, is all of them.
	time.Sleep(time.Millisecond)
	rl.cleanup(0)

	rl.mu.Lock()
	after := len(rl.visitors)
	rl.mu.Unlock()
	if after != 0 {
		t.Errorf("expected cleanup to remove all stale visitors, %d remain", after)
	}
}

func TestRateLimiter_StartCleanup_StopsOnContextCancel(t *testing.T) {
	rl := NewRateLimiter(60, 5)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		rl.StartCleanup(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// StartCleanup returned, as it should once its context is cancelled.
	case <-time.After(2 * time.Second):
		t.Fatal("StartCleanup did not stop within 2s of context cancellation")
	}
}
