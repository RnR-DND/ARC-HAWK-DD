package interfaces

import (
	"context"

	"github.com/google/uuid"
)

// LineageSync defines the contract for lineage synchronization
// This interface decouples modules from the concrete Neo4j implementation
type LineageSync interface {
	// SyncAssetToNeo4j syncs a single asset and its findings to the graph
	SyncAssetToNeo4j(ctx context.Context, assetID uuid.UUID) error

	// SyncAllAssets triggers full lineage synchronization
	SyncAllAssets(ctx context.Context) error

	// IsAvailable returns true if lineage service is configured
	IsAvailable() bool
}

// NoOpLineageSync provides a no-op implementation for when lineage is disabled
// This allows graceful degradation when Neo4j is not available
type NoOpLineageSync struct{}

// SyncAssetToNeo4j does nothing (graceful degradation)
func (n *NoOpLineageSync) SyncAssetToNeo4j(ctx context.Context, assetID uuid.UUID) error {
	return nil
}

// SyncAllAssets does nothing (graceful degradation)
func (n *NoOpLineageSync) SyncAllAssets(ctx context.Context) error {
	return nil
}

// IsAvailable always returns false
func (n *NoOpLineageSync) IsAvailable() bool {
	return false
}
