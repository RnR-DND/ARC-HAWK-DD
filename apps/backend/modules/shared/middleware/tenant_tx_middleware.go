package middleware

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TenantTxMiddleware wraps every authenticated request in a DB transaction
// with SET LOCAL so RLS tenant isolation is truly scoped per-request.
//
// Handlers that need the transaction (e.g. to satisfy RLS policies) retrieve
// it via: tx := c.MustGet("tx").(*sql.Tx)
//
// Handlers that continue to use the module-level *sql.DB still get application-
// layer tenant filtering (WHERE tenant_id = $n) but do NOT get DB-layer RLS
// protection. Migrate hot paths to use "tx" from context over time.
func TenantTxMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, exists := c.Get("tenant_id")
		if !exists || raw == nil {
			c.Next()
			return
		}

		var tenantID string
		switch v := raw.(type) {
		case string:
			if v == "" {
				c.Next()
				return
			}
			tenantID = v
		case uuid.UUID:
			if v == uuid.Nil {
				c.Next()
				return
			}
			tenantID = v.String()
		default:
			c.Next()
			return
		}

		tx, err := db.BeginTx(c.Request.Context(), nil)
		if err != nil {
			log.Printf("ERROR: TenantTxMiddleware BeginTx: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "db unavailable"})
			return
		}

		if _, err := tx.ExecContext(c.Request.Context(),
			"SET LOCAL app.current_tenant_id = $1", tenantID); err != nil {
			_ = tx.Rollback()
			log.Printf("ERROR: TenantTxMiddleware SET LOCAL: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "rls setup failed"})
			return
		}

		c.Set("tx", tx)

		defer func() {
			if c.Writer.Status() >= 500 {
				_ = tx.Rollback()
			} else {
				_ = tx.Commit()
			}
		}()

		c.Next()
	}
}
