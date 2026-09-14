package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter is a per-client-IP token bucket, used to blunt credential
// stuffing on the auth endpoints and to keep any single client from
// hammering the API.
//
// HONEST LIMITATION: state lives in this one process's memory. A restart
// resets every client back to a fresh bucket, and the limit is NOT shared if
// the app ever ran as more than one instance. The standard fix for that is a
// shared store like Redis - explicitly out of scope for the MVP (CLAUDE.md
// section 6) and unnecessary here anyway, since ADR-006 already requires
// exactly one backend instance for the scheduler to be correct.
//
// A visitor's bucket is created on first request and refills continuously -
// this is the "token bucket" algorithm golang.org/x/time/rate implements:
// think of a bucket that gains one token every (60/requestsPerMinute)
// seconds, up to `burst` tokens. Every request spends one token; an empty
// bucket means "wait".
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    rate.Limit
	burst    int
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter builds a limiter allowing requestsPerMinute sustained
// requests per client IP, with room for a short burst above that.
func NewRateLimiter(requestsPerMinute, burst int) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*visitor),
		// rate.Limit is "events per second". Converting from per-minute here
		// keeps every call site talking in the unit the product spec
		// (API_SPEC.md section 4) actually uses.
		limit: rate.Limit(float64(requestsPerMinute) / 60.0),
		burst: burst,
	}
}

// get returns the bucket for one client key, creating it on first sight.
func (rl *RateLimiter) get(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[key]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(rl.limit, rl.burst)}
		rl.visitors[key] = v
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// cleanup drops any client not seen recently, so the map does not grow
// forever as new IPs make one request each and are never heard from again.
func (rl *RateLimiter) cleanup(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for key, v := range rl.visitors {
		if time.Since(v.lastSeen) > maxAge {
			delete(rl.visitors, key)
		}
	}
}

// StartCleanup runs cleanup on a ticker until ctx is cancelled.
//
// Tied to the same shutdown context as the rest of the app (see main.go), so
// this goroutine stops cleanly instead of leaking past process shutdown.
func (rl *RateLimiter) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.cleanup(10 * time.Minute)
		}
	}
}

// Middleware enforces the limit, keyed by the caller's IP address.
//
// c.ClientIP() trusts standard proxy headers only when Gin is configured
// with trusted proxies; with none configured (our case) it falls back to the
// raw connection address, which is what we want for a service with no
// reverse proxy in front of it yet. If one is added later, Gin's trusted
// proxy list must be configured too, or every client would appear to share
// the proxy's IP and rate-limit each other.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		limiter := rl.get(c.ClientIP())

		if !limiter.Allow() {
			// Matches the standard error envelope every other handler uses
			// (see handler/response.go) so the frontend needs only one
			// error-handling code path.
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "Too many requests. Please slow down.",
				},
			})
			return
		}

		c.Next()
	}
}
