package api

import (
	"net/http"
	"strconv"

	"github.com/arc-platform/backend/modules/scanning/service"
	"github.com/arc-platform/backend/modules/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ScanStatusHandler handles scan status requests
type ScanStatusHandler struct {
	scanService      *service.ScanService
	websocketService interface{}
}

// NewScanStatusHandler creates a new scan status handler
func NewScanStatusHandler(scanService *service.ScanService, websocketService interface{}) *ScanStatusHandler {
	return &ScanStatusHandler{
		scanService:      scanService,
		websocketService: websocketService,
	}
}

// GetScan handles GET /api/v1/scans/:id
// Returns full scan details
func (h *ScanStatusHandler) GetScan(c *gin.Context) {
	scanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scan ID",
		})
		return
	}

	scan, err := h.scanService.GetScanRun(c.Request.Context(), scanID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Scan not found",
		})
		return
	}

	c.JSON(http.StatusOK, scan)
}

// GetScanStatus handles GET /api/v1/scans/:id/status
// Returns lightweight status for polling
func (h *ScanStatusHandler) GetScanStatus(c *gin.Context) {
	scanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scan ID",
		})
		return
	}

	scan, err := h.scanService.GetScanRun(c.Request.Context(), scanID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Scan not found",
		})
		return
	}

	// Progress estimation removed to prevent "invented timing" (Audit Item #2)
	// Frontend should show indeterminate loading state for "running"
	var progress *int
	completed := 100
	zero := 0

	if scan.Status == "completed" {
		progress = &completed
	} else if scan.Status == "pending" {
		progress = &zero
	}
	// For "running" or "failed", leave progress nil

	c.JSON(http.StatusOK, gin.H{
		"scan_id":        scan.ID,
		"status":         scan.Status,
		"progress":       progress,
		"findings_count": scan.TotalFindings,
		"assets_count":   scan.TotalAssets,
	})
}

// ListScans handles GET /api/v1/scans
// Returns a paginated list of scan runs
func (h *ScanStatusHandler) ListScans(c *gin.Context) {
	limit := 10
	if limitQuery := c.Query("limit"); limitQuery != "" {
		if parsedLimit, err := strconv.Atoi(limitQuery); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetQuery := c.Query("offset"); offsetQuery != "" {
		if parsedOffset, err := strconv.Atoi(offsetQuery); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	scans, err := h.scanService.ListScanRuns(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch scan list",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": scans,
	})
}

// CompleteScan handles POST /api/v1/scans/:id/complete
// Updates scan status to completed (called by scanner service)
func (h *ScanStatusHandler) CompleteScan(c *gin.Context) {
	scanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scan ID",
		})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Only allow specific status updates
	if req.Status != "completed" && req.Status != "failed" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid status",
		})
		return
	}

	if err := h.scanService.UpdateScanStatus(c.Request.Context(), scanID, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update scan status",
		})
		return
	}

	// Broadcast completion via WebSocket
	if h.websocketService != nil {
		if wsService, ok := h.websocketService.(*websocket.WebSocketService); ok {
			// Fetch updated scan details
			if scan, err := h.scanService.GetScanRun(c.Request.Context(), scanID); err == nil {
				duration := scan.ScanCompletedAt.Sub(scan.ScanStartedAt)
				wsService.BroadcastScanComplete(scan.ID.String(), scan.TotalFindings, duration)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scan status updated",
	})
}

// CancelScan handles POST /api/v1/scans/:id/cancel
// Cancels a running or pending scan
func (h *ScanStatusHandler) CancelScan(c *gin.Context) {
	scanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scan ID",
		})
		return
	}

	if err := h.scanService.CancelScan(c.Request.Context(), scanID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to cancel scan",
			"details": err.Error(),
		})
		return
	}

	// Broadcast cancellation via WebSocket
	if h.websocketService != nil {
		if wsService, ok := h.websocketService.(*websocket.WebSocketService); ok {
			wsService.BroadcastScanProgress(scanID.String(), 0, "cancelled", "Scan cancelled by user")
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scan cancelled successfully",
		"scan_id": scanID,
	})
}
