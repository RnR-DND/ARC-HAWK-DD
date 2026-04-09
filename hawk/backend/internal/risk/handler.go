package risk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/arc/hawk/backend/internal/auth"
	"github.com/arc/hawk/backend/internal/shared"
)

// Risk score weights from the architecture spec.
const (
	WeightPIIDensity          = 0.35
	WeightSensitivity         = 0.30
	WeightAccessExposure      = 0.20
	WeightRetentionViolation  = 0.15
	RetentionViolationBonus   = 15.0 // +15 points for Sec 17 violation
)

// Tier thresholds.
const (
	TierCriticalMin = 80
	TierHighMin     = 60
	TierMediumMin   = 40
	TierLowMin      = 0
)

// RiskScore represents a computed risk score with component breakdown.
type RiskScore struct {
	AssetID             string  `json:"asset_id"`
	AssetName           string  `json:"asset_name"`
	Score               float64 `json:"score"`
	Tier                string  `json:"tier"`
	PIIDensity          float64 `json:"pii_density"`
	PIIDensityWeighted  float64 `json:"pii_density_weighted"`
	SensitivityWeight   float64 `json:"sensitivity_weight"`
	SensitivityWeighted float64 `json:"sensitivity_weighted"`
	AccessExposure      float64 `json:"access_exposure"`
	AccessExposureWeighted float64 `json:"access_exposure_weighted"`
	RetentionViolation  float64 `json:"retention_violation"`
	RetentionWeighted   float64 `json:"retention_weighted"`
	PreviousTier        string  `json:"previous_tier,omitempty"`
	TierChanged         bool    `json:"tier_changed"`
	LastCalculatedAt    time.Time `json:"last_calculated_at"`
}

// RiskScoresResponse wraps the risk score list with summary stats.
type RiskScoresResponse struct {
	Scores       []RiskScore       `json:"scores"`
	Summary      RiskSummary       `json:"summary"`
	Pagination   shared.Pagination `json:"pagination"`
	Total        int64             `json:"total"`
}

// RiskSummary provides aggregate risk statistics.
type RiskSummary struct {
	CriticalCount int     `json:"critical_count"`
	HighCount     int     `json:"high_count"`
	MediumCount   int     `json:"medium_count"`
	LowCount      int     `json:"low_count"`
	AverageScore  float64 `json:"average_score"`
}

// RegisterRoutes registers risk-related HTTP handlers.
func RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/risk/scores", getRiskScores)
}

// getRiskScores returns risk scores with component breakdown.
func getRiskScores(c *gin.Context) {
	policy := auth.GetPolicy(c)
	pg := shared.ParsePagination(c)

	policyFilter, policyArgs := auth.PolicySQLFilter(policy, "a.id", "a.risk_tier")

	// Get summary counts
	var summary RiskSummary
	err := shared.ReadDB().QueryRow(c.Request.Context(), fmt.Sprintf(`
		SELECT
			COUNT(*) FILTER (WHERE a.risk_tier = 'Critical'),
			COUNT(*) FILTER (WHERE a.risk_tier = 'High'),
			COUNT(*) FILTER (WHERE a.risk_tier = 'Medium'),
			COUNT(*) FILTER (WHERE a.risk_tier = 'Low'),
			COALESCE(AVG(a.risk_score), 0)
		FROM assets a WHERE %s
	`, policyFilter), policyArgs...).Scan(
		&summary.CriticalCount, &summary.HighCount,
		&summary.MediumCount, &summary.LowCount,
		&summary.AverageScore,
	)
	if err != nil {
		shared.ErrInternal(c, "failed to compute risk summary", err)
		return
	}

	// Count total
	var total int64
	err = shared.ReadDB().QueryRow(c.Request.Context(), fmt.Sprintf(`
		SELECT COUNT(*) FROM assets a WHERE %s
	`, policyFilter), policyArgs...).Scan(&total)
	if err != nil {
		shared.ErrInternal(c, "failed to count assets", err)
		return
	}

	// Apply optional tier filter
	tierFilter := c.Query("tier")
	extraCondition := ""
	fetchArgs := append([]any{}, policyArgs...)
	if tierFilter != "" {
		fetchArgs = append(fetchArgs, tierFilter)
		extraCondition = fmt.Sprintf(" AND a.risk_tier = $%d", len(fetchArgs))
	}

	// Sort by score descending by default
	sortDir := "DESC"
	if dir := c.Query("dir"); dir == "asc" {
		sortDir = "ASC"
	}

	fetchArgs = append(fetchArgs, pg.PageSize, pg.Offset)
	rows, err := shared.ReadDB().Query(c.Request.Context(), fmt.Sprintf(`
		SELECT a.id, a.name, a.risk_score, a.risk_tier,
		       a.pii_density, a.sensitivity_weight, a.access_exposure,
		       CASE WHEN a.retention_policy_days IS NULL OR a.retention_policy_days <= 0 THEN 100.0 ELSE 0.0 END as retention_violation,
		       COALESCE(a.previous_risk_tier, '') as previous_tier,
		       a.risk_tier != COALESCE(a.previous_risk_tier, a.risk_tier) as tier_changed,
		       COALESCE(a.risk_calculated_at, a.updated_at) as last_calculated_at
		FROM assets a
		WHERE %s%s
		ORDER BY a.risk_score %s
		LIMIT $%d OFFSET $%d
	`, policyFilter, extraCondition, sortDir, len(fetchArgs)-1, len(fetchArgs)),
		fetchArgs...)
	if err != nil {
		shared.ErrInternal(c, "failed to fetch risk scores", err)
		return
	}
	defer rows.Close()

	var scores []RiskScore
	for rows.Next() {
		var rs RiskScore
		var retentionViolation float64
		if err := rows.Scan(
			&rs.AssetID, &rs.AssetName, &rs.Score, &rs.Tier,
			&rs.PIIDensity, &rs.SensitivityWeight, &rs.AccessExposure,
			&retentionViolation,
			&rs.PreviousTier, &rs.TierChanged, &rs.LastCalculatedAt,
		); err != nil {
			shared.ErrInternal(c, "failed to scan risk score row", err)
			return
		}

		rs.RetentionViolation = retentionViolation
		rs.PIIDensityWeighted = rs.PIIDensity * WeightPIIDensity
		rs.SensitivityWeighted = rs.SensitivityWeight * WeightSensitivity
		rs.AccessExposureWeighted = rs.AccessExposure * WeightAccessExposure
		rs.RetentionWeighted = rs.RetentionViolation * WeightRetentionViolation

		scores = append(scores, rs)
	}

	if scores == nil {
		scores = []RiskScore{}
	}

	c.JSON(http.StatusOK, RiskScoresResponse{
		Scores:     scores,
		Summary:    summary,
		Pagination: pg,
		Total:      total,
	})
}

// RecalculateRiskScores recalculates risk scores for all assets affected by a scan.
// Called after every scan completion.
func RecalculateRiskScores(ctx context.Context, scanJobID string) error {
	shared.RiskScoreRecalculations.Inc()
	db := shared.WriteDB()
	cfg := shared.LoadConfig()

	// Get all assets affected by this scan
	rows, err := db.Query(ctx, `
		SELECT DISTINCT a.id, a.pii_field_count, a.field_count,
		       a.sensitivity_weight, a.access_exposure,
		       a.retention_policy_days, a.risk_tier
		FROM assets a
		JOIN scan_results sr ON sr.asset_id = a.id
		WHERE sr.scan_job_id = $1
	`, scanJobID)
	if err != nil {
		return fmt.Errorf("query affected assets: %w", err)
	}
	defer rows.Close()

	type assetRisk struct {
		id                  string
		piiCount            int
		fieldCount          int
		sensitivityWeight   float64
		accessExposure      float64
		retentionPolicyDays *int
		currentTier         string
	}

	var assets []assetRisk
	for rows.Next() {
		var ar assetRisk
		if err := rows.Scan(&ar.id, &ar.piiCount, &ar.fieldCount,
			&ar.sensitivityWeight, &ar.accessExposure,
			&ar.retentionPolicyDays, &ar.currentTier); err != nil {
			return fmt.Errorf("scan asset row: %w", err)
		}
		assets = append(assets, ar)
	}

	now := time.Now().UTC()
	batch := &pgx.Batch{}

	for _, a := range assets {
		// Calculate PII density (0-100 scale)
		piiDensity := 0.0
		if a.fieldCount > 0 {
			piiDensity = float64(a.piiCount) / float64(a.fieldCount) * 100
		}

		// Retention violation: 100 if no policy, 0 if policy exists
		retentionViolation := 100.0
		if a.retentionPolicyDays != nil && *a.retentionPolicyDays > 0 {
			retentionViolation = 0.0
		}

		// Calculate score using exact formula
		score := (piiDensity * WeightPIIDensity) +
			(a.sensitivityWeight * WeightSensitivity) +
			(a.accessExposure * WeightAccessExposure) +
			(retentionViolation * WeightRetentionViolation)

		// Apply Sec 17 retention violation bonus
		if retentionViolation > 0 {
			score += RetentionViolationBonus
		}

		// Clamp to 0-100
		if score > 100 {
			score = 100
		}
		if score < 0 {
			score = 0
		}

		// Determine tier
		newTier := CalculateTier(score)

		batch.Queue(`
			UPDATE assets
			SET risk_score = $1, risk_tier = $2, pii_density = $3,
			    previous_risk_tier = risk_tier, risk_calculated_at = $4
			WHERE id = $5
		`, score, newTier, piiDensity, now, a.id)

		// Alert on tier change
		if newTier != a.currentTier && a.currentTier != "" {
			alertTierChange(ctx, cfg, a.id, a.currentTier, newTier, score)
		}
	}

	if batch.Len() > 0 {
		br := db.SendBatch(ctx, batch)
		defer br.Close()
		for i := 0; i < batch.Len(); i++ {
			if _, err := br.Exec(); err != nil {
				shared.Sugar().Errorw("failed to update risk score",
					"error", err, "batch_index", i)
			}
		}
	}

	return nil
}

// CalculateTier determines the risk tier from a score.
func CalculateTier(score float64) string {
	switch {
	case score >= TierCriticalMin:
		return "Critical"
	case score >= TierHighMin:
		return "High"
	case score >= TierMediumMin:
		return "Medium"
	default:
		return "Low"
	}
}

// alertTierChange sends webhook and email alerts when an asset's risk tier changes.
func alertTierChange(ctx context.Context, cfg *shared.Config, assetID, oldTier, newTier string, score float64) {
	shared.Sugar().Infow("risk tier changed",
		"asset_id", assetID,
		"old_tier", oldTier,
		"new_tier", newTier,
		"score", score,
	)

	// Record the tier change event
	db := shared.WriteDB()
	_, _ = db.Exec(ctx, `
		INSERT INTO risk_tier_changes (asset_id, old_tier, new_tier, score, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, assetID, oldTier, newTier, score, time.Now().UTC())

	// Send webhook notification if configured
	if cfg.WebhookURL != "" {
		go sendWebhookAlert(cfg.WebhookURL, assetID, oldTier, newTier, score)
	}

	// Publish email alert task to Celery
	_ = shared.PublishCeleryTask(ctx, "escalation", shared.CeleryTask{
		ID:   fmt.Sprintf("tier-change-%s-%d", assetID, time.Now().UnixNano()),
		Task: "hawk.tasks.send_tier_change_email",
		Kwargs: map[string]any{
			"asset_id": assetID,
			"old_tier": oldTier,
			"new_tier": newTier,
			"score":    score,
		},
	})
}

// sendWebhookAlert sends a POST request to the configured webhook URL.
func sendWebhookAlert(webhookURL, assetID, oldTier, newTier string, score float64) {
	payload, _ := json.Marshal(map[string]any{
		"event":    "risk_tier_change",
		"asset_id": assetID,
		"old_tier": oldTier,
		"new_tier": newTier,
		"score":    score,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		shared.Sugar().Errorw("webhook alert failed", "error", err, "asset_id", assetID)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		shared.Sugar().Warnw("webhook returned non-2xx",
			"status", resp.StatusCode, "asset_id", assetID)
	}
}
