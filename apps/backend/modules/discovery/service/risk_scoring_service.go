package service

import (
	"context"
	"fmt"
	"log"
	"math"

	"github.com/arc-platform/backend/modules/discovery/domain"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// RiskScoringService computes a composite risk score per asset using a weighted formula:
//
//	score = sum_per_classification( finding_count * sensitivity_weight * exposure_weight )
//
// Weights are config-driven and surfaced in the board report metadata so the formula
// is transparent. Pure-function ComputeScore is heavily unit-tested.
type RiskScoringService struct {
	repo    *Repo
	weights domain.RiskWeights
}

// NewRiskScoringService creates a new risk scoring service with default weights.
// Use NewRiskScoringServiceWithWeights to override.
func NewRiskScoringService(repo *Repo) *RiskScoringService {
	return &RiskScoringService{repo: repo, weights: domain.DefaultRiskWeights()}
}

// NewRiskScoringServiceWithWeights creates a service with custom weights.
func NewRiskScoringServiceWithWeights(repo *Repo, weights domain.RiskWeights) *RiskScoringService {
	return &RiskScoringService{repo: repo, weights: weights}
}

// Weights returns the weights used by this service (for board report metadata).
func (s *RiskScoringService) Weights() domain.RiskWeights {
	return s.weights
}

// ComputeScore is a pure function: given a slice of inventory rows for one asset
// and the weight config, return a single composite score.
//
// Safety:
//   - Empty slice returns 0
//   - Negative inputs are clamped to 0
//   - Score is rounded to 2 decimals to keep DB and UI consistent
func ComputeScore(rows []domain.InventoryRow, weights domain.RiskWeights) float64 {
	if len(rows) == 0 {
		return 0
	}
	total := 0.0
	for _, r := range rows {
		findings := float64(r.FindingCount)
		if findings < 0 {
			findings = 0
		}
		sensitivity := float64(r.Sensitivity)
		if sensitivity < 0 {
			sensitivity = 0
		}
		// Exposure is currently a static 1.0 — sources don't yet expose internal/external classification.
		// When connections module exposes exposure metadata, plumb it here.
		exposure := 1.0

		total += findings*weights.Volume + sensitivity*weights.Sensitivity + findings*sensitivity*exposure*weights.Exposure
	}
	// Round to 2 decimals.
	return math.Round(total*100) / 100
}

// ComputeDPDPAScore applies the DPDPA 2023 risk formula to a DPDPARiskInputs struct.
// Returns a score in [0, 100].
//
//	risk_score = (pii_density × 0.35) + (sensitivity_weight × 0.30) +
//	             (access_exposure × 0.20) + (retention_violation × 0.15)
//
// All inputs are clamped to [0, 1] before the formula is applied.
func ComputeDPDPAScore(inputs domain.DPDPARiskInputs) float64 {
	clamp := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}
	score := clamp(inputs.PIIDensity)*0.35 +
		clamp(inputs.SensitivityWeight)*0.30 +
		clamp(inputs.AccessExposure)*0.20 +
		clamp(inputs.RetentionViolation)*0.15
	return math.Round(score*10000) / 100 // scale to [0,100], 2 decimal places
}

// buildDPDPAInputs derives DPDPARiskInputs from inventory rows plus a retention violation flag.
func buildDPDPAInputs(inv []domain.InventoryRow, retentionViolation bool) domain.DPDPARiskInputs {
	if len(inv) == 0 {
		return domain.DPDPARiskInputs{}
	}

	totalFields := len(inv)
	piiFields := 0
	totalSensitivityWeight := 0.0

	sensitivityWeightMap := map[int]float64{
		3: 1.0, // high
		2: 0.6, // medium
		1: 0.2, // low
	}

	for _, r := range inv {
		if r.FindingCount > 0 {
			piiFields++
			w, ok := sensitivityWeightMap[r.Sensitivity]
			if !ok {
				w = 0.2
			}
			totalSensitivityWeight += w
		}
	}

	piiDensity := float64(piiFields) / float64(totalFields)
	var sensitivityAvg float64
	if piiFields > 0 {
		sensitivityAvg = totalSensitivityWeight / float64(piiFields)
	}

	// access_exposure: defaulted to 0.5 (moderate) until connections module
	// exposes per-source RBAC metadata. This is surfaced transparently in
	// ContributingFactors so DPOs understand the assumption.
	accessExposure := 0.5

	retViolation := 0.0
	if retentionViolation {
		retViolation = 1.0
	}

	return domain.DPDPARiskInputs{
		PIIDensity:         piiDensity,
		SensitivityWeight:  sensitivityAvg,
		AccessExposure:     accessExposure,
		RetentionViolation: retViolation,
	}
}

// ScoreAsset computes and persists a DPDPA risk score for one asset in the tenant in ctx.
// Returns the persisted RiskScore. snapshotID is optional (nil = ad-hoc score not tied to a snapshot).
func (s *RiskScoringService) ScoreAsset(ctx context.Context, assetID uuid.UUID, snapshotID *uuid.UUID) (*domain.RiskScore, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("score asset: %w", err)
	}

	// Read all inventory rows for this asset.
	rows, err := s.repo.DB().QueryContext(ctx, `
		SELECT id, tenant_id, asset_id, asset_name, source_id, source_name,
		       classification, sensitivity, finding_count, last_scanned_at, refreshed_at
		FROM discovery_inventory
		WHERE tenant_id = $1 AND asset_id = $2
	`, tenantID, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inv []domain.InventoryRow
	for rows.Next() {
		var r domain.InventoryRow
		if err := rows.Scan(
			&r.ID, &r.TenantID, &r.AssetID, &r.AssetName, &r.SourceID, &r.SourceName,
			&r.Classification, &r.Sensitivity, &r.FindingCount, &r.LastScannedAt, &r.RefreshedAt,
		); err != nil {
			return nil, err
		}
		inv = append(inv, r)
	}

	// Check whether any finding for this asset is past its retention deadline.
	var retentionViolation bool
	if err := s.repo.DB().QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM retention_violations
			WHERE asset_id = $1 AND tenant_id = $2
			LIMIT 1
		)
	`, assetID, tenantID).Scan(&retentionViolation); err != nil {
		// retention_violations view may not exist in older deployments — treat as false.
		retentionViolation = false
	}

	inputs := buildDPDPAInputs(inv, retentionViolation)
	score := ComputeDPDPAScore(inputs)
	tier := domain.TierFromScore(score)

	rs := &domain.RiskScore{
		TenantID:   tenantID,
		AssetID:    assetID,
		SnapshotID: snapshotID,
		Score:      score,
		ContributingFactors: map[string]interface{}{
			"formula":              "dpdpa_2023",
			"pii_density":          inputs.PIIDensity,
			"sensitivity_weight":   inputs.SensitivityWeight,
			"access_exposure":      inputs.AccessExposure,
			"access_exposure_note": "defaulted to 0.5 pending RBAC metadata",
			"retention_violation":  inputs.RetentionViolation,
			"tier":                 string(tier),
			"row_count":            len(inv),
		},
	}
	if err := s.repo.InsertRiskScore(ctx, rs); err != nil {
		return nil, err
	}

	// Persist a risk_score_history row for time-series trend analysis (migration 000031).
	// Non-fatal: a failure here should not block the caller.
	_, histErr := s.repo.DB().ExecContext(ctx, `
		INSERT INTO risk_score_history
		    (asset_id, score, tier, pii_density, sensitivity_weight, access_exposure, retention_violation)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`,
		assetID,
		score,
		string(tier),
		inputs.PIIDensity,
		inputs.SensitivityWeight,
		inputs.AccessExposure,
		inputs.RetentionViolation,
	)
	if histErr != nil {
		// Log but don't surface — risk_score_history is supplementary.
		log.Printf("WARN: risk_score_history insert: %v", histErr)
	}

	return rs, nil
}

func maxSensitivity(rows []domain.InventoryRow) int {
	max := 0
	for _, r := range rows {
		if r.Sensitivity > max {
			max = r.Sensitivity
		}
	}
	return max
}

func totalFindings(rows []domain.InventoryRow) int {
	t := 0
	for _, r := range rows {
		t += r.FindingCount
	}
	return t
}
