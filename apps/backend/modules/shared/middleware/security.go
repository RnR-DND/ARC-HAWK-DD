package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders adds critical security headers to every response
// Hardens the application against XSS, Clickjacking, and MIME-sniffing
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// HSTS - Force HTTPS for 1 year (31536000 seconds)
		// Protects against SSL stripping
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Prevent MIME-sniffing attacks
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent Clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Prevent XSS attacks (Legacy browsers)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Content Security Policy - Restrict resources to same origin by default
		// This is a strict default; might need adjustment if using CDNs or external fonts/scripts
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self' ws: wss:;")

		// Referrer Policy - Don't leak full URLs to external sites
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}
