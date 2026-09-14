package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders sets a small set of response headers that cost nothing and
// close off entire classes of browser-side attacks.
//
// These are deliberately conservative. A full Content-Security-Policy is
// genuinely valuable but needs to be tuned against the real frontend's
// script/style sources or it breaks the app; getting that wrong silently is
// worse than not having one yet, so it is left as documented future work
// rather than shipped half-tuned.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Stops the browser "sniffing" a response into a different content
		// type than the one we declared. Without this, a JSON error message
		// containing something that looks like HTML/JS could, on some old
		// browsers, be executed as a script.
		c.Header("X-Content-Type-Options", "nosniff")

		// Refuses to let this API's responses be rendered inside a <frame> or
		// <iframe> on another site - the standard defence against
		// clickjacking. This is an API, not a page meant to be framed.
		c.Header("X-Frame-Options", "DENY")

		// Never send our own URLs (which can contain a slug, a monitor id, or
		// a query string) to a THIRD-PARTY site as the Referer header when a
		// link on one of our pages is followed outward.
		c.Header("Referrer-Policy", "no-referrer")

		// Disables a set of powerful browser features this API has no
		// legitimate reason to use, so an XSS elsewhere in the browser
		// cannot turn around and ask for the camera, microphone or location
		// on our origin.
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		c.Next()
	}
}
