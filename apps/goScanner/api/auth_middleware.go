package api

import (
	"crypto/subtle"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// ServiceTokenAuth is a Gin middleware that requires a shared secret on
// requests to privileged scanner endpoints.
//
// Background: the scanner previously accepted POST /scan from anyone on the
// Docker network (and previously from the host, via the published port). Any
// caller could spoof tenant_id in the JSON body and trigger scans against
// another tenant's sources. This middleware closes that gap.
//
// Configuration:
//   - SCANNER_SERVICE_TOKEN must be set in the scanner and backend envs and
//     must match (both containers read it from env).
//   - SCANNER_AUTH_REQUIRED=false disables enforcement (dev only). In any
//     other case, missing or empty SCANNER_SERVICE_TOKEN causes the scanner
//     to reject all /scan calls.
//
// Callers send the token as `X-Scanner-Token: <token>`. Constant-time compare
// prevents timing leaks.
func ServiceTokenAuth() gin.HandlerFunc {
	expected := os.Getenv("SCANNER_SERVICE_TOKEN")
	authRequired := os.Getenv("SCANNER_AUTH_REQUIRED") != "false"

	return func(c *gin.Context) {
		if !authRequired {
			// Dev mode: allow anonymous. Log once per startup elsewhere.
			c.Next()
			return
		}
		if expected == "" {
			log.Printf("ERROR: SCANNER_SERVICE_TOKEN not set but SCANNER_AUTH_REQUIRED=true; rejecting")
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "service_misconfigured",
				"message": "Scanner auth is required but no service token is configured",
			})
			c.Abort()
			return
		}
		presented := c.GetHeader("X-Scanner-Token")
		if presented == "" || subtle.ConstantTimeCompare([]byte(presented), []byte(expected)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid or missing X-Scanner-Token",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
