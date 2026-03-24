package api

import (
	"log"
	"net/http"

	"github.com/arc-platform/backend/modules/connections/service"
	"github.com/gin-gonic/gin"
)

// ConnectionSyncHandler handles connection sync operations
type ConnectionSyncHandler struct {
	syncService *service.ConnectionSyncService
}

// NewConnectionSyncHandler creates a new connection sync handler
func NewConnectionSyncHandler(syncService *service.ConnectionSyncService) *ConnectionSyncHandler {
	return &ConnectionSyncHandler{
		syncService: syncService,
	}
}

// SyncToScanner handles POST /api/v1/connections/sync
// Manually triggers sync of database connections to scanner YAML
func (h *ConnectionSyncHandler) SyncToScanner(c *gin.Context) {
	ctx := c.Request.Context()

	if err := h.syncService.SyncToYAML(ctx); err != nil {
		log.Printf("ERROR: Failed to sync connections: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to sync connections to scanner",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Connections synced to scanner successfully",
	})
}

// ValidateSync handles GET /api/v1/connections/sync/validate
// Validates that YAML file is in sync with database
func (h *ConnectionSyncHandler) ValidateSync(c *gin.Context) {
	ctx := c.Request.Context()

	inSync, err := h.syncService.ValidateSync(ctx)
	if err != nil {
		log.Printf("ERROR: Failed to validate sync: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to validate sync status",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"in_sync": inSync,
		"message": func() string {
			if inSync {
				return "Scanner configuration is in sync with database"
			}
			return "Scanner configuration is out of sync - sync required"
		}(),
	})
}
