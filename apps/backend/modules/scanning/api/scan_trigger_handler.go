package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/arc-platform/backend/modules/scanning/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/encryption"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/infrastructure/vault"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/arc-platform/backend/modules/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ScanTriggerHandler handles scan trigger requests
type ScanTriggerHandler struct {
	scanService      *service.ScanService
	websocketService any // WebSocket service for broadcasting
	repo             *persistence.PostgresRepository
	encryption       *encryption.EncryptionService
	vault            *vault.Client
	patternsService  *service.PatternsService
	auditLogger      interfaces.AuditLogger
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

	classificationConfidenceHist = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "classification_confidence",
		Help:    "Distribution of PII classification confidence scores emitted by the scanner",
		Buckets: []float64{0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
	})
)

func NewScanTriggerHandler(scanService *service.ScanService, websocketService any, repo *persistence.PostgresRepository, enc *encryption.EncryptionService, vaultClient *vault.Client, auditLogger interfaces.AuditLogger) *ScanTriggerHandler {
	return &ScanTriggerHandler{
		scanService:      scanService,
		websocketService: websocketService,
		repo:             repo,
		encryption:       enc,
		vault:            vaultClient,
		patternsService:  service.NewPatternsService(repo),
		auditLogger:      auditLogger,
	}
}

// recordAudit is a helper that swallows audit errors (never break a scan on an
// audit failure) but logs them so operators can detect chain gaps.
func (h *ScanTriggerHandler) recordAudit(ctx context.Context, action, resourceType, resourceID string, metadata map[string]interface{}) {
	if h.auditLogger == nil {
		return
	}
	if err := h.auditLogger.Record(ctx, action, resourceType, resourceID, metadata); err != nil {
		log.Printf("WARN: audit record failed action=%s resource=%s/%s: %v", action, resourceType, resourceID, err)
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

	// Extract tenant ID before goroutine — Gin context must not be used inside goroutines.
	tenantID := tenantIDFromCtx(c)

	// Per-tenant concurrent-scan cap. Prevents a single tenant from spawning
	// unbounded scan goroutines and OOMing the backend for every tenant.
	release, ok := getTenantScanLimiter().TryAcquire(tenantID)
	if !ok {
		scanTriggerFailureCounter.WithLabelValues("unknown", "tenant_rate_limited").Inc()
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":   "tenant_scan_limit",
			"message": "Too many concurrent scans for this tenant. Wait for one to finish before starting another.",
		})
		return
	}

	// Trigger background scan; limiter released when goroutine exits.
	go func() {
		defer release()
		h.executeScan(scanRun.ID, &req, tenantID)
	}()

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

// tenantIDFromCtx extracts the tenant UUID from the Gin context.
// Must be called on the HTTP goroutine (before go h.executeScan).
func tenantIDFromCtx(c *gin.Context) uuid.UUID {
	if v, ok := c.Get("tenant_id"); ok {
		switch t := v.(type) {
		case uuid.UUID:
			return t
		case string:
			if id, err := uuid.Parse(t); err == nil {
				return id
			}
		}
	}
	return persistence.DevSystemTenantID
}

func (h *ScanTriggerHandler) executeScan(scanID uuid.UUID, req *service.TriggerScanRequest, tenantID uuid.UUID) {
	// Apply a 35-minute context timeout so this goroutine cannot run forever
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Minute)
	defer cancel()
	log.Printf("Starting scan execution: %s", scanID.String())

	// Audit the scan lifecycle. SCAN_STARTED captures actor, tenant, and
	// sources at dispatch time. DPDPA Section 8(2) requires an immutable
	// record of which data was accessed when and by whom.
	h.recordAudit(ctx, "SCAN_STARTED", "scan", scanID.String(), map[string]interface{}{
		"tenant_id":      tenantID.String(),
		"scan_name":      req.Name,
		"source_count":   len(req.Sources),
		"sources":        req.Sources,
		"execution_mode": req.ExecutionMode,
		"pii_types":      req.PIITypes,
	})

	// Broadcast scan start via WebSocket
	if h.websocketService != nil {
		if wsService, ok := h.websocketService.(*websocket.WebSocketService); ok {
			wsService.BroadcastScanStarted(scanID.String(), req.Name, len(req.Sources))
		}
	}

	// Resolve full connection configs (including passwords) from the database.
	// Credentials are passed in-memory over the internal Docker network,
	// never written to disk (maintaining C-6 audit compliance).
	connectionConfigs, err := h.resolveConnectionConfigs(req.Sources)
	if err != nil {
		log.Printf("ERROR: Connection resolution failed for scan %s: %v", scanID.String(), err)
		h.markScanFailed(scanID, "connection_resolution_failed")
		return
	}
	if len(connectionConfigs) == 0 && len(req.Sources) > 0 {
		log.Printf("WARN: No runtime connection configs resolved for scan %s — scanner will use connection.yml fallback", scanID.String())
	}

	// Resolve custom patterns for this tenant (tenantID was captured from Gin context before goroutine spawn)
	var customPatternsList []map[string]any
	if h.patternsService != nil {
		if patterns, pErr := h.patternsService.GetActivePatterns(ctx, tenantID); pErr == nil {
			for _, p := range patterns {
				customPatternsList = append(customPatternsList, map[string]any{
					"name":              p.Name,
					"display_name":      p.DisplayName,
					"regex":             p.Regex,
					"category":          p.Category,
					"context_keywords":  p.ContextKeywords,
					"negative_keywords": p.NegativeKeywords,
				})
			}
		} else {
			log.Printf("WARN: failed to load custom patterns for scan %s: %v", scanID, pErr)
		}
	}

	// Build the HTTP payload expected by the Go scanner API
	payload := map[string]any{
		"scan_id":             scanID.String(),
		"scan_name":           req.Name,
		"tenant_id":           tenantID.String(),
		"sources":             req.Sources,
		"pii_types":           req.PIITypes,
		"execution_mode":      req.ExecutionMode,
		"connection_configs":  connectionConfigs,
		"custom_patterns":     customPatternsList,
		"classification_mode": req.ClassificationMode,
	}
	if len(req.PIITypesPerSource) > 0 {
		payload["pii_types_per_source"] = req.PIITypesPerSource
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR: Failed to serialize scanner payload: %v", err)
		h.markScanFailed(scanID, "payload_serialization_error")
		return
	}

	// Use SCANNER_URL from environment, defaulting to Go scanner
	scannerURL := os.Getenv("SCANNER_URL")
	if scannerURL == "" {
		scannerURL = "http://go-scanner:8001"
	}
	url := fmt.Sprintf("%s/scan", scannerURL)

	// Retry with exponential backoff (3 attempts: 2s, 4s, 8s)
	client := &http.Client{Timeout: 30 * time.Second}
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		log.Printf("Dispatching scan to %s (attempt %d/3)", url, attempt)

		reqHttp, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			log.Printf("ERROR: %v", lastErr)
			break // No point retrying a request construction error
		}
		reqHttp.Header.Set("Content-Type", "application/json")
		// Authenticate to the scanner using the shared service token. The
		// scanner's ServiceTokenAuth middleware (apps/goScanner/api/auth_middleware.go)
		// rejects any /scan call without a matching token in release mode.
		if token := os.Getenv("SCANNER_SERVICE_TOKEN"); token != "" {
			reqHttp.Header.Set("X-Scanner-Token", token)
		}

		resp, err := client.Do(reqHttp)
		if err != nil {
			lastErr = fmt.Errorf("scanner API unreachable: %w", err)
			log.Printf("ERROR: %v (attempt %d/3)", lastErr, attempt)
			if attempt < 3 {
				jitter := time.Duration(rand.Intn(500)) * time.Millisecond
				backoff := time.Duration(1<<uint(attempt))*time.Second + jitter
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

		// Success — update DB to "running" AND broadcast via WebSocket
		log.Printf("Scan dispatched successfully: %s", scanID.String())
		if err := h.scanService.UpdateScanStatus(ctx, scanID, "running"); err != nil {
			log.Printf("WARN: Failed to update scan %s to running: %v", scanID.String(), err)
		}
		if h.websocketService != nil {
			if wsService, ok := h.websocketService.(*websocket.WebSocketService); ok {
				wsService.BroadcastScanProgress(scanID.String(), 0, "running", "Scan dispatched to scanner")
			}
		}
		return
	}

	// All retries exhausted — mark scan as failed so the UI stops spinning
	log.Printf("ERROR: All dispatch attempts failed for scan %s: %v", scanID.String(), lastErr)
	h.markScanFailed(scanID, "scanner_dispatch_failed")
}

// resolveConnectionConfigs looks up connection profiles and returns a map of
// source_type → profile_name → full config.
// When Vault is enabled, credentials come exclusively from Vault.
// When Vault is disabled, credentials are decrypted from PostgreSQL.
func (h *ScanTriggerHandler) resolveConnectionConfigs(sourceNames []string) (map[string]map[string]any, error) {
	configs := make(map[string]map[string]any)

	if h.repo == nil || h.encryption == nil {
		log.Printf("WARN: repo or encryption not available, scanner will use connection.yml fallback")
		return configs, nil
	}

	ctx := context.Background()
	connections, err := h.repo.ListConnections(ctx)
	if err != nil {
		return configs, fmt.Errorf("failed to list connections: %w", err)
	}

	sourceSet := make(map[string]bool)
	for _, s := range sourceNames {
		sourceSet[s] = true
	}

	vaultEnabled := h.vault != nil && h.vault.IsEnabled()

	for _, conn := range connections {
		if !sourceSet[conn.ProfileName] {
			continue
		}

		var config map[string]any

		if vaultEnabled {
			vc, vErr := h.vault.ReadConnectionSecret(conn.SourceType, conn.ProfileName)
			if vErr != nil {
				log.Printf("ERROR: Vault read failed for %s/%s: %v", conn.SourceType, conn.ProfileName, vErr)
				continue
			}
			if vc == nil {
				log.Printf("WARN: Credentials not found in Vault for %s/%s — skipping", conn.SourceType, conn.ProfileName)
				continue
			}
			config = vc
			log.Printf("INFO: Resolved connection config for %s/%s from Vault", conn.SourceType, conn.ProfileName)
		} else {
			if err := h.encryption.Decrypt(conn.ConfigEncrypted, &config); err != nil {
				log.Printf("WARN: Failed to decrypt config for %s/%s: %v", conn.SourceType, conn.ProfileName, err)
				continue
			}
			log.Printf("INFO: Resolved connection config for %s/%s from PostgreSQL", conn.SourceType, conn.ProfileName)
		}

		if configs[conn.SourceType] == nil {
			configs[conn.SourceType] = make(map[string]any)
		}
		configs[conn.SourceType][conn.ProfileName] = config
	}

	return configs, nil
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
	// Broadcast failure via WebSocket so frontend updates immediately
	if h.websocketService != nil {
		if wsService, ok := h.websocketService.(*websocket.WebSocketService); ok {
			wsService.BroadcastScanProgress(scanID.String(), 0, "failed", reason)
		}
	}
}

// GetScanDelta returns findings added/removed since the previous scan.
// Returns 204 if there is no previous scan to compare against.
func (h *ScanTriggerHandler) GetScanDelta(c *gin.Context) {
	scanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid scan ID"})
		return
	}
	delta, err := h.scanService.GetScanDelta(c.Request.Context(), scanID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if delta == nil {
		c.Status(204)
		return
	}
	c.JSON(200, delta)
}
