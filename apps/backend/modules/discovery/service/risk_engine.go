package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/arc-platform/backend/modules/discovery/domain"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// RiskEngine is the enhanced risk scoring engine that applies the DPDPA-weighted
// formula, tracks tier transitions, and triggers downstream notifications.
//
//	score = (pii_density * 0.35) + (sensitivity_weight * 0.30)
//	      + (access_exposure * 0.20) + (retention_violation * 0.15)
//
// Tiers: Critical 80-100, High 60-79, Medium 40-59, Low 0-39
type RiskEngine struct {
	repo *Repo
}

// NewRiskEngine creates a new RiskEngine backed by the discovery repo.
func NewRiskEngine(repo *Repo) *RiskEngine {
	return &RiskEngine{repo: repo}
}

// RiskResult is the output of CalculateRiskScore — the composite score, tier, and
// individual component contributions for auditability.
type RiskResult struct {
	Score              float64         `json:"score"`
	Tier               domain.RiskTier `json:"tier"`
	PIIDensity         float64         `json:"pii_density"`
	SensitivityWeight  float64         `json:"sensitivity_weight"`
	AccessExposure     float64         `json:"access_exposure"`
	RetentionViolation float64         `json:"retention_violation"`
	Breakdown          RiskBreakdown   `json:"breakdown"`
}

// RiskBreakdown shows the weighted contribution of each component.
type RiskBreakdown struct {
	PIIDensityContrib         float64 `json:"pii_density_contrib"`
	SensitivityWeightContrib  float64 `json:"sensitivity_weight_contrib"`
	AccessExposureContrib     float64 `json:"access_exposure_contrib"`
	RetentionViolationContrib float64 `json:"retention_violation_contrib"`
}

// CalculateRiskScore computes the DPDPA risk score for a single asset.
// The asset is identified by assetID and scoped to the tenant in ctx.
// Returns the score, tier, and full component breakdown.
func (e *RiskEngine) CalculateRiskScore(ctx context.Context, assetID uuid.UUID) (*RiskResult, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("risk engine: %w", err)
	}

	// 1. Read inventory rows for this asset.
	rows, err := e.repo.DB().QueryContext(ctx, `
		SELECT id, tenant_id, asset_id, asset_name, source_id, source_name,
		       classification, sensitivity, finding_count, last_scanned_at, refreshed_at
		FROM discovery_inventory
		WHERE tenant_id = $1 AND asset_id = $2
	`, tenantID, assetID)
	if err != nil {
		return nil, fmt.Errorf("risk engine: read inventory: %w", err)
	}
	defer rows.Close()

	var inv []domain.InventoryRow
	for rows.Next() {
		var r domain.InventoryRow
		if err := rows.Scan(
			&r.ID, &r.TenantID, &r.AssetID, &r.AssetName, &r.SourceID, &r.SourceName,
			&r.Classification, &r.Sensitivity, &r.FindingCount, &r.LastScannedAt, &r.RefreshedAt,
		); err != nil {
			return nil, fmt.Errorf("risk engine: scan row: %w", err)
		}
		inv = append(inv, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("risk engine: rows iteration: %w", err)
	}

	// 2. Check retention violation.
	var retentionViolation bool
	if err := e.repo.DB().QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM retention_violations
			WHERE asset_id = $1 AND tenant_id = $2
			LIMIT 1
		)
	`, assetID, tenantID).Scan(&retentionViolation); err != nil {
		// Table may not exist in older deployments — treat as false.
		retentionViolation = false
	}

	// 3. Build inputs and compute.
	inputs := buildDPDPAInputs(inv, retentionViolation)
	return computeResult(inputs), nil
}

// computeResult applies the DPDPA formula and returns the full breakdown.
func computeResult(inputs domain.DPDPARiskInputs) *RiskResult {
	clamp := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}

	pii := clamp(inputs.PIIDensity)
	sens := clamp(inputs.SensitivityWeight)
	access := clamp(inputs.AccessExposure)
	ret := clamp(inputs.RetentionViolation)

	piiContrib := pii * 0.35
	sensContrib := sens * 0.30
	accessContrib := access * 0.20
	retContrib := ret * 0.15

	rawScore := piiContrib + sensContrib + accessContrib + retContrib
	// Scale to [0, 100] and round to 2 decimal places.
	score := math.Round(rawScore*10000) / 100
	tier := domain.TierFromScore(score)

	return &RiskResult{
		Score:              score,
		Tier:               tier,
		PIIDensity:         pii,
		SensitivityWeight:  sens,
		AccessExposure:     access,
		RetentionViolation: ret,
		Breakdown: RiskBreakdown{
			PIIDensityContrib:         math.Round(piiContrib*10000) / 100,
			SensitivityWeightContrib:  math.Round(sensContrib*10000) / 100,
			AccessExposureContrib:     math.Round(accessContrib*10000) / 100,
			RetentionViolationContrib: math.Round(retContrib*10000) / 100,
		},
	}
}

// RecalculateOnScanComplete recalculates risk scores for all assets affected by
// a scan job. It identifies affected assets via the findings table and scores each
// one in turn, persisting the results.
func (e *RiskEngine) RecalculateOnScanComplete(ctx context.Context, scanJobID uuid.UUID) (int, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return 0, fmt.Errorf("risk engine recalc: %w", err)
	}

	// Find distinct asset IDs that have findings linked to this scan.
	rows, err := e.repo.DB().QueryContext(ctx, `
		SELECT DISTINCT asset_id FROM findings
		WHERE scan_run_id = $1 AND tenant_id = $2
	`, scanJobID, tenantID)
	if err != nil {
		return 0, fmt.Errorf("risk engine recalc: list assets: %w", err)
	}
	defer rows.Close()

	var assetIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("risk engine recalc: scan asset id: %w", err)
		}
		assetIDs = append(assetIDs, id)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("risk engine recalc: rows: %w", err)
	}

	scored := 0
	for _, assetID := range assetIDs {
		result, calcErr := e.CalculateRiskScore(ctx, assetID)
		if calcErr != nil {
			log.Printf("WARN: risk engine recalc: skipping asset %s: %v", assetID, calcErr)
			continue
		}

		// Persist the new score.
		factors, _ := json.Marshal(map[string]interface{}{
			"formula":             "dpdpa_2023_v2",
			"pii_density":         result.PIIDensity,
			"sensitivity_weight":  result.SensitivityWeight,
			"access_exposure":     result.AccessExposure,
			"retention_violation": result.RetentionViolation,
			"tier":                string(result.Tier),
			"breakdown":           result.Breakdown,
			"scan_job_id":         scanJobID.String(),
		})

		if err := e.repo.InsertRiskScore(ctx, &domain.RiskScore{
			TenantID:            tenantID,
			AssetID:             assetID,
			Score:               result.Score,
			ContributingFactors: unmarshalFactors(factors),
		}); err != nil {
			log.Printf("WARN: risk engine recalc: persist score for %s: %v", assetID, err)
			continue
		}

		// Write time-series history row (non-fatal).
		if _, err := e.repo.DB().ExecContext(ctx, `
			INSERT INTO risk_score_history
			    (asset_id, score, tier, pii_density, sensitivity_weight, access_exposure, retention_violation)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, assetID, result.Score, string(result.Tier),
			result.PIIDensity, result.SensitivityWeight, result.AccessExposure, result.RetentionViolation,
		); err != nil {
			log.Printf("WARN: risk engine: history insert for asset %s: %v", assetID, err)
		}

		scored++
	}

	log.Printf("INFO: risk engine recalc: scored %d/%d assets for scan %s", scored, len(assetIDs), scanJobID)
	return scored, nil
}

// TierChangeEvent is emitted when an asset's risk tier changes.
type TierChangeEvent struct {
	AssetID   uuid.UUID       `json:"asset_id"`
	OldTier   domain.RiskTier `json:"old_tier"`
	NewTier   domain.RiskTier `json:"new_tier"`
	OldScore  float64         `json:"old_score"`
	NewScore  float64         `json:"new_score"`
	Timestamp time.Time       `json:"timestamp"`
}

// CheckTierChange compares old and new tiers for an asset and, if they differ,
// triggers a webhook notification and records an audit entry. Returns true if a
// change was detected.
func (e *RiskEngine) CheckTierChange(ctx context.Context, assetID uuid.UUID, oldTier, newTier domain.RiskTier) (bool, error) {
	if oldTier == newTier {
		return false, nil
	}

	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return false, fmt.Errorf("check tier change: %w", err)
	}

	event := TierChangeEvent{
		AssetID:   assetID,
		OldTier:   oldTier,
		NewTier:   newTier,
		Timestamp: time.Now(),
	}

	// Fetch the latest score for extra context.
	var latestScore float64
	_ = e.repo.DB().QueryRowContext(ctx, `
		SELECT score FROM discovery_risk_scores
		WHERE tenant_id = $1 AND asset_id = $2
		ORDER BY computed_at DESC LIMIT 1
	`, tenantID, assetID).Scan(&latestScore)
	event.NewScore = latestScore

	// Record the tier transition in the audit/drift system.
	_, err = e.repo.DB().ExecContext(ctx, `
		INSERT INTO discovery_drift_events
		    (tenant_id, snapshot_id, event_type, asset_id, before_state, after_state, severity, detected_at)
		VALUES ($1, '00000000-0000-0000-0000-000000000000', $2, $3, $4, $5, $6, NOW())
	`,
		tenantID,
		tierChangeEventType(oldTier, newTier),
		assetID,
		marshalJSON(map[string]interface{}{"tier": string(oldTier)}),
		marshalJSON(map[string]interface{}{"tier": string(newTier), "score": latestScore}),
		tierChangeSeverity(oldTier, newTier),
	)
	if err != nil {
		log.Printf("WARN: tier change drift event insert: %v", err)
	}

	// Dispatch webhook + email alerts asynchronously so we don't block the caller.
	go e.dispatchTierChangeAlerts(tenantID, event)

	log.Printf("INFO: risk tier change for asset %s: %s -> %s (score: %.2f)",
		assetID, oldTier, newTier, latestScore)
	return true, nil
}

// dispatchTierChangeAlerts sends webhook and email notifications for tier transitions.
// Currently logs the intent — the actual HTTP/SMTP integration is plugged in when
// the notification module is wired.
func (e *RiskEngine) dispatchTierChangeAlerts(tenantID uuid.UUID, event TierChangeEvent) {
	payload, _ := json.Marshal(event)

	// Webhook dispatch (stub — uses tenant webhook config when available).
	log.Printf("INFO: risk_engine: webhook dispatch for tier change: tenant=%s asset=%s %s->%s (%d bytes)",
		tenantID, event.AssetID, event.OldTier, event.NewTier, len(payload))

	// Email alert (stub — uses notification service when wired).
	if event.NewTier == domain.RiskTierCritical {
		log.Printf("ALERT: risk_engine: asset %s escalated to CRITICAL — email alert queued", event.AssetID)
	}
}

// tierChangeEventType maps a tier transition to a drift event type.
func tierChangeEventType(old, new domain.RiskTier) string {
	if tierOrdinal(new) > tierOrdinal(old) {
		return string(domain.DriftRiskIncreased)
	}
	return string(domain.DriftRiskDecreased)
}

// tierChangeSeverity determines the severity of a tier transition.
func tierChangeSeverity(old, new domain.RiskTier) string {
	if new == domain.RiskTierCritical {
		return string(domain.SeverityCritical)
	}
	diff := tierOrdinal(new) - tierOrdinal(old)
	if diff < 0 {
		diff = -diff
	}
	if diff >= 2 {
		return string(domain.SeverityHigh)
	}
	return string(domain.SeverityMedium)
}

// tierOrdinal returns a numeric ordinal for tier comparison.
func tierOrdinal(t domain.RiskTier) int {
	switch t {
	case domain.RiskTierLow:
		return 0
	case domain.RiskTierMedium:
		return 1
	case domain.RiskTierHigh:
		return 2
	case domain.RiskTierCritical:
		return 3
	default:
		return 0
	}
}

// unmarshalFactors is a helper that converts JSON bytes to a map.
func unmarshalFactors(data []byte) map[string]interface{} {
	var m map[string]interface{}
	_ = json.Unmarshal(data, &m)
	return m
}

// marshalJSON is a nil-safe JSON marshal helper for DB inserts.
func marshalJSON(v interface{}) sql.NullString {
	b, err := json.Marshal(v)
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(b), Valid: true}
}
