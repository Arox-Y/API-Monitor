package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS allows the frontend to call this API from a different origin.
//
// Browsers block a page on one origin from reading responses from another,
// unless the server explicitly permits it. Our frontend runs on
// localhost:3000 and the API on localhost:8080 - different ports means
// different origins, so without this every request from the browser fails.
//
// Note this is a BROWSER rule. It stops a malicious page from reading our
// responses using a visitor's cookies; it is not a firewall, and it does not
// stop anyone calling the API with curl.
func CORS(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Only echo back the origin if it is the one we configured.
		//
		// THE WILDCARD TRAP: `Access-Control-Allow-Origin: *` cannot be
		// combined with credentials. Browsers reject that combination
		// outright, and since our auth is a cookie, every request is
		// credentialed. The allowed origin must be an exact value.
		if origin != "" && origin == allowedOrigin {
			c.Header("Access-Control-Allow-Origin", origin)

			// Without this the browser refuses to SEND our session cookie,
			// so every request would look logged-out.
			c.Header("Access-Control-Allow-Credentials", "true")

			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type")

			// Tell caches that the response depends on the Origin header, so
			// a response for one origin is never reused for another.
			c.Header("Vary", "Origin")
		}

		// Before certain requests the browser sends an OPTIONS "preflight"
		// asking whether the real request is allowed. It carries no body and
		// must not reach our handlers - answer it here and stop.
		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Max-Age", "600") // cache the answer 10 min
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
