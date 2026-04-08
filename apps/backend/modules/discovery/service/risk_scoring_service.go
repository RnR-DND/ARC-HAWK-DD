package service

import (
	"context"
	"fmt"
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

// ScoreAsset computes and persists a risk score for one asset in the tenant in ctx.
// Returns the persisted RiskScore. snapshotID is optional (nil = ad-hoc score not tied to a snapshot).
func (s *RiskScoringService) ScoreAsset(ctx context.Context, assetID uuid.UUID, snapshotID *uuid.UUID) (*domain.RiskScore, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("score asset: %w", err)
	}

	// Read all classifications for this asset.
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

	score := ComputeScore(inv, s.weights)

	rs := &domain.RiskScore{
		TenantID:   tenantID,
		AssetID:    assetID,
		SnapshotID: snapshotID,
		Score:      score,
		ContributingFactors: map[string]interface{}{
			"weights":           s.weights,
			"row_count":         len(inv),
			"max_sensitivity":   maxSensitivity(inv),
			"total_findings":    totalFindings(inv),
		},
	}
	if err := s.repo.InsertRiskScore(ctx, rs); err != nil {
		return nil, err
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
