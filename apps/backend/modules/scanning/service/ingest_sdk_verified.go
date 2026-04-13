package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/utils"
	"github.com/arc-platform/backend/pkg/validators"
	"github.com/google/uuid"
)

// VerifiedScanInput represents batch of SDK-validated findings
type VerifiedScanInput struct {
	ScanID   string                 `json:"scan_id"`
	Findings []VerifiedFinding      `json:"findings"`
	Metadata map[string]interface{} `json:"metadata"`
}

// IngestSDKVerified processes SDK-validated findings
// This is the simplified Phase 2 ingestion that trusts SDK validation
func (s *IngestionService) IngestSDKVerified(ctx context.Context, input VerifiedScanInput) error {
	adapter := NewSDKAdapter()

	// Start transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Use scan_id from request to link back to the orchestrated scan run
	scanRunID, err := uuid.Parse(input.ScanID)
	if err != nil {
		scanRunID = uuid.New()
	}

	// Look up existing ScanRun first (orchestrator creates it before the scanner runs)
	// Do NOT set status to "completed" here — the scanner calls /complete after all chunks,
	// which properly sets both status and scan_completed_at timestamp.
	var scanRun *entity.ScanRun
	if existing, lookupErr := s.repo.GetScanRunByID(ctx, scanRunID); lookupErr == nil {
		scanRun = existing
	} else {
		// ScanRun not found — create a new one (keep as "running")
		scanRun = &entity.ScanRun{
			ID:     scanRunID,
			Status: "running",
			Metadata: map[string]interface{}{
				"sdk_scan":    true,
				"sdk_version": "2.0",
			},
		}
		if err := tx.CreateScanRun(ctx, scanRun); err != nil {
			return fmt.Errorf("failed to create scan run: %w", err)
		}
	}

	// Track assets and stats
	assetMap := make(map[uuid.UUID]bool)
	acceptedFindingsCount := 0

	// Process each finding
	for _, vf := range input.Findings {
		// CRITICAL: Validate PII type against locked scope (LAW 3)
		if !IsLockedPIIType(vf.PIIType) {
			continue
		}

		// Format validation — reject false positives (defense-in-depth)
		matchValue := vf.ContextExcerpt
		if matchValue != "" && !validators.Validate(vf.PIIType, matchValue) {
			continue
		}

		acceptedFindingsCount++

		assetID, err := s.processSingleSDKFinding(ctx, tx, adapter, scanRun.ID, &vf)
		if err != nil {
			log.Printf("error processing finding (pii_type=%s): %v", vf.PIIType, err)
			continue
		}

		assetMap[assetID] = true
	}

	// Update ScanRun total counts — accumulate across chunks, don't overwrite
	scanRun.TotalFindings += acceptedFindingsCount
	scanRun.TotalAssets += len(assetMap)

	if err := tx.UpdateScanRun(ctx, scanRun); err != nil {
		return fmt.Errorf("failed to update scan run with final stats: %w", err)
	}

	// Commit transaction first so findings are visible to the count queries below
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update asset stats (TotalFindings, RiskScore) — must run AFTER commit so
	// CountFindings sees the newly inserted rows.
	for assetID := range assetMap {
		if err := s.recalculateAssetRisk(ctx, assetID); err != nil {
			log.Printf("failed to recalculate risk for asset %s: %v", assetID, err)
		}
	}

	return nil
}

func (s *IngestionService) processSingleSDKFinding(
	ctx context.Context,
	tx *persistence.PostgresTransaction,
	adapter *SDKAdapter,
	scanRunID uuid.UUID,
	vf *VerifiedFinding,
) (uuid.UUID, error) {
	// 1. Get or create asset using AssetManager
	asset := adapter.MapToAsset(vf)

	// Delegate to AssetManager (single source of truth)
	assetID, _, err := s.assetManager.CreateOrUpdateAsset(ctx, asset)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create/update asset: %w", err)
	}
	asset.ID = assetID

	// 2. Create finding — enforce PII_STORE_MODE (C-3)
	finding := adapter.MapToFinding(vf, scanRunID, asset.ID)
	finding.TenantID = persistence.DevSystemTenantID
	applyPIIStoreMode(finding)
	if err := tx.CreateFinding(ctx, finding); err != nil {
		return assetID, fmt.Errorf("failed to create finding: %w", err)
	}

	// 3. Create classification
	classification := adapter.MapToClassification(vf, finding.ID)
	if err := tx.CreateClassification(ctx, classification); err != nil {
		return assetID, fmt.Errorf("failed to create classification: %w", err)
	}

	// Note: Lineage sync is now handled automatically by AssetService

	return assetID, nil
}

// applyPIIStoreMode enforces PII_STORE_MODE on findings before persistence (C-3).
// Modes: "full" (default, store raw), "mask" (store masked), "none" (store hash only).
func applyPIIStoreMode(finding *entity.Finding) {
	mode := strings.ToLower(os.Getenv("PII_STORE_MODE"))
	if mode == "" || mode == "full" {
		return // store raw values (current behavior)
	}

	masker := utils.NewPIIMasker()

	switch mode {
	case "mask":
		for i, match := range finding.Matches {
			finding.Matches[i] = masker.MaskCreditCard(match) // generic mask
		}
		if finding.SampleText != "" {
			finding.SampleText = "[MASKED]"
		}
	case "none":
		for i, match := range finding.Matches {
			finding.Matches[i] = masker.HashValue(match)
		}
		finding.SampleText = ""
	}
}
