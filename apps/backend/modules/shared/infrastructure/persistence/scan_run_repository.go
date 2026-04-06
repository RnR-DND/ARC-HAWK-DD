package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/google/uuid"
)

// ============================================================================
// ScanRunRepository Implementation
// ============================================================================

func (r *PostgresRepository) CreateScanRun(ctx context.Context, scanRun *entity.ScanRun) error {
	metadataJSON, err := json.Marshal(scanRun.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO scan_runs (id, profile_name, scan_started_at, scan_completed_at, host, 
			total_findings, total_assets, status, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		scanRun.ID, scanRun.ProfileName, scanRun.ScanStartedAt, scanRun.ScanCompletedAt,
		scanRun.Host, scanRun.TotalFindings, scanRun.TotalAssets, scanRun.Status, metadataJSON,
	).Scan(&scanRun.CreatedAt, &scanRun.UpdatedAt)
}

func (r *PostgresRepository) GetScanRunByID(ctx context.Context, id uuid.UUID) (*entity.ScanRun, error) {
	query := `
		SELECT id, profile_name, scan_started_at, scan_completed_at, host, 
			total_findings, total_assets, status, metadata, created_at, updated_at
		FROM scan_runs WHERE id = $1`

	scanRun := &entity.ScanRun{}
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&scanRun.ID, &scanRun.ProfileName, &scanRun.ScanStartedAt, &scanRun.ScanCompletedAt,
		&scanRun.Host, &scanRun.TotalFindings, &scanRun.TotalAssets, &scanRun.Status,
		&metadataJSON, &scanRun.CreatedAt, &scanRun.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("scan run not found")
		}
		return nil, err
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &scanRun.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return scanRun, nil
}

func (r *PostgresRepository) ListScanRuns(ctx context.Context, limit, offset int) ([]*entity.ScanRun, error) {
	query := `
		SELECT id, profile_name, scan_started_at, scan_completed_at, host, 
			total_findings, total_assets, status, metadata, created_at, updated_at
		FROM scan_runs 
		ORDER BY scan_started_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scanRuns []*entity.ScanRun
	for rows.Next() {
		scanRun := &entity.ScanRun{}
		var metadataJSON []byte

		err := rows.Scan(
			&scanRun.ID, &scanRun.ProfileName, &scanRun.ScanStartedAt, &scanRun.ScanCompletedAt,
			&scanRun.Host, &scanRun.TotalFindings, &scanRun.TotalAssets, &scanRun.Status,
			&metadataJSON, &scanRun.CreatedAt, &scanRun.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &scanRun.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		scanRuns = append(scanRuns, scanRun)
	}

	return scanRuns, rows.Err()
}

func (r *PostgresRepository) UpdateScanRun(ctx context.Context, scanRun *entity.ScanRun) error {
	existing, err := r.GetScanRunByID(ctx, scanRun.ID)
	if err != nil {
		return fmt.Errorf("scan run not found: %w", err)
	}

	if existing.Status == "completed" || existing.Status == "failed" {
		return fmt.Errorf("scan run is immutable after completion (status: %s)", existing.Status)
	}

	metadataJSON, err := json.Marshal(scanRun.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE scan_runs 
		SET total_findings = $1, total_assets = $2, status = $3, metadata = $4, 
		    scan_started_at = $5, scan_completed_at = $6, updated_at = NOW()
		WHERE id = $7`

	_, err = r.db.ExecContext(ctx, query,
		scanRun.TotalFindings, scanRun.TotalAssets, scanRun.Status, metadataJSON,
		scanRun.ScanStartedAt, scanRun.ScanCompletedAt, scanRun.ID,
	)
	return err
}

func (r *PostgresRepository) DeleteScanRun(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete classifications for findings in this scan
	if _, err := tx.ExecContext(ctx, `DELETE FROM classifications WHERE finding_id IN (SELECT id FROM findings WHERE scan_run_id = $1)`, id); err != nil {
		return fmt.Errorf("failed to delete classifications: %w", err)
	}

	// Delete findings
	if _, err := tx.ExecContext(ctx, `DELETE FROM findings WHERE scan_run_id = $1`, id); err != nil {
		return fmt.Errorf("failed to delete findings: %w", err)
	}

	// Delete scan run
	result, err := tx.ExecContext(ctx, `DELETE FROM scan_runs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete scan run: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("scan run not found")
	}

	return tx.Commit()
}

// GetScanPIISummary returns PII type counts for a specific scan
func (r *PostgresRepository) GetScanPIISummary(ctx context.Context, scanID uuid.UUID) ([]map[string]interface{}, error) {
	query := `
		SELECT c.sub_category AS pii_type, COUNT(*) AS count, AVG(c.confidence_score) AS avg_confidence
		FROM classifications c
		JOIN findings f ON c.finding_id = f.id
		WHERE f.scan_run_id = $1
		GROUP BY c.sub_category
		ORDER BY count DESC`

	rows, err := r.db.QueryContext(ctx, query, scanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summary []map[string]interface{}
	for rows.Next() {
		var piiType string
		var count int
		var avgConf float64
		if err := rows.Scan(&piiType, &count, &avgConf); err != nil {
			return nil, err
		}
		summary = append(summary, map[string]interface{}{
			"type":           piiType,
			"count":          count,
			"avg_confidence": avgConf,
		})
	}

	return summary, rows.Err()
}

func (r *PostgresRepository) GetLatestScanRun(ctx context.Context) (*entity.ScanRun, error) {
	query := `
		SELECT id, profile_name, scan_started_at, scan_completed_at, host, 
			total_findings, total_assets, status, metadata, created_at, updated_at
		FROM scan_runs 
		ORDER BY scan_started_at DESC
		LIMIT 1`

	scanRun := &entity.ScanRun{}
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query).Scan(
		&scanRun.ID, &scanRun.ProfileName, &scanRun.ScanStartedAt, &scanRun.ScanCompletedAt,
		&scanRun.Host, &scanRun.TotalFindings, &scanRun.TotalAssets, &scanRun.Status,
		&metadataJSON, &scanRun.CreatedAt, &scanRun.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &scanRun.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return scanRun, nil
}
