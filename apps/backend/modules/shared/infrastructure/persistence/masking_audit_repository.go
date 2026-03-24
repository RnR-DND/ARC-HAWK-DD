package persistence

import (
	"context"
	"database/sql"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/google/uuid"
)

// MaskingAuditRepository handles masking audit log persistence
type MaskingAuditRepository struct {
	db *sql.DB
}

// NewMaskingAuditRepository creates a new masking audit repository
func NewMaskingAuditRepository(db *sql.DB) *MaskingAuditRepository {
	return &MaskingAuditRepository{db: db}
}

// CreateAuditEntry creates a new masking audit log entry
func (r *MaskingAuditRepository) CreateAuditEntry(ctx context.Context, entry *entity.MaskingAudit) error {
	query := `
		INSERT INTO masking_audit_log (
			id, asset_id, masked_by, masking_strategy, findings_count, masked_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		entry.ID,
		entry.AssetID,
		entry.MaskedBy,
		entry.MaskingStrategy,
		entry.FindingsCount,
		entry.MaskedAt,
		entry.Metadata,
	)

	return err
}

// GetAuditLogByAsset retrieves all audit log entries for a specific asset
func (r *MaskingAuditRepository) GetAuditLogByAsset(ctx context.Context, assetID uuid.UUID) ([]entity.MaskingAudit, error) {
	query := `
		SELECT id, asset_id, masked_by, masking_strategy, findings_count, masked_at, metadata, created_at
		FROM masking_audit_log
		WHERE asset_id = $1
		ORDER BY masked_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []entity.MaskingAudit
	for rows.Next() {
		var entry entity.MaskingAudit
		err := rows.Scan(
			&entry.ID,
			&entry.AssetID,
			&entry.MaskedBy,
			&entry.MaskingStrategy,
			&entry.FindingsCount,
			&entry.MaskedAt,
			&entry.Metadata,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// GetLatestAuditEntry retrieves the most recent audit log entry for an asset
func (r *MaskingAuditRepository) GetLatestAuditEntry(ctx context.Context, assetID uuid.UUID) (*entity.MaskingAudit, error) {
	query := `
		SELECT id, asset_id, masked_by, masking_strategy, findings_count, masked_at, metadata, created_at
		FROM masking_audit_log
		WHERE asset_id = $1
		ORDER BY masked_at DESC
		LIMIT 1
	`

	var entry entity.MaskingAudit
	err := r.db.QueryRowContext(ctx, query, assetID).Scan(
		&entry.ID,
		&entry.AssetID,
		&entry.MaskedBy,
		&entry.MaskingStrategy,
		&entry.FindingsCount,
		&entry.MaskedAt,
		&entry.Metadata,
		&entry.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &entry, nil
}
