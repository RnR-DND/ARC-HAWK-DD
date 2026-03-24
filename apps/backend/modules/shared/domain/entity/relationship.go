package entity

import (
	"time"

	"github.com/google/uuid"
)

// AssetRelationship represents graph edges between assets
type AssetRelationship struct {
	ID               uuid.UUID              `json:"id"`
	SourceAssetID    uuid.UUID              `json:"source_asset_id"`
	TargetAssetID    uuid.UUID              `json:"target_asset_id"`
	RelationshipType string                 `json:"relationship_type"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}
