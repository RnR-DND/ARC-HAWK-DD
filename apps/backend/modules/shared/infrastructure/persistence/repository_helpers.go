package persistence

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
)

func (r *PostgresRepository) scanRelationships(rows *sql.Rows) ([]*entity.AssetRelationship, error) {
	var relationships []*entity.AssetRelationship
	for rows.Next() {
		rel := &entity.AssetRelationship{}
		var metadataJSON []byte

		err := rows.Scan(
			&rel.ID, &rel.SourceAssetID, &rel.TargetAssetID,
			&rel.RelationshipType, &metadataJSON, &rel.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &rel.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		relationships = append(relationships, rel)
	}

	return relationships, rows.Err()
}
