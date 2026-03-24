package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/domain/repository"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DashboardHandler handles dashboard-specific endpoints
type DashboardHandler struct {
	pgRepo *persistence.PostgresRepository
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(pgRepo *persistence.PostgresRepository) *DashboardHandler {
	return &DashboardHandler{
		pgRepo: pgRepo,
	}
}

// DashboardMetrics represents real-time dashboard metrics
type DashboardMetrics struct {
	TotalPII         int       `json:"total_pii"`
	HighRiskFindings int       `json:"high_risk_findings"`
	AssetsHit        int       `json:"assets_hit"`
	ActionsRequired  int       `json:"actions_required"`
	LastScanTime     time.Time `json:"last_scan_time"`
	SystemHealth     string    `json:"system_health"` // "healthy", "degraded", "unhealthy"
}

// GetDashboardMetrics returns real-time dashboard metrics
// GET /api/v1/dashboard/metrics
func (h *DashboardHandler) GetDashboardMetrics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	metrics := DashboardMetrics{
		SystemHealth: "healthy",
	}

	// Extract tenant_id from Gin context
	tenantIDVal, exists := c.Get("tenant_id")
	var tenantID interface{}
	if !exists {
		// For anonymous access, use Nil UUID (default tenant)
		// This matches the behavior where auth middleware allows anonymous requests
		tenantID = uuid.Nil
	} else {
		tenantID = tenantIDVal
	}

	// Add to context using the key expected by EnsureTenantID
	// Based on tenant_helper.go, it uses "tenant_id" key
	ctx = context.WithValue(ctx, "tenant_id", tenantID)

	// Extract environment filter (default to PROD)
	envFilter := c.Query("env")
	if envFilter == "" {
		envFilter = "PROD"
	}

	// Get total PII count (excluding false positives)
	var findings []*entity.Finding
	var err error

	if tenantID == uuid.Nil {
		// Use Global list for system/anonymous view to match ClassificationSummary behavior
		findings, err = h.pgRepo.ListGlobalFindings(ctx, 100000, 0)
	} else {
		findings, err = h.pgRepo.ListFindings(ctx, repository.FindingFilters{}, 100000, 0)
	}

	if err != nil {
		fmt.Printf("❌ Dashboard Metrics Error: Failed to list findings: %v\n", err) // Added logging
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch findings",
		})
		return
	}

	totalPII := 0
	highRiskCount := 0
	assetsMap := make(map[string]bool)
	actionsRequired := 0

	for _, finding := range findings {
		// Strict Isolation: Filter by requested environment
		if finding.Environment != envFilter {
			continue
		}

		// Count all findings (we'll filter by severity instead of status)
		totalPII++

		// Count high risk findings
		if finding.Severity == "Critical" || finding.Severity == "High" {
			highRiskCount++
		}

		// Track unique assets
		assetsMap[finding.AssetID.String()] = true

		// Count actions required (all pending findings that haven't been enriched)
		if finding.EnrichmentScore == nil || *finding.EnrichmentScore < 0.5 {
			actionsRequired++
		}
	}

	metrics.TotalPII = totalPII
	metrics.HighRiskFindings = highRiskCount
	metrics.AssetsHit = len(assetsMap)
	metrics.ActionsRequired = actionsRequired

	// Get last scan time
	var lastScanTime time.Time
	err = h.pgRepo.GetDB().QueryRow(`
		SELECT MAX(created_at) FROM scans WHERE status = 'completed'
	`).Scan(&lastScanTime)
	if err == nil {
		metrics.LastScanTime = lastScanTime
	}

	c.JSON(http.StatusOK, gin.H{
		"data": metrics,
	})
}
