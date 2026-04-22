package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// FeedbackService records analyst corrections and applies Bayesian confidence adjustment.
type FeedbackService struct {
	db *sql.DB
}

func NewFeedbackService(db *sql.DB) *FeedbackService {
	return &FeedbackService{db: db}
}

// PatternPrecisionStat summarises feedback for one pattern within a tenant.
type PatternPrecisionStat struct {
	PatternCode     string  `json:"pattern_code"`
	Confirmed       int     `json:"confirmed"`
	FalsePositives  int     `json:"false_positives"`
	Precision       float64 `json:"precision"`
	ConfidenceDelta float64 `json:"confidence_delta"`
}

// RecordCorrection writes one analyst correction to feedback_corrections and
// asynchronously adjusts the pattern confidence threshold when enough data exists.
func (s *FeedbackService) RecordCorrection(ctx context.Context, tenantID, userID, findingID, correctionType string) error {
	// Fetch pattern_name from the finding — the only column we need for indexing.
	var patternCode string
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(pattern_name, '') FROM findings WHERE id = $1 AND tenant_id = $2`,
		findingID, tenantID,
	).Scan(&patternCode)
	if err != nil {
		return fmt.Errorf("finding lookup: %w", err)
	}

	// corrected_by is nullable; pass nil when no user UUID is available.
	var correctedBy interface{}
	if userID != "" && userID != "00000000-0000-0000-0000-000000000000" {
		correctedBy = userID
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO feedback_corrections
		    (tenant_id, finding_id, pattern_code, correction_type, corrected_by)
		VALUES ($1, $2, $3, $4, $5)
	`, tenantID, findingID, patternCode, correctionType, correctedBy)
	if err != nil {
		return fmt.Errorf("insert correction: %w", err)
	}

	slog.InfoContext(ctx, "feedback recorded",
		"finding_id", findingID,
		"pattern_code", patternCode,
		"correction_type", correctionType,
		"tenant_id", tenantID,
	)

	go s.maybeAdjustPatternThreshold(context.Background(), tenantID, patternCode)
	return nil
}

// GetPatternPrecisionStats returns per-pattern precision stats for a tenant.
func (s *FeedbackService) GetPatternPrecisionStats(ctx context.Context, tenantID string) ([]PatternPrecisionStat, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			pattern_code,
			COUNT(*) FILTER (WHERE correction_type = 'confirmed')        AS confirmed,
			COUNT(*) FILTER (WHERE correction_type = 'false_positive')   AS false_positives,
			CASE
				WHEN COUNT(*) FILTER (WHERE correction_type IN ('confirmed','false_positive')) = 0
				THEN 1.0
				ELSE COUNT(*) FILTER (WHERE correction_type = 'confirmed')::float /
				     COUNT(*) FILTER (WHERE correction_type IN ('confirmed','false_positive'))::float
			END AS precision
		FROM feedback_corrections
		WHERE tenant_id = $1
		GROUP BY pattern_code
		ORDER BY false_positives DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []PatternPrecisionStat
	for rows.Next() {
		var st PatternPrecisionStat
		if err := rows.Scan(&st.PatternCode, &st.Confirmed, &st.FalsePositives, &st.Precision); err != nil {
			continue
		}
		// Bayesian delta: precision < 0.7 → negative delta (reduce confidence)
		st.ConfidenceDelta = (st.Precision - 0.7) * 100
		stats = append(stats, st)
	}
	return stats, rows.Err()
}

// maybeAdjustPatternThreshold raises or lowers a pattern's confidence threshold
// for a specific tenant once ≥10 feedback corrections exist.
func (s *FeedbackService) maybeAdjustPatternThreshold(ctx context.Context, tenantID, patternCode string) {
	var totalFeedback int
	var precision float64
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE correction_type IN ('confirmed','false_positive')),
			CASE WHEN COUNT(*) FILTER (WHERE correction_type IN ('confirmed','false_positive')) = 0
			     THEN 1.0
			     ELSE COUNT(*) FILTER (WHERE correction_type = 'confirmed')::float /
			          COUNT(*) FILTER (WHERE correction_type IN ('confirmed','false_positive'))::float
			END
		FROM feedback_corrections
		WHERE tenant_id = $1 AND pattern_code = $2
	`, tenantID, patternCode).Scan(&totalFeedback, &precision)
	if err != nil || totalFeedback < 10 {
		return
	}

	newThreshold := int(precision * 100)
	if newThreshold < 30 {
		newThreshold = 30
	}
	if newThreshold > 90 {
		newThreshold = 90
	}

	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO pattern_confidence_overrides (tenant_id, pattern_code, min_confidence_score, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (tenant_id, pattern_code) DO UPDATE
		SET min_confidence_score = $3, updated_at = NOW()
	`, tenantID, patternCode, newThreshold)

	slog.InfoContext(ctx, "bayesian threshold adjusted",
		"pattern", patternCode,
		"precision", precision,
		"new_threshold", newThreshold,
		"feedback_count", totalFeedback,
	)
}
