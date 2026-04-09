package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// RetentionViolation represents an asset with retention policy violations
type RetentionViolation struct {
	FindingID       string    `json:"finding_id"`
	AssetID         string    `json:"asset_id"`
	AssetName       string    `json:"asset_name"`
	PIIType         string    `json:"pii_type"`
	FirstDetectedAt time.Time `json:"first_detected_at"`
	RetentionDays   int       `json:"retention_policy_days"`
	DeletionDueAt   time.Time `json:"deletion_due_at"`
	DaysOverdue     int       `json:"days_overdue"`
}

// RetentionPolicy represents a retention policy configuration
type RetentionPolicy struct {
	AssetID       string `json:"asset_id"`
	RetentionDays int    `json:"retention_days"`
	PolicyName    string `json:"policy_name"`
	PolicyBasis   string `json:"policy_basis"`
}

// RetentionService handles retention policy operations
type RetentionService struct {
	db *sql.DB
}

// NewRetentionService creates a new retention service
func NewRetentionService(db *sql.DB) *RetentionService {
	return &RetentionService{db: db}
}

// SetRetentionPolicy sets the retention policy for an asset (upserts per tenant).
func (s *RetentionService) SetRetentionPolicy(ctx context.Context, assetID string, policyDays int, policyName, policyBasis string) error {
	if policyDays <= 0 {
		policyDays = 365
	}
	query := `
		UPDATE assets
		SET retention_policy_days = $1,
		    retention_policy_name = $2,
		    retention_policy_basis = $3,
		    updated_at = NOW()
		WHERE id = $4::UUID
	`

	result, err := s.db.ExecContext(ctx, query, policyDays, policyName, policyBasis, assetID)
	if err != nil {
		return fmt.Errorf("failed to set retention policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("asset not found: %s", assetID)
	}

	return nil
}

// GetRetentionPolicy gets the retention policy for an asset
func (s *RetentionService) GetRetentionPolicy(ctx context.Context, assetID string) (*RetentionPolicy, error) {
	query := `
		SELECT id, retention_policy_days, retention_policy_name, retention_policy_basis
		FROM assets
		WHERE id = $1
	`

	var policy RetentionPolicy
	err := s.db.QueryRowContext(ctx, query, assetID).Scan(
		&policy.AssetID,
		&policy.RetentionDays,
		&policy.PolicyName,
		&policy.PolicyBasis,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("asset not found: %s", assetID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get retention policy: %w", err)
	}

	return &policy, nil
}

// GetRetentionViolations returns findings that exceed retention policy
func (s *RetentionService) GetRetentionViolations(ctx context.Context) ([]RetentionViolation, error) {
	query := `
		SELECT 
			finding_id,
			asset_id,
			asset_name,
			pii_type,
			first_detected_at,
			retention_policy_days,
			deletion_due_at,
			days_overdue
		FROM retention_violations
		ORDER BY days_overdue DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get retention violations: %w", err)
	}
	defer rows.Close()

	var violations []RetentionViolation
	for rows.Next() {
		var violation RetentionViolation
		err := rows.Scan(
			&violation.FindingID,
			&violation.AssetID,
			&violation.AssetName,
			&violation.PIIType,
			&violation.FirstDetectedAt,
			&violation.RetentionDays,
			&violation.DeletionDueAt,
			&violation.DaysOverdue,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan retention violation: %w", err)
		}
		violations = append(violations, violation)
	}

	return violations, nil
}

// CalculateDeletionDue calculates when a finding should be deleted
func (s *RetentionService) CalculateDeletionDue(ctx context.Context, findingID string) (time.Time, error) {
	query := `
		SELECT
			f.created_at,
			COALESCE(a.retention_policy_days, 365)
		FROM findings f
		JOIN assets a ON f.asset_id = a.id
		WHERE f.id = $1::UUID
	`

	var firstDetected time.Time
	var retentionDays int

	err := s.db.QueryRowContext(ctx, query, findingID).Scan(&firstDetected, &retentionDays)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to calculate deletion due: %w", err)
	}

	deletionDue := firstDetected.AddDate(0, 0, retentionDays)
	return deletionDue, nil
}

// GetRetentionTimeline gets the retention timeline for an asset
func (s *RetentionService) GetRetentionTimeline(ctx context.Context, assetID string) ([]RetentionTimelineEntry, error) {
	query := `
		SELECT
			f.id,
			f.pattern_name,
			f.created_at,
			COALESCE(a.retention_policy_days, 365),
			calculate_deletion_due(f.created_at, COALESCE(a.retention_policy_days, 365)) as deletion_due_at,
			CASE
				WHEN calculate_deletion_due(f.created_at, COALESCE(a.retention_policy_days, 365)) < NOW()
				THEN 'OVERDUE'
				WHEN calculate_deletion_due(f.created_at, COALESCE(a.retention_policy_days, 365)) < NOW() + INTERVAL '30 days'
				THEN 'EXPIRING_SOON'
				ELSE 'COMPLIANT'
			END as status
		FROM findings f
		JOIN assets a ON f.asset_id = a.id
		WHERE f.asset_id = $1::UUID
		ORDER BY f.created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, assetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get retention timeline: %w", err)
	}
	defer rows.Close()

	var timeline []RetentionTimelineEntry
	for rows.Next() {
		var entry RetentionTimelineEntry
		err := rows.Scan(
			&entry.FindingID,
			&entry.PIIType,
			&entry.FirstDetectedAt,
			&entry.RetentionDays,
			&entry.DeletionDueAt,
			&entry.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timeline entry: %w", err)
		}
		timeline = append(timeline, entry)
	}

	return timeline, nil
}

// RetentionTimelineEntry represents an entry in the retention timeline
type RetentionTimelineEntry struct {
	FindingID       string    `json:"finding_id"`
	PIIType         string    `json:"pii_type"`
	FirstDetectedAt time.Time `json:"first_detected_at"`
	RetentionDays   int       `json:"retention_days"`
	DeletionDueAt   time.Time `json:"deletion_due_at"`
	Status          string    `json:"status"` // COMPLIANT, EXPIRING_SOON, OVERDUE
}
