package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/arc-platform/backend/modules/scanning/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/arc-platform/backend/modules/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ScanStatusHandler handles scan status requests
type ScanStatusHandler struct {
	scanService      *service.ScanService
	repo             *persistence.PostgresRepository
	websocketService interface{}
	auditLogger      interfaces.AuditLogger
	lineageSync      interfaces.LineageSync
}

// NewScanStatusHandler creates a new scan status handler
func NewScanStatusHandler(scanService *service.ScanService, websocketService interface{}, repo *persistence.PostgresRepository, auditLogger interfaces.AuditLogger, lineageSync interfaces.LineageSync) *ScanStatusHandler {
	if lineageSync == nil {
		lineageSync = &interfaces.NoOpLineageSync{}
	}
	return &ScanStatusHandler{
		scanService:      scanService,
		repo:             repo,
		websocketService: websocketService,
		auditLogger:      auditLogger,
		lineageSync:      lineageSync,
	}
}

func (h *ScanStatusHandler) recordAudit(ctx context.Context, action, resourceID string, meta map[string]interface{}) {
	if h.auditLogger == nil {
		return
	}
	if err := h.auditLogger.Record(ctx, action, "scan", resourceID, meta); err != nil {
		log.Printf("WARN: audit record failed action=%s scan=%s: %v", action, resourceID, err)
	}
}

// syncLineageForScan runs in a background goroutine after a scan reaches a
// terminal state. It queries the distinct asset_ids produced by this scan
// and replays them through the lineage sync, so the Neo4j graph reflects
// newly-discovered assets without waiting for a remediation action.
func (h *ScanStatusHandler) syncLineageForScan(scanID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*60_000_000_000) // 5 min
	defer cancel()

	if h.repo == nil || h.lineageSync == nil {
		return
	}
	rows, err := h.repo.GetDB().QueryContext(ctx, `
		SELECT DISTINCT asset_id FROM findings WHERE scan_run_id = $1 AND asset_id IS NOT NULL
	`, scanID)
	if err != nil {
		log.Printf("WARN: lineage sync query failed for scan %s: %v", scanID, err)
		return
	}
	defer rows.Close()

	synced := 0
	for rows.Next() {
		var assetID uuid.UUID
		if err := rows.Scan(&assetID); err != nil {
			continue
		}
		if err := h.lineageSync.SyncAssetToNeo4j(ctx, assetID); err != nil {
			log.Printf("WARN: lineage sync failed for asset %s (scan %s): %v", assetID, scanID, err)
			continue
		}
		synced++
	}
	log.Printf("Lineage sync complete for scan %s: %d assets synced", scanID, synced)
}

// GetScan godoc
// @Summary Get scan by ID
// @Description Returns full details for a single scan run
// @Tags scanning
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Scan UUID"
// @Success 200 {object} gin.H "Scan details"
// @Failure 400 {object} gin.H "Invalid ID"
// @Failure 404 {object} gin.H "Not found"
// @Security BearerAuth
// @Router /scans/{id} [get]
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

// GetScanStatus godoc
// @Summary Get scan status (lightweight poll)
// @Tags scanning
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Scan UUID"
// @Success 200 {object} gin.H "status, progress, findings_count, assets_count"
// @Failure 404 {object} gin.H "Not found"
// @Security BearerAuth
// @Router /scans/{id}/status [get]
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

// ListScans godoc
// @Summary List scan runs
// @Tags scanning
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param limit query int false "Page size (default 10)"
// @Param offset query int false "Offset"
// @Success 200 {object} gin.H "data: []ScanRun"
// @Security BearerAuth
// @Router /scans [get]
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

// CompleteScan godoc
// @Summary Mark scan complete (scanner callback)
// @Description Called by the Go scanner service to set terminal status. Requires X-Scanner-Token header.
// @Tags scanning
// @Accept json
// @Produce json
// @Param X-Scanner-Token header string true "Scanner service token"
// @Param X-Tenant-ID header string true "Tenant UUID"
// @Param id path string true "Scan UUID"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Router /scans/{id}/complete [post]
func (h *ScanStatusHandler) CompleteScan(c *gin.Context) {
	scanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scan ID",
		})
		return
	}

	var req struct {
		Status      string                 `json:"status" binding:"required"`
		Message     string                 `json:"message"`
		Diagnostics map[string]interface{} `json:"diagnostics"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Only allow specific status updates. "partial" is used when ingestion
	// succeeded for some chunks but not all (P0-5): data is present but
	// incomplete. Surface it distinct from "completed" so the UI can warn.
	if req.Status != "completed" && req.Status != "failed" && req.Status != "partial" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid status",
		})
		return
	}

	// Store scanner diagnostics BEFORE status transition (UpdateScanRun
	// blocks updates once status is "completed")
	if req.Message != "" || len(req.Diagnostics) > 0 {
		if scan, err := h.scanService.GetScanRun(c.Request.Context(), scanID); err == nil {
			if scan.Metadata == nil {
				scan.Metadata = make(map[string]interface{})
			}
			if req.Message != "" {
				scan.Metadata["status_message"] = req.Message
			}
			if len(req.Diagnostics) > 0 {
				scan.Metadata["diagnostics"] = req.Diagnostics
			}
			if err := h.repo.UpdateScanRun(c.Request.Context(), scan); err != nil {
				log.Printf("WARN: Failed to store scan diagnostics for %s: %v", scanID, err)
			}
		}
	}

	if err := h.scanService.UpdateScanStatus(c.Request.Context(), scanID, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update scan status",
		})
		return
	}

	// Emit terminal-state audit event. Maps the three valid terminal statuses
	// to the corresponding action so downstream consumers (compliance, SIEM)
	// can filter. DPDPA Section 8(2) accountability trail.
	action := "SCAN_COMPLETED"
	switch req.Status {
	case "failed":
		action = "SCAN_FAILED"
	case "partial":
		action = "SCAN_PARTIAL"
	}
	meta := map[string]interface{}{"status": req.Status}
	if req.Message != "" {
		meta["message"] = req.Message
	}
	if scan, err := h.scanService.GetScanRun(c.Request.Context(), scanID); err == nil {
		meta["total_findings"] = scan.TotalFindings
		meta["tenant_id"] = scan.TenantID.String()
	}
	h.recordAudit(c.Request.Context(), action, scanID.String(), meta)

	// Kick lineage sync for every asset touched by this scan. Previously sync
	// only ran after remediation, leaving the graph stale for freshly-discovered
	// assets. P1-10. Fire-and-forget; lineage sync errors are logged, not
	// surfaced to the scanner caller.
	if req.Status == "completed" || req.Status == "partial" {
		if h.lineageSync != nil && h.lineageSync.IsAvailable() {
			go h.syncLineageForScan(scanID)
		}
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

// CancelScan godoc
// @Summary Cancel a running or pending scan
// @Tags scanning
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Scan UUID"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Security BearerAuth
// @Router /scans/{id}/cancel [post]
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

// DeleteScan godoc
// @Summary Delete a scan run
// @Tags scanning
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Scan UUID"
// @Success 200 {object} gin.H
// @Failure 404 {object} gin.H
// @Security BearerAuth
// @Router /scans/{id} [delete]
func (h *ScanStatusHandler) DeleteScan(c *gin.Context) {
	scanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scan ID"})
		return
	}

	if err := h.repo.DeleteScanRun(c.Request.Context(), scanID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Scan deleted successfully"})
}

// GetScanPIISummary godoc
// @Summary Get PII type summary for a scan
// @Tags scanning
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Scan UUID"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /scans/{id}/pii-summary [get]
func (h *ScanStatusHandler) GetScanPIISummary(c *gin.Context) {
	scanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scan ID"})
		return
	}

	summary, err := h.repo.GetScanPIISummary(c.Request.Context(), scanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get PII summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": summary})
}

// ProgressEvent is the payload from the Go scanner reporting scan progress.
type ProgressEvent struct {
	ScanID        string  `json:"scan_id"`
	FindingsFound int     `json:"findings_found"`
	Source        string  `json:"current_source"`
	PercentDone   float64 `json:"percent_done"`
}

// ReceiveProgressEvent godoc
// @Summary Receive live scan progress event (scanner callback)
// @Description Called by Go scanner every N findings to broadcast WebSocket progress. Requires X-Scanner-Token.
// @Tags scanning
// @Accept json
// @Produce json
// @Param X-Scanner-Token header string true "Scanner token"
// @Param id path string true "Scan UUID"
// @Success 200 {object} gin.H
// @Router /scans/{id}/progress-event [post]
func (h *ScanStatusHandler) ReceiveProgressEvent(c *gin.Context) {
	scanID := c.Param("id")
	var event ProgressEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	if event.ScanID == "" {
		event.ScanID = scanID
	}

	if h.websocketService != nil {
		if wsService, ok := h.websocketService.(*websocket.WebSocketService); ok {
			msg := fmt.Sprintf("Found %d findings in %s", event.FindingsFound, event.Source)
			wsService.BroadcastScanProgress(event.ScanID, int(event.PercentDone), "running", msg)
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
