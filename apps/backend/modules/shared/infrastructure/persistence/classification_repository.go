package persistence

import (
	"context"

	"fmt"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/google/uuid"
)

// ============================================================================
// ClassificationRepository Implementation
// ============================================================================

func (r *PostgresRepository) CreateClassification(ctx context.Context, classification *entity.Classification) error {
	query := `
		INSERT INTO classifications (id, finding_id, classification_type, sub_category, 
			confidence_score, justification, dpdpa_category, requires_consent, retention_period)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		classification.ID, classification.FindingID, classification.ClassificationType,
		classification.SubCategory, classification.ConfidenceScore, classification.Justification,
		classification.DPDPACategory, classification.RequiresConsent, classification.RetentionPeriod,
	).Scan(&classification.CreatedAt, &classification.UpdatedAt)
}

func (r *PostgresRepository) GetClassificationsByFindingID(ctx context.Context, findingID uuid.UUID) ([]*entity.Classification, error) {
	query := `
		SELECT id, finding_id, classification_type, sub_category, confidence_score, 
			justification, dpdpa_category, requires_consent, retention_period, 
			created_at, updated_at
		FROM classifications 
		WHERE finding_id = $1`

	rows, err := r.db.QueryContext(ctx, query, findingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var classifications []*entity.Classification
	for rows.Next() {
		c := &entity.Classification{}
		var retentionPeriod *string // Use pointer to handle NULL
		var confidenceScore float64

		err := rows.Scan(
			&c.ID, &c.FindingID, &c.ClassificationType, &c.SubCategory,
			&confidenceScore, &c.Justification, &c.DPDPACategory,
			&c.RequiresConsent, &retentionPeriod,
			&c.CreatedAt, &c.UpdatedAt,
		)
		c.ConfidenceScore = &confidenceScore
		if err != nil {
			return nil, err
		}

		// Handle NULL retention_period
		if retentionPeriod != nil {
			c.RetentionPeriod = *retentionPeriod
		} else {
			c.RetentionPeriod = "" // Default to empty string if NULL
		}

		classifications = append(classifications, c)
	}

	return classifications, rows.Err()
}

func (r *PostgresRepository) GetClassificationSummary(ctx context.Context) (map[string]interface{}, error) {
	// Query classification types (AUTO-EXCLUDE Non-PII for clean dashboard stats)
	query := `
		SELECT 
			classification_type, 
			COUNT(*) as count,
			AVG(confidence_score) as avg_confidence
		FROM classifications
		WHERE classification_type != 'Non-PII'
		GROUP BY classification_type`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := make(map[string]interface{})
	typeBreakdown := make(map[string]interface{})

	for rows.Next() {
		var classificationType string
		var count int
		var avgConfidence float64

		if err := rows.Scan(&classificationType, &count, &avgConfidence); err != nil {
			return nil, err
		}

		typeBreakdown[classificationType] = map[string]interface{}{
			"count":          count,
			"avg_confidence": avgConfidence,
		}
	}

	summary["by_type"] = typeBreakdown

	// Query severity breakdown (use filtered findings via JOIN)
	severityQuery := `
		SELECT 
			f.severity, 
			COUNT(DISTINCT f.id) as count
		FROM findings f
		LEFT JOIN classifications c ON f.id = c.finding_id
		WHERE (c.classification_type IS NULL OR c.classification_type != 'Non-PII')
		GROUP BY f.severity`

	severityRows, err := r.db.QueryContext(ctx, severityQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query severity stats: %w", err)
	}
	defer severityRows.Close()

	severityBreakdown := make(map[string]int)
	for severityRows.Next() {
		var severity string
		var count int
		if err := severityRows.Scan(&severity, &count); err != nil {
			return nil, err
		}
		severityBreakdown[severity] = count
	}
	summary["by_severity"] = severityBreakdown

	// Get total count (exclude Non-PII for accurate dashboard display)
	var total int
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM classifications WHERE classification_type != 'Non-PII'").Scan(&total)
	if err != nil {
		return nil, err
	}
	summary["total"] = total

	// Get verified/confirmed count from review_states
	var verifiedCount int
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM review_states WHERE status = 'confirmed'").Scan(&verifiedCount)
	if err != nil {
		return nil, err
	}
	summary["verified_count"] = verifiedCount

	// Get false positive count
	var falsePositiveCount int
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM review_states WHERE status = 'false_positive'").Scan(&falsePositiveCount)
	if err != nil {
		return nil, err
	}
	summary["false_positive_count"] = falsePositiveCount

	return summary, rows.Err()
}
