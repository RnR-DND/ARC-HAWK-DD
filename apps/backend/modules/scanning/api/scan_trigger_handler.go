package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/arc-platform/backend/modules/scanning/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

// ScanTriggerHandler handles scan trigger requests
type ScanTriggerHandler struct {
	scanService      *service.ScanService
	websocketService interface{} // WebSocket service for broadcasting
}

// Prometheus metrics
var (
	scanTriggerCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scan_trigger_total",
			Help: "Total number of scan triggers",
		},
		[]string{"source_type", "pii_types", "execution_mode"},
	)

	scanTriggerFailureCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scan_trigger_failures_total",
			Help: "Total number of scan trigger failures",
		},
		[]string{"source_type", "error_type"},
	)

	scanTriggerDuration = prometheus.NewHistogramVec(
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
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
			"error":   "Failed to create scan run",
			"details": err.Error(),
		})
		return
	}

	// Record successful trigger
	scanTriggerCounter.WithLabelValues(req.ExecutionMode, fmt.Sprintf("%v", req.PIITypes)).Inc()

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
	// Log scan start
	log.Printf("Starting scan execution: %s", scanID.String())

	// Create scanner config directory
	configDir := filepath.Join("/tmp", "scan_configs", scanID.String())
	os.MkdirAll(configDir, 0755)

	// Write configuration to file
	configFile := filepath.Join(configDir, "config.yml")
	configData := map[string]interface{}{
		"sources": map[string]interface{}{
			"fs": map[string]interface{}{
				"real_file_system": map[string]interface{}{
					"path":            "/Users/prathameshyadav/ARC-Hawk/apps/scanner/real_test_data",
					"recursive":       true,
					"file_extensions": []string{".txt", ".csv", ".json", ".xml", ".log"},
				},
			},
		},
		"pii_types":      req.PIITypes,
		"execution_mode": req.ExecutionMode,
	}

	configBytes, _ := json.Marshal(configData)
	err := ioutil.WriteFile(configFile, configBytes, 0644)
	if err != nil {
		log.Printf("ERROR: Failed to write scanner config: %v", err)
		return
	}

	// Trigger the actual scanner
	cmd := exec.Command("python", "-m", "hawk_scanner.main", "fs", "--connection", configFile)
	cmd.Dir = "../scanner" // Run from the scanner directory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Printf("ERROR: Scanner execution failed: %v", err)
	}

	log.Printf("Scan execution completed: %s", scanID.String())
}
