package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/google/uuid"
)

// AssetService handles asset retrieval and management
// This is the SINGLE SOURCE OF TRUTH for asset lifecycle
type AssetService struct {
	repo        *persistence.PostgresRepository
	lineageSync interfaces.LineageSync
	auditLogger interfaces.AuditLogger
}

// NewAssetService creates a new asset service
func NewAssetService(repo *persistence.PostgresRepository, lineageSync interfaces.LineageSync, auditLogger interfaces.AuditLogger) *AssetService {
	if lineageSync == nil {
		lineageSync = &interfaces.NoOpLineageSync{}
	}
	return &AssetService{
		repo:        repo,
		lineageSync: lineageSync,
		auditLogger: auditLogger,
	}
}

// CreateOrUpdateAsset creates a new asset or updates existing one
// This is the SINGLE SOURCE OF TRUTH for asset creation
// Returns: assetID, isNew, error
func (s *AssetService) CreateOrUpdateAsset(ctx context.Context, asset *entity.Asset) (uuid.UUID, bool, error) {
	// Generate stable ID if not provided
	if asset.StableID == "" {
		asset.StableID = s.generateStableID(asset)
	}

	// Check if asset already exists
	existingAsset, err := s.repo.GetAssetByStableID(ctx, asset.StableID)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("failed to check existing asset: %w", err)
	}

	var assetID uuid.UUID
	var isNew bool

	if existingAsset != nil {
		// Update existing asset
		assetID = existingAsset.ID
		asset.ID = assetID

		// Update metadata if needed (risk score, finding count, etc.)
		// For now, we keep the existing asset and just return its ID
		isNew = false

		log.Printf("📦 Asset already exists: %s (ID: %s)", asset.Name, assetID)

		// Audit Log for Update (Implicit)
		if s.auditLogger != nil {
			_ = s.auditLogger.Record(ctx, "ASSET_ACCESSED", "asset", assetID.String(), map[string]interface{}{
				"stable_id": asset.StableID,
				"action":    "identified_existing",
			})
		}
	} else {
		// Create new asset
		if asset.ID == uuid.Nil {
			asset.ID = uuid.New()
		}

		if err := s.repo.CreateAsset(ctx, asset); err != nil {
			return uuid.Nil, false, fmt.Errorf("failed to create asset: %w", err)
		}

		assetID = asset.ID
		isNew = true

		log.Printf("✅ Created new asset: %s (ID: %s)", asset.Name, assetID)

		// Audit Log for Create
		if s.auditLogger != nil {
			_ = s.auditLogger.Record(ctx, "ASSET_CREATED", "asset", assetID.String(), map[string]interface{}{
				"name":        asset.Name,
				"data_source": asset.DataSource,
				"owner":       asset.Owner,
			})
		}
	}

	// Trigger lineage sync (async, non-blocking)
	if s.lineageSync.IsAvailable() {
		go func() {
			// Use background context to avoid cancellation
			if err := s.lineageSync.SyncAssetToNeo4j(context.Background(), assetID); err != nil {
				// Log error but don't fail asset creation
				log.Printf("⚠️  WARNING: Failed to sync asset %s to lineage: %v", assetID, err)
			} else {
				log.Printf("🔗 Lineage synced for asset: %s", assetID)
			}
		}()
	}

	return assetID, isNew, nil
}

// generateStableID creates a stable identifier from asset properties
func (s *AssetService) generateStableID(asset *entity.Asset) string {
	var identifier string

	if asset.DataSource == "postgresql" || asset.DataSource == "mysql" {
		// For databases: use data source + host + path (table name)
		identifier = fmt.Sprintf("%s::%s::%s", asset.DataSource, asset.Host, asset.Path)
	} else {
		// For filesystem: use file path
		identifier = asset.Path
	}

	// Normalize to lowercase to prevent duplicates on case-insensitive systems
	normalizedPath := strings.ToLower(identifier)
	hash := sha256.Sum256([]byte(normalizedPath))
	return hex.EncodeToString(hash[:])
}

// GetAsset retrieves an asset by ID with full context
func (s *AssetService) GetAsset(ctx context.Context, id uuid.UUID) (*entity.Asset, error) {
	asset, err := s.repo.GetAssetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}
	return asset, nil
}

// GetAssetByStableID retrieves asset by stable identifier
func (s *AssetService) GetAssetByStableID(ctx context.Context, stableID string) (*entity.Asset, error) {
	return s.repo.GetAssetByStableID(ctx, stableID)
}

// UpdateAssetStats updates finding count and risk score
func (s *AssetService) UpdateAssetStats(ctx context.Context, assetID uuid.UUID, riskScore, findingCount int) error {
	return s.repo.UpdateAssetStats(ctx, assetID, riskScore, findingCount)
}

// ListAssets returns a list of assets
func (s *AssetService) ListAssets(ctx context.Context, limit, offset int) ([]*entity.Asset, error) {
	return s.repo.ListAssets(ctx, limit, offset)
}
