package persistence

import (
	"context"

	"database/sql"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/google/uuid"
)

// ============================================================================
// ReviewStateRepository Implementation
// ============================================================================

func (r *PostgresRepository) CreateReviewState(ctx context.Context, reviewState *entity.ReviewState) error {
	query := `
		INSERT INTO review_states (id, finding_id, status, reviewed_by, reviewed_at, comments)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		reviewState.ID, reviewState.FindingID, reviewState.Status,
		reviewState.ReviewedBy, reviewState.ReviewedAt, reviewState.Comments,
	).Scan(&reviewState.CreatedAt, &reviewState.UpdatedAt)
}

func (r *PostgresRepository) GetReviewStateByFindingID(ctx context.Context, findingID uuid.UUID) (*entity.ReviewState, error) {
	query := `
		SELECT id, finding_id, status, reviewed_by, reviewed_at, comments, created_at, updated_at
		FROM review_states 
		WHERE finding_id = $1
		ORDER BY created_at DESC
		LIMIT 1`

	rs := &entity.ReviewState{}
	err := r.db.QueryRowContext(ctx, query, findingID).Scan(
		&rs.ID, &rs.FindingID, &rs.Status, &rs.ReviewedBy,
		&rs.ReviewedAt, &rs.Comments, &rs.CreatedAt, &rs.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return rs, nil
}

func (r *PostgresRepository) UpdateReviewState(ctx context.Context, reviewState *entity.ReviewState) error {
	query := `
		UPDATE review_states 
		SET status = $1, reviewed_by = $2, reviewed_at = $3, comments = $4
		WHERE id = $5`

	_, err := r.db.ExecContext(ctx, query,
		reviewState.Status, reviewState.ReviewedBy, reviewState.ReviewedAt,
		reviewState.Comments, reviewState.ID,
	)
	return err
}
