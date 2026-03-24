package interfaces

// Repository interfaces for inter-module communication
// These define contracts that modules can depend on without tight coupling

import (
	"context"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/google/uuid"
)

// AssetRepository defines operations for asset management
type AssetRepository interface {
	CreateAsset(ctx context.Context, asset *entity.Asset) error
	GetAssetByID(ctx context.Context, id uuid.UUID) (*entity.Asset, error)
	GetAssetByStableID(ctx context.Context, stableID string) (*entity.Asset, error)
	ListAssets(ctx context.Context, limit, offset int) ([]*entity.Asset, error)
	UpdateAssetRiskScore(ctx context.Context, id uuid.UUID, score int) error
	UpdateAssetStats(ctx context.Context, id uuid.UUID, score int, totalFindings int) error
	UpdateMaskingStatus(ctx context.Context, assetID uuid.UUID, isMasked bool, strategy string) error
}

// FindingRepository defines operations for finding management
type FindingRepository interface {
	CreateFinding(ctx context.Context, finding *entity.Finding) error
	GetFindingByID(ctx context.Context, id uuid.UUID) (*entity.Finding, error)
	ListFindingsByAsset(ctx context.Context, assetID uuid.UUID, limit, offset int) ([]*entity.Finding, error)
	UpdateMaskedValues(ctx context.Context, maskedData map[uuid.UUID]string) error
}

// LineageProvider defines operations for lineage management
type LineageProvider interface {
	UpdateAssetLineage(ctx context.Context, asset *entity.Asset) error
	GetLineageGraph(ctx context.Context) (interface{}, error)
}

// ClassificationProvider defines operations for PII classification
type ClassificationProvider interface {
	ClassifyFinding(ctx context.Context, finding *entity.Finding) (*entity.Classification, error)
	GetClassificationSummary(ctx context.Context) (interface{}, error)
}

// MaskingProvider defines operations for data masking
type MaskingProvider interface {
	MaskAsset(ctx context.Context, assetID uuid.UUID, strategy string, maskedBy string) error
	GetMaskingStatus(ctx context.Context, assetID uuid.UUID) (interface{}, error)
}

// AnalyticsProvider defines operations for analytics
type AnalyticsProvider interface {
	GetPIIHeatmap(ctx context.Context) (interface{}, error)
	GetRiskTrend(ctx context.Context) (interface{}, error)
}

// ComplianceProvider defines operations for compliance
type ComplianceProvider interface {
	GetComplianceOverview(ctx context.Context) (interface{}, error)
	GetConsentViolations(ctx context.Context) (interface{}, error)
}
