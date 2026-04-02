package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/arc-platform/backend/modules/scanning/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ScanTriggerHandler handles scan trigger requests
type ScanTriggerHandler struct {
	scanService      *service.ScanService
	websocketService interface{} // WebSocket service for broadcasting
}

// Prometheus metrics
var (
	scanTriggerCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scan_trigger_total",
			Help: "Total number of scan triggers",
		},
		[]string{"source_type", "pii_types", "execution_mode"},
	)

	scanTriggerFailureCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scan_trigger_failures_total",
			Help: "Total number of scan trigger failures",
		},
		[]string{"source_type", "error_type"},
	)

	scanTriggerDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "scan_trigger_duration_seconds",
			Help: "Time spent processing scan trigger requests",
		},
		[]string{"source_type"},
	)
)

func NewScanTriggerHandler(scanService *service.ScanService, websocketService interface{}) *ScanTriggerHandler {
	return &ScanTriggerHandler{
		scanService:      scanService,
		websocketService: websocketService,
	}
}

// TriggerScan handles POST /api/v1/scans/trigger
// Accepts scan configuration, creates scan entity, and triggers scanner
func (h *ScanTriggerHandler) TriggerScan(c *gin.Context) {
	start := time.Now()
	defer func() {
		scanTriggerDuration.WithLabelValues("unknown").Observe(time.Since(start).Seconds())
	}()

	var req service.TriggerScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		scanTriggerFailureCounter.WithLabelValues("unknown", "validation_error").Inc()
		log.Printf("ERROR: Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Get user from context (default to "system" if not authenticated)
	triggeredBy := "system"
	if user, exists := c.Get("user_id"); exists {
		if userStr, ok := user.(string); ok {
			triggeredBy = userStr
		}
	}

	// Validate request
	if err := h.validateRequest(&req); err != nil {
		scanTriggerFailureCounter.WithLabelValues("unknown", "validation_error").Inc()
		log.Printf("ERROR: Scan request validation failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Validation failed",
		})
		return
	}

	// Create scan run entity
	ctx := c.Request.Context()
	scanRun, err := h.scanService.CreateScanRun(ctx, &req, triggeredBy)
	if err != nil {
		scanTriggerFailureCounter.WithLabelValues("unknown", "creation_error").Inc()
		log.Printf("ERROR: Failed to create scan run: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create scan run",
		})
		return
	}

	// Record successful trigger
	scanTriggerCounter.WithLabelValues("all_sources", fmt.Sprintf("%v", req.PIITypes), req.ExecutionMode).Inc()

	// Trigger background scan
	go h.executeScan(scanRun.ID, &req)

	c.JSON(http.StatusOK, gin.H{
		"message": "Scan triggered successfully",
		"scan_id": scanRun.ID,
		"status":  "pending",
	})
}

func (h *ScanTriggerHandler) validateRequest(req *service.TriggerScanRequest) error {
	if req.Name == "" {
		return fmt.Errorf("scan name is required")
	}
	if len(req.Sources) == 0 {
		return fmt.Errorf("at least one source is required")
	}
	if len(req.PIITypes) == 0 {
		return fmt.Errorf("at least one PII type is required")
	}
	if req.ExecutionMode != "sequential" && req.ExecutionMode != "parallel" {
		return fmt.Errorf("execution mode must be 'sequential' or 'parallel'")
	}
	return nil
}

func (h *ScanTriggerHandler) executeScan(scanID uuid.UUID, req *service.TriggerScanRequest) {
	log.Printf("Starting scan execution: %s", scanID.String())

	// Build the HTTP payload expected by the python scanner API
	payload := map[string]interface{}{
		"scan_id":        scanID.String(),
		"scan_name":      req.Name,
		"sources":        req.Sources,
		"pii_types":      req.PIITypes,
		"execution_mode": req.ExecutionMode,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR: Failed to serialize scanner payload: %v", err)
		h.markScanFailed(scanID, "payload_serialization_error")
		return
	}

	// Use SCANNER_URL from environment directly — no guessing
	scannerURL := os.Getenv("SCANNER_URL")
	if scannerURL == "" {
		scannerURL = "http://scanner:5002"
	}
	url := fmt.Sprintf("%s/scan", scannerURL)

	// Retry with exponential backoff (3 attempts: 2s, 4s, 8s)
	client := &http.Client{Timeout: 30 * time.Second}
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		log.Printf("Dispatching scan to %s (attempt %d/3)", url, attempt)

		reqHttp, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			log.Printf("ERROR: %v", lastErr)
			break // No point retrying a request construction error
		}
		reqHttp.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(reqHttp)
		if err != nil {
			lastErr = fmt.Errorf("scanner API unreachable: %w", err)
			log.Printf("ERROR: %v (attempt %d/3)", lastErr, attempt)
			if attempt < 3 {
				backoff := time.Duration(1<<uint(attempt)) * time.Second // 2s, 4s
				log.Printf("Retrying in %v...", backoff)
				time.Sleep(backoff)
			}
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("scanner rejected request (%d): %s", resp.StatusCode, string(body))
			log.Printf("ERROR: %v", lastErr)
			break // Scanner explicitly rejected — no point retrying
		}

		// Success
		log.Printf("Scan dispatched successfully: %s", scanID.String())
		return
	}

	// All retries exhausted — mark scan as failed so the UI stops spinning
	log.Printf("ERROR: All dispatch attempts failed for scan %s: %v", scanID.String(), lastErr)
	h.markScanFailed(scanID, "scanner_dispatch_failed")
}

// markScanFailed updates the scan status to "failed" in PostgreSQL
// so the frontend correctly shows a failure badge instead of spinning forever.
func (h *ScanTriggerHandler) markScanFailed(scanID uuid.UUID, reason string) {
	ctx := context.Background()
	if err := h.scanService.UpdateScanStatus(ctx, scanID, "failed"); err != nil {
		log.Printf("ERROR: Failed to mark scan %s as failed: %v", scanID.String(), err)
	} else {
		log.Printf("Scan %s marked as failed (reason: %s)", scanID.String(), reason)
	}
}
