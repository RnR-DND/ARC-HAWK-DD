package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

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

	envFilter := c.Query("env")
	db := h.pgRepo.GetDB()

	// Build WHERE clause for tenant + environment filtering
	var tenantID uuid.UUID
	if val, ok := c.Get("tenant_id"); ok {
		if id, canCast := val.(uuid.UUID); canCast {
			tenantID = id
		}
	}

	baseWhere := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if tenantID != uuid.Nil {
		baseWhere += fmt.Sprintf(" AND tenant_id = $%d", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if envFilter != "" {
		baseWhere += fmt.Sprintf(" AND environment = $%d", argIdx)
		args = append(args, envFilter)
		argIdx++
	}

	// Total PII count (SQL COUNT instead of loading all into memory)
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM findings "+baseWhere, args...).Scan(&metrics.TotalPII); err != nil {
		fmt.Printf("WARN: Failed to count findings: %v\n", err)
	}

	// Critical findings only (not High) — so the number differs from total
	critArgs := make([]any, len(args))
	copy(critArgs, args)
	critArgs = append(critArgs, "Critical")
	critQuery := fmt.Sprintf("SELECT COUNT(*) FROM findings %s AND severity = $%d", baseWhere, argIdx)
	if err := db.QueryRowContext(ctx, critQuery, critArgs...).Scan(&metrics.HighRiskFindings); err != nil {
		fmt.Printf("WARN: Failed to count critical findings: %v\n", err)
	}

	// Unique assets hit
	if err := db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT asset_id) FROM findings "+baseWhere, args...).Scan(&metrics.AssetsHit); err != nil {
		fmt.Printf("WARN: Failed to count assets: %v\n", err)
	}

	// Remediation tasks: Critical+High findings not yet reviewed/remediated
	remArgs := make([]any, len(args))
	copy(remArgs, args)
	remArgs = append(remArgs, "Critical", "High")
	remQuery := fmt.Sprintf(`SELECT COUNT(*) FROM findings f %s AND f.severity IN ($%d, $%d) AND NOT EXISTS (
		SELECT 1 FROM review_states rs WHERE rs.finding_id = f.id AND rs.status IN ('confirmed', 'false_positive', 'remediated')
	)`, baseWhere, argIdx, argIdx+1)
	if err := db.QueryRowContext(ctx, remQuery, remArgs...).Scan(&metrics.ActionsRequired); err != nil {
		fmt.Printf("WARN: Failed to count actions required: %v\n", err)
	}

	// Last scan time
	var lastScanTime time.Time
	var scanErr error
	if tenantID != uuid.Nil {
		scanErr = db.QueryRowContext(ctx,
			`SELECT COALESCE(MAX(created_at), '0001-01-01') FROM scan_runs WHERE status = 'completed' AND tenant_id = $1`,
			tenantID,
		).Scan(&lastScanTime)
	} else {
		scanErr = db.QueryRowContext(ctx,
			`SELECT COALESCE(MAX(created_at), '0001-01-01') FROM scan_runs WHERE status = 'completed'`,
		).Scan(&lastScanTime)
	}
	if scanErr == nil {
		metrics.LastScanTime = lastScanTime
	}

	c.JSON(http.StatusOK, metrics)
}
