package interfaces

import (
	"context"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/google/uuid"
)

// AssetManager defines the contract for asset lifecycle management
// This interface ensures Assets Module is the single source of truth for asset creation
type AssetManager interface {
	// CreateOrUpdateAsset creates a new asset or updates existing one
	// Returns asset ID, whether it was newly created, and error
	CreateOrUpdateAsset(ctx context.Context, asset *entity.Asset) (uuid.UUID, bool, error)

	// GetAssetByStableID retrieves asset by stable identifier
	GetAssetByStableID(ctx context.Context, stableID string) (*entity.Asset, error)

	// UpdateAssetStats updates finding count and risk score
	UpdateAssetStats(ctx context.Context, assetID uuid.UUID, riskScore, findingCount int) error
}
