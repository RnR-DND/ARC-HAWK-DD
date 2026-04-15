package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/domain/repository"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ============================================================================
// FindingRepository Implementation
// ============================================================================

func (r *PostgresRepository) CreateFinding(ctx context.Context, finding *entity.Finding) error {
	contextJSON, err := json.Marshal(finding.Context)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	// Enforce Tenant ID
	tenantID, err := EnsureTenantID(ctx)
	if err != nil {
		return err
	}
	finding.TenantID = tenantID

	query := `
		INSERT INTO findings (id, tenant_id, scan_run_id, asset_id, pattern_id, pattern_name,
			matches, sample_text, normalized_value_hash, severity, severity_description, confidence_score, environment, context)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		finding.ID, finding.TenantID, finding.ScanRunID, finding.AssetID, finding.PatternID, finding.PatternName,
		pq.Array(finding.Matches), finding.SampleText, finding.NormalizedValueHash, finding.Severity, finding.SeverityDescription,
		finding.ConfidenceScore, finding.Environment, contextJSON,
	).Scan(&finding.CreatedAt, &finding.UpdatedAt)
}

func (r *PostgresRepository) GetFindingByID(ctx context.Context, id uuid.UUID) (*entity.Finding, error) {
	tenantID, err := EnsureTenantID(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, tenant_id, scan_run_id, asset_id, pattern_id, pattern_name, matches, sample_text, 
			severity, severity_description, confidence_score, environment, context, created_at, updated_at
		FROM findings WHERE id = $1 AND tenant_id = $2`

	finding := &entity.Finding{}
	var contextJSON []byte

	err = r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&finding.ID, &finding.TenantID, &finding.ScanRunID, &finding.AssetID, &finding.PatternID, &finding.PatternName,
		pq.Array(&finding.Matches), &finding.SampleText, &finding.Severity, &finding.SeverityDescription,
		&finding.ConfidenceScore, &finding.Environment, &contextJSON, &finding.CreatedAt, &finding.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("finding not found")
		}
		return nil, err
	}

	if len(contextJSON) > 0 {
		if err := json.Unmarshal(contextJSON, &finding.Context); err != nil {
			return nil, fmt.Errorf("failed to unmarshal context: %w", err)
		}
	}

	return finding, nil
}

func (r *PostgresRepository) ListFindingsByScanRun(ctx context.Context, scanRunID uuid.UUID, limit, offset int) ([]*entity.Finding, error) {
	tenantID, err := EnsureTenantID(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT f.id, f.tenant_id, f.scan_run_id, f.asset_id, f.pattern_id, f.pattern_name, f.matches, f.sample_text, 
			f.severity, f.severity_description, f.confidence_score, f.environment, f.context, f.created_at, f.updated_at
		FROM findings f
		LEFT JOIN classifications c ON f.id = c.finding_id
		WHERE f.scan_run_id = $1 AND f.tenant_id = $2 AND (c.classification_type IS NULL OR c.classification_type != 'Non-PII')
		ORDER BY f.created_at DESC
		LIMIT $3 OFFSET $4`

	return r.scanFindings(ctx, query, scanRunID, tenantID, limit, offset)
}

func (r *PostgresRepository) ListFindingsByAsset(ctx context.Context, assetID uuid.UUID, limit, offset int) ([]*entity.Finding, error) {
	tenantID, err := EnsureTenantID(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT f.id, f.tenant_id, f.scan_run_id, f.asset_id, f.pattern_id, f.pattern_name, f.matches, f.sample_text, 
			f.severity, f.severity_description, f.confidence_score, f.environment, f.context, f.created_at, f.updated_at
		FROM findings f
		LEFT JOIN classifications c ON f.id = c.finding_id
		WHERE f.asset_id = $1 AND f.tenant_id = $2 AND (c.classification_type IS NULL OR c.classification_type != 'Non-PII')
		ORDER BY f.created_at DESC
		LIMIT $3 OFFSET $4`

	return r.scanFindings(ctx, query, assetID, tenantID, limit, offset)
}

func (r *PostgresRepository) ListFindings(ctx context.Context, filters repository.FindingFilters, limit, offset int) ([]*entity.Finding, error) {
	tenantID, err := EnsureTenantID(ctx)
	if err != nil {
		return nil, err
	}

	// Allowlist for sort columns to prevent SQL injection
	validSortCols := map[string]string{
		"created_at":   "f.created_at",
		"severity":     "f.severity",
		"pattern_name": "f.pattern_name",
		"asset_name":   "a.name",
		"confidence":   "f.confidence_score",
	}
	sortCol, ok := validSortCols[strings.ToLower(filters.SortBy)]
	if !ok {
		sortCol = "f.created_at"
	}
	sortDir := "DESC"
	if strings.EqualFold(filters.SortOrder, "asc") {
		sortDir = "ASC"
	}

	// AUTO-EXCLUDE Non-PII: Join with classifications to filter out false positives
	// Join assets for search and asset-name filter support
	// Join review_states for status filter
	query := `
		SELECT DISTINCT f.id, f.tenant_id, f.scan_run_id, f.asset_id, f.pattern_id, f.pattern_name, f.matches, f.sample_text,
			f.severity, f.severity_description, f.confidence_score, f.environment, f.context, f.created_at, f.updated_at
		FROM findings f
		LEFT JOIN classifications c ON f.id = c.finding_id
		LEFT JOIN assets a ON f.asset_id = a.id
		LEFT JOIN review_states rs ON rs.finding_id = f.id
		WHERE f.tenant_id = $1 AND (c.classification_type IS NULL OR c.classification_type != 'Non-PII')`

	args := []interface{}{tenantID}
	argCount := 2

	if filters.ScanRunID != nil {
		query += fmt.Sprintf(" AND f.scan_run_id = $%d", argCount)
		args = append(args, *filters.ScanRunID)
		argCount++
	}

	if filters.AssetID != nil {
		query += fmt.Sprintf(" AND f.asset_id = $%d", argCount)
		args = append(args, *filters.AssetID)
		argCount++
	}

	if filters.AssetName != "" {
		query += fmt.Sprintf(" AND a.name ILIKE $%d", argCount)
		args = append(args, "%"+filters.AssetName+"%")
		argCount++
	}

	if filters.Severity != "" {
		query += fmt.Sprintf(" AND f.severity = ANY(string_to_array($%d, ','))", argCount)
		args = append(args, filters.Severity)
		argCount++
	}

	if filters.PatternName != "" {
		query += fmt.Sprintf(" AND f.pattern_name ILIKE $%d", argCount)
		args = append(args, "%"+filters.PatternName+"%")
		argCount++
	}

	if filters.DataSource != "" {
		query += fmt.Sprintf(" AND f.data_source = $%d", argCount)
		args = append(args, filters.DataSource)
		argCount++
	}

	if filters.Search != "" {
		query += fmt.Sprintf(" AND (a.name ILIKE $%d OR a.path ILIKE $%d OR f.pattern_name ILIKE $%d)", argCount, argCount, argCount)
		args = append(args, "%"+filters.Search+"%")
		argCount++
	}

	// Status filter maps to review_states.status
	switch filters.ReviewStatus {
	case "Active":
		query += " AND (rs.status IS NULL OR rs.status = 'pending')"
	case "Suppressed":
		query += " AND rs.status = 'false_positive'"
	case "Remediated":
		query += " AND rs.status = 'confirmed'"
	}

	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d", sortCol, sortDir, argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanFindingsFromRows(rows)
}

// ListGlobalFindings retrieves findings across all tenants (for system dashboard)
// WARNING: Bypasses tenant isolation. Callers MUST verify system-admin role before invoking.
func (r *PostgresRepository) ListGlobalFindings(ctx context.Context, limit, offset int) ([]*entity.Finding, error) {
	query := `
		SELECT DISTINCT f.id, f.tenant_id, f.scan_run_id, f.asset_id, f.pattern_id, f.pattern_name, f.matches, f.sample_text, 
			f.severity, f.severity_description, f.confidence_score, f.environment, f.context, f.created_at, f.updated_at
		FROM findings f
		LEFT JOIN classifications c ON f.id = c.finding_id
		WHERE (c.classification_type IS NULL OR c.classification_type != 'Non-PII')
		ORDER BY f.created_at DESC
		LIMIT $1 OFFSET $2`

	return r.scanFindings(ctx, query, limit, offset)
}

func (r *PostgresRepository) CountFindings(ctx context.Context, filters repository.FindingFilters) (int, error) {
	tenantID, err := EnsureTenantID(ctx)
	if err != nil {
		return 0, err
	}

	// AUTO-EXCLUDE Non-PII: Join with classifications to filter out false positives
	// Join assets for asset-name filter, join review_states for status filter
	query := `
		SELECT COUNT(DISTINCT f.id)
		FROM findings f
		LEFT JOIN classifications c ON f.id = c.finding_id
		LEFT JOIN assets a ON f.asset_id = a.id
		LEFT JOIN review_states rs ON rs.finding_id = f.id
		WHERE f.tenant_id = $1 AND (c.classification_type IS NULL OR c.classification_type != 'Non-PII')`

	args := []interface{}{tenantID}
	argCount := 2

	if filters.ScanRunID != nil {
		query += fmt.Sprintf(" AND f.scan_run_id = $%d", argCount)
		args = append(args, *filters.ScanRunID)
		argCount++
	}

	if filters.AssetID != nil {
		query += fmt.Sprintf(" AND f.asset_id = $%d", argCount)
		args = append(args, *filters.AssetID)
		argCount++
	}

	if filters.AssetName != "" {
		query += fmt.Sprintf(" AND a.name ILIKE $%d", argCount)
		args = append(args, "%"+filters.AssetName+"%")
		argCount++
	}

	if filters.Severity != "" {
		query += fmt.Sprintf(" AND f.severity = ANY(string_to_array($%d, ','))", argCount)
		args = append(args, filters.Severity)
		argCount++
	}

	if filters.PatternName != "" {
		query += fmt.Sprintf(" AND f.pattern_name ILIKE $%d", argCount)
		args = append(args, "%"+filters.PatternName+"%")
		argCount++
	}

	if filters.DataSource != "" {
		query += fmt.Sprintf(" AND f.data_source = $%d", argCount)
		args = append(args, filters.DataSource)
		argCount++
	}

	if filters.Search != "" {
		query += fmt.Sprintf(" AND (a.name ILIKE $%d OR a.path ILIKE $%d OR f.pattern_name ILIKE $%d)", argCount, argCount, argCount)
		args = append(args, "%"+filters.Search+"%")
		argCount++
	}

	// Status filter maps to review_states.status
	switch filters.ReviewStatus {
	case "Active":
		query += " AND (rs.status IS NULL OR rs.status = 'pending')"
	case "Suppressed":
		query += " AND rs.status = 'false_positive'"
	case "Remediated":
		query += " AND rs.status = 'confirmed'"
	}

	var count int
	err = r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func (r *PostgresRepository) scanFindings(ctx context.Context, query string, args ...interface{}) ([]*entity.Finding, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanFindingsFromRows(rows)
}

func (r *PostgresRepository) scanFindingsFromRows(rows *sql.Rows) ([]*entity.Finding, error) {
	var findings []*entity.Finding
	for rows.Next() {
		finding := &entity.Finding{}
		var contextJSON []byte

		err := rows.Scan(
			&finding.ID, &finding.TenantID, &finding.ScanRunID, &finding.AssetID, &finding.PatternID, &finding.PatternName,
			pq.Array(&finding.Matches), &finding.SampleText, &finding.Severity, &finding.SeverityDescription,
			&finding.ConfidenceScore, &finding.Environment, &contextJSON, &finding.CreatedAt, &finding.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(contextJSON) > 0 {
			if err := json.Unmarshal(contextJSON, &finding.Context); err != nil {
				return nil, fmt.Errorf("failed to unmarshal context: %w", err)
			}
		}

		findings = append(findings, finding)
	}

	return findings, rows.Err()
}

func (r *PostgresRepository) CreateFeedback(ctx context.Context, feedback *entity.FindingFeedback) error {
	query := `
		INSERT INTO finding_feedback (id, finding_id, user_id, feedback_type, original_classification, proposed_classification, comments)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, processed`

	return r.db.QueryRowContext(ctx, query,
		feedback.ID, feedback.FindingID, feedback.UserID, feedback.FeedbackType,
		feedback.OriginalClassification, feedback.ProposedClassification, feedback.Comments,
	).Scan(&feedback.CreatedAt, &feedback.Processed)
}

func (r *PostgresRepository) GetFeedbackForDataset(ctx context.Context) ([]entity.FeedbackWithFinding, error) {
	query := `
		SELECT 
			fb.id, fb.finding_id, fb.user_id, fb.feedback_type, fb.original_classification, fb.proposed_classification, fb.comments, fb.created_at, fb.processed,
			f.id, f.scan_run_id, f.asset_id, f.pattern_id, f.pattern_name, f.matches, f.sample_text, f.severity, f.severity_description, f.confidence_score, f.context, f.created_at, f.updated_at
		FROM finding_feedback fb
		JOIN findings f ON fb.finding_id = f.id
		WHERE fb.feedback_type IN ('CONFIRMED', 'FALSE_POSITIVE')
		LIMIT 1000
		ORDER BY fb.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query feedback: %w", err)
	}
	defer rows.Close()

	var results []entity.FeedbackWithFinding

	for rows.Next() {
		var item entity.FeedbackWithFinding
		var contextJSON []byte

		err := rows.Scan(
			&item.Feedback.ID, &item.Feedback.FindingID, &item.Feedback.UserID, &item.Feedback.FeedbackType, &item.Feedback.OriginalClassification, &item.Feedback.ProposedClassification, &item.Feedback.Comments, &item.Feedback.CreatedAt, &item.Feedback.Processed,
			&item.Finding.ID, &item.Finding.ScanRunID, &item.Finding.AssetID, &item.Finding.PatternID, &item.Finding.PatternName, pq.Array(&item.Finding.Matches), &item.Finding.SampleText, &item.Finding.Severity, &item.Finding.SeverityDescription, &item.Finding.ConfidenceScore, &contextJSON, &item.Finding.CreatedAt, &item.Finding.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feedback row: %w", err)
		}

		if len(contextJSON) > 0 {
			if err := json.Unmarshal(contextJSON, &item.Finding.Context); err != nil {
				return nil, fmt.Errorf("failed to unmarshal context: %w", err)
			}
		}

		results = append(results, item)
	}

	return results, nil
}

// UpdateMaskedValues updates the masked_value field for multiple findings
func (r *PostgresRepository) UpdateMaskedValues(ctx context.Context, maskedData map[uuid.UUID]string) error {
	tenantID, err := EnsureTenantID(ctx)
	if err != nil {
		return err
	}

	if len(maskedData) == 0 {
		return nil
	}

	// Use a transaction for batch update
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `UPDATE findings SET masked_value = $1 WHERE id = $2 AND tenant_id = $3`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for findingID, maskedValue := range maskedData {
		if _, err := stmt.ExecContext(ctx, maskedValue, findingID, tenantID); err != nil {
			return fmt.Errorf("failed to update finding %s: %w", findingID, err)
		}
	}

	return tx.Commit()
}

// GetFindingsByAssetWithMasking retrieves findings for an asset, returning masked values if available
func (r *PostgresRepository) GetFindingsByAssetWithMasking(ctx context.Context, assetID uuid.UUID) ([]*entity.Finding, error) {
	tenantID, err := EnsureTenantID(ctx)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT f.id, f.tenant_id, f.scan_run_id, f.asset_id, f.pattern_id, f.pattern_name, 
			f.matches, f.masked_value, f.sample_text, f.severity, f.severity_description, 
			f.confidence_score, f.context, f.created_at, f.updated_at,
			a.is_masked
		FROM findings f
		JOIN assets a ON f.asset_id = a.id
		WHERE f.asset_id = $1 AND f.tenant_id = $2
		ORDER BY f.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, assetID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var findings []*entity.Finding
	for rows.Next() {
		finding := &entity.Finding{}
		var contextJSON []byte
		var isMasked bool

		err := rows.Scan(
			&finding.ID, &finding.TenantID, &finding.ScanRunID, &finding.AssetID, &finding.PatternID, &finding.PatternName,
			pq.Array(&finding.Matches), &finding.MaskedValue, &finding.SampleText, &finding.Severity, &finding.SeverityDescription,
			&finding.ConfidenceScore, &contextJSON, &finding.CreatedAt, &finding.UpdatedAt,
			&isMasked,
		)
		if err != nil {
			return nil, err
		}

		if len(contextJSON) > 0 {
			if err := json.Unmarshal(contextJSON, &finding.Context); err != nil {
				return nil, fmt.Errorf("failed to unmarshal context: %w", err)
			}
		}

		// If asset is masked and masked_value is set, replace matches with masked value
		if isMasked && finding.MaskedValue != "" {
			finding.Matches = []string{finding.MaskedValue}
		}

		findings = append(findings, finding)
	}

	return findings, rows.Err()
}
