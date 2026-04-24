package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
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

// GetDashboardMetrics godoc
// @Summary Get real-time dashboard metrics
// @Description Returns scan counts, finding totals, risk distribution, and recent activity
// @Tags scanning
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /dashboard/metrics [get]
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

// RiskTrendPoint is one day in the 30-day risk trend.
type RiskTrendPoint struct {
	Date      string `json:"date"`
	Score     int    `json:"score"`
	ScanCount int    `json:"scan_count"`
}

// GetRiskTrend returns daily risk scores for the last N days.
// GET /dashboard/risk-trend?days=30
func (h *DashboardHandler) GetRiskTrend(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	days := 30
	if d := c.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	db := h.pgRepo.GetDB()

	rows, err := db.QueryContext(ctx, `
		SELECT
			DATE(sr.scan_started_at)::text                                    AS day,
			COUNT(DISTINCT sr.id)                                             AS scan_count,
			COALESCE(SUM(CASE WHEN f.severity = 'Critical' THEN 1 ELSE 0 END), 0) AS critical_ct,
			COALESCE(SUM(CASE WHEN f.severity = 'High'     THEN 1 ELSE 0 END), 0) AS high_ct,
			COALESCE(SUM(CASE WHEN f.severity = 'Medium'   THEN 1 ELSE 0 END), 0) AS medium_ct,
			COALESCE(SUM(CASE WHEN f.severity = 'Low'      THEN 1 ELSE 0 END), 0) AS low_ct,
			COUNT(f.id)                                                       AS total_findings
		FROM scan_runs sr
		LEFT JOIN findings f ON f.scan_run_id = sr.id
		WHERE sr.scan_started_at >= NOW() - ($1 || ' days')::INTERVAL
		  AND sr.status = 'completed'
		GROUP BY DATE(sr.scan_started_at)
		ORDER BY day ASC
	`, strconv.Itoa(days))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	defer rows.Close()

	points := []RiskTrendPoint{}
	for rows.Next() {
		var day string
		var scanCount, critCt, highCt, medCt, lowCt, totalFindings int
		if err := rows.Scan(&day, &scanCount, &critCt, &highCt, &medCt, &lowCt, &totalFindings); err != nil {
			continue
		}
		// Weighted score 0–100: heavier weight on critical/high.
		score := 0
		if totalFindings > 0 {
			weighted := critCt*8 + highCt*5 + medCt*2 + lowCt
			maxPossible := totalFindings * 8
			score = weighted * 100 / maxPossible
		}
		points = append(points, RiskTrendPoint{Date: day, Score: score, ScanCount: scanCount})
	}

	c.JSON(http.StatusOK, gin.H{"trend": points, "days": days})
}
