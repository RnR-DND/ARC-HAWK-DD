package interfaces

import (
	"context"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/domain/repository"
	"github.com/google/uuid"
)

// FindingsProvider defines the contract for findings retrieval
// This interface allows Lineage Module to access findings without direct PostgreSQL queries
type FindingsProvider interface {
	// GetFindingsByAsset retrieves all findings for a specific asset
	GetFindingsByAsset(ctx context.Context, assetID uuid.UUID, limit, offset int) ([]*entity.Finding, error)

	// GetClassificationsByFinding retrieves classifications for a finding
	GetClassificationsByFinding(ctx context.Context, findingID uuid.UUID) ([]*entity.Classification, error)

	// CountFindings returns total findings matching filters
	CountFindings(ctx context.Context, filters repository.FindingFilters) (int, error)
}
