package persistence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/domain/repository"
	"github.com/google/uuid"
)

// ============================================================================
// AssetRelationshipRepository Implementation
// ============================================================================

func (r *PostgresRepository) CreateAssetRelationship(ctx context.Context, relationship *entity.AssetRelationship) error {
	metadataJSON, err := json.Marshal(relationship.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO asset_relationships (id, source_asset_id, target_asset_id, relationship_type, metadata)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (source_asset_id, target_asset_id, relationship_type) DO NOTHING
		RETURNING created_at`

	err = r.db.QueryRowContext(ctx, query,
		relationship.ID, relationship.SourceAssetID, relationship.TargetAssetID,
		relationship.RelationshipType, metadataJSON,
	).Scan(&relationship.CreatedAt)

	// Ignore conflict errors (duplicate relationships)
	if err != nil && err.Error() == "sql: no rows in result set" {
		return nil
	}

	return err
}

func (r *PostgresRepository) GetAssetRelationshipsBySourceAsset(ctx context.Context, sourceAssetID uuid.UUID) ([]*entity.AssetRelationship, error) {
	query := `
		SELECT id, source_asset_id, target_asset_id, relationship_type, metadata, created_at
		FROM asset_relationships 
		WHERE source_asset_id = $1`

	rows, err := r.db.QueryContext(ctx, query, sourceAssetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRelationships(rows)
}

func (r *PostgresRepository) GetAllAssetRelationships(ctx context.Context) ([]*entity.AssetRelationship, error) {
	query := `
		SELECT id, source_asset_id, target_asset_id, relationship_type, metadata, created_at
		FROM asset_relationships`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRelationships(rows)
}

func (r *PostgresRepository) GetFilteredAssetRelationships(ctx context.Context, filters repository.RelationshipFilters) ([]*entity.AssetRelationship, error) {
	query := `
		SELECT id, source_asset_id, target_asset_id, relationship_type, metadata, created_at
		FROM asset_relationships WHERE 1=1`

	args := []interface{}{}
	argCount := 1

	if filters.RelationshipType != "" {
		query += fmt.Sprintf(" AND relationship_type = $%d", argCount)
		args = append(args, filters.RelationshipType)
		argCount++
	}

	if filters.SourceAssetID != nil {
		query += fmt.Sprintf(" AND source_asset_id = $%d", argCount)
		args = append(args, *filters.SourceAssetID)
		argCount++
	}

	if filters.TargetAssetID != nil {
		query += fmt.Sprintf(" AND target_asset_id = $%d", argCount)
		args = append(args, *filters.TargetAssetID)
		argCount++
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRelationships(rows)
}
