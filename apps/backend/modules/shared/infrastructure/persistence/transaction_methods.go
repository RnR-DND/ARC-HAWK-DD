package persistence

import (
	"context"
	"encoding/json"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Transaction methods for PostgresTransaction
// These mirror the main repository methods but use t.tx instead of r.db

// CreateScanRun creates a new scan run within a transaction
func (t *PostgresTransaction) CreateScanRun(ctx context.Context, scanRun *entity.ScanRun) error {
	// Marshal metadata to JSON
	metadataJSON, err := json.Marshal(scanRun.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO scan_runs (
			id, profile_name, scan_started_at, scan_completed_at, host, status,
			total_findings, total_assets, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
	`

	_, err = t.tx.ExecContext(ctx, query,
		scanRun.ID,
		scanRun.ProfileName,
		scanRun.ScanStartedAt,
		scanRun.ScanCompletedAt,
		scanRun.Host,
		scanRun.Status,
		scanRun.TotalFindings,
		scanRun.TotalAssets,
		metadataJSON,
	)

	return err
}

// CreateAsset creates a new asset within a transaction
func (t *PostgresTransaction) CreateAsset(ctx context.Context, asset *entity.Asset) error {
	metadataJSON, err := json.Marshal(asset.FileMetadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO assets (id, stable_id, asset_type, name, path, data_source, host, 
			environment, owner, source_system, file_metadata, risk_score, total_findings)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at, updated_at`

	return t.tx.QueryRowContext(ctx, query,
		asset.ID, asset.StableID, asset.AssetType, asset.Name, asset.Path,
		asset.DataSource, asset.Host, asset.Environment, asset.Owner, asset.SourceSystem,
		metadataJSON, asset.RiskScore, asset.TotalFindings,
	).Scan(&asset.CreatedAt, &asset.UpdatedAt)
}

// GetAssetByStableID retrieves an asset by stable ID within a transaction
func (t *PostgresTransaction) GetAssetByStableID(ctx context.Context, stableID string) (*entity.Asset, error) {
	query := `
		SELECT id, stable_id, asset_type, name, path, data_source, host, 
		       environment, owner, source_system, file_metadata, risk_score, total_findings, created_at, updated_at
		FROM assets
		WHERE stable_id = $1
		LIMIT 1
	`

	var asset entity.Asset
	var metadataJSON []byte

	err := t.tx.QueryRowContext(ctx, query, stableID).Scan(
		&asset.ID,
		&asset.StableID,
		&asset.AssetType,
		&asset.Name,
		&asset.Path,
		&asset.DataSource,
		&asset.Host,
		&asset.Environment,
		&asset.Owner,
		&asset.SourceSystem,
		&metadataJSON,
		&asset.RiskScore,
		&asset.TotalFindings,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil // Not found, not an error
		}
		return nil, err
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &asset.FileMetadata)
	}

	return &asset, nil
}

// CreateFinding creates a new finding within a transaction
func (t *PostgresTransaction) CreateFinding(ctx context.Context, finding *entity.Finding) error {
	contextJSON, err := json.Marshal(finding.Context)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO findings (id, scan_run_id, asset_id, pattern_id, pattern_name, 
			matches, sample_text, severity, severity_description, confidence_score, context)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`

	return t.tx.QueryRowContext(ctx, query,
		finding.ID, finding.ScanRunID, finding.AssetID, finding.PatternID, finding.PatternName,
		pq.Array(finding.Matches), finding.SampleText, finding.Severity, finding.SeverityDescription,
		finding.ConfidenceScore, contextJSON,
	).Scan(&finding.CreatedAt, &finding.UpdatedAt)
}

// CreateClassification creates a new classification within a transaction
func (t *PostgresTransaction) CreateClassification(ctx context.Context, classification *entity.Classification) error {
	query := `
		INSERT INTO classifications (
			id, finding_id, classification_type, sub_category, confidence_score,
			justification, dpdpa_category, requires_consent,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`

	_, err := t.tx.ExecContext(ctx, query,
		classification.ID,
		classification.FindingID,
		classification.ClassificationType,
		classification.SubCategory,
		classification.ConfidenceScore,
		classification.Justification,
		classification.DPDPACategory,
		classification.RequiresConsent,
	)

	return err
}

// CreateReviewState creates a new review state within a transaction
func (t *PostgresTransaction) CreateReviewState(ctx context.Context, reviewState *entity.ReviewState) error {
	query := `
		INSERT INTO review_states (
			id, finding_id, status, reviewed_by, reviewed_at, comments, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`

	_, err := t.tx.ExecContext(ctx, query,
		reviewState.ID,
		reviewState.FindingID,
		reviewState.Status,
		reviewState.ReviewedBy,
		reviewState.ReviewedAt,
		reviewState.Comments,
	)

	return err
}

// CreateOrGetPattern creates or retrieves a pattern within a transaction
func (t *PostgresTransaction) CreateOrGetPattern(ctx context.Context, pattern *entity.Pattern) error {
	// Try to get existing pattern
	var existingID uuid.UUID
	query := `SELECT id FROM patterns WHERE name = $1 LIMIT 1`
	err := t.tx.QueryRowContext(ctx, query, pattern.Name).Scan(&existingID)

	if err == nil {
		// Pattern exists, use existing ID
		pattern.ID = existingID
		return nil
	}

	// Pattern doesn't exist, create it
	insertQuery := `
		INSERT INTO patterns (id, name, pattern_type, category, description, pattern_definition, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (name) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`

	return t.tx.QueryRowContext(ctx, insertQuery,
		pattern.ID,
		pattern.Name,
		pattern.PatternType,
		pattern.Category,
		pattern.Description,
		pattern.PatternDefinition,
		pattern.IsActive,
	).Scan(&pattern.ID)
}

// UpdateAssetStats updates asset statistics within a transaction
func (t *PostgresTransaction) UpdateAssetStats(ctx context.Context, assetID uuid.UUID, stats map[string]interface{}) error {
	query := `
		UPDATE assets
		SET total_findings = $1,
		    risk_score = $2,
		    updated_at = NOW()
		WHERE id = $3
	`

	_, err := t.tx.ExecContext(ctx, query,
		stats["total_findings"],
		stats["risk_score"],
		assetID,
	)

	return err
}

// UpdateScanRun updates scan run statistics within a transaction
func (t *PostgresTransaction) UpdateScanRun(ctx context.Context, scanRun *entity.ScanRun) error {
	// Marshal metadata to JSON
	metadataJSON, err := json.Marshal(scanRun.Metadata)
	if err != nil {
		return err
	}

	query := `
		UPDATE scan_runs
		SET total_findings = $1,
		    total_assets = $2,
		    metadata = $3,
		    status = $4,
		    updated_at = NOW()
		WHERE id = $5
	`

	_, err = t.tx.ExecContext(ctx, query,
		scanRun.TotalFindings,
		scanRun.TotalAssets,
		metadataJSON,
		scanRun.Status,
		scanRun.ID,
	)

	return err
}
