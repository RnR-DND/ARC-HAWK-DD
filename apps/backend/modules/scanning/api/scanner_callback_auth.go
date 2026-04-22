package api

import (
	"crypto/subtle"
	"net/http"
	"os"

	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ScannerCallbackAuth is a middleware that allows two authentication paths
// on scanner-callback endpoints (ingest-verified, /:id/complete, /:id/progress-event):
//
//  1. A valid user JWT / API key (set by the global auth middleware earlier)
//  2. A valid X-Scanner-Token header matching SCANNER_SERVICE_TOKEN
//
// The scanner itself doesn't have a user JWT but does hold the shared service
// token, so path (2) is how it authenticates its callbacks.
//
// Without this middleware, a network-local attacker could POST forged findings
// to /ingest-verified with a spoofed X-Tenant-ID header and corrupt the
// findings table.
func ScannerCallbackAuth() gin.HandlerFunc {
	expected := os.Getenv("SCANNER_SERVICE_TOKEN")
	authRequired := os.Getenv("SCANNER_AUTH_REQUIRED") != "false"

	return func(c *gin.Context) {
		// If the global auth middleware already authenticated the caller (JWT
		// or API key), let them through — a real user is making this call.
		if authed, _ := c.Get("authenticated"); authed == true {
			c.Next()
			return
		}

		// Otherwise require the scanner token.
		if !authRequired {
			c.Next()
			return
		}
		if expected == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "service_misconfigured",
				"message": "Scanner callback auth required but no service token configured",
			})
			c.Abort()
			return
		}
		presented := c.GetHeader("X-Scanner-Token")
		if presented == "" || subtle.ConstantTimeCompare([]byte(presented), []byte(expected)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Scanner callbacks require X-Scanner-Token",
			})
			c.Abort()
			return
		}
		// Token-authenticated scanner caller — seed tenant_id from header so
		// downstream handlers see a proper tenant context.
		if hdr := c.GetHeader("X-Tenant-ID"); hdr != "" {
			if parsed, err := uuid.Parse(hdr); err == nil && parsed != uuid.Nil {
				c.Set("tenant_id", parsed)
			}
		}
		if _, ok := c.Get("tenant_id"); !ok {
			c.Set("tenant_id", persistence.DevSystemTenantID)
		}
		c.Set("user_id", uuid.Nil)
		c.Set("user_role", "scanner")
		c.Set("authenticated", true)
		c.Next()
	}
}
