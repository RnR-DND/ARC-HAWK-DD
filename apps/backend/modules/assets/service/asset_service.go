package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// DeleteAsset deletes an asset and all associated data
func (s *AssetService) DeleteAsset(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteAsset(ctx, id); err != nil {
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	if s.auditLogger != nil {
		_ = s.auditLogger.Record(ctx, "ASSET_DELETED", "asset", id.String(), nil)
	}

	return nil
}

// ListAssets returns a list of assets
func (s *AssetService) ListAssets(ctx context.Context, limit, offset int) ([]*entity.Asset, error) {
	return s.repo.ListAssets(ctx, limit, offset)
}

// BulkTagRequest describes a bulk tag operation.
type BulkTagRequest struct {
	AssetIDs []uuid.UUID       `json:"asset_ids"`
	Tags     map[string]string `json:"tags"` // key → value
	Mode     string            `json:"mode"` // "merge" | "replace" — "merge" is default
	// ManualOverride: if true, these tags override any existing manual tags.
	// If false (default), manual tags (set via single-asset edit) win over bulk tags.
	ManualOverride bool `json:"manual_override"`
}

// BulkTagResult is returned immediately; actual work runs in the background.
type BulkTagResult struct {
	JobID      string `json:"job_id"`
	AssetCount int    `json:"asset_count"`
	Status     string `json:"status"` // "queued"
}

// BulkTagAssets queues a background goroutine to apply tags to a batch of assets.
// Imports: encoding/json is imported at file-level via standard module tooling.
// Tag conflict resolution: if ManualOverride is false (default), existing tags that
// were set via a single-asset manual edit (file_metadata->>'_manual_tag_source' = 'manual')
// are preserved and the bulk value is ignored for those keys.
func (s *AssetService) BulkTagAssets(ctx context.Context, req *BulkTagRequest, actor string) (*BulkTagResult, error) {
	if len(req.AssetIDs) == 0 {
		return nil, fmt.Errorf("no asset IDs provided")
	}
	if len(req.Tags) == 0 {
		return nil, fmt.Errorf("no tags provided")
	}

	jobID := uuid.New().String()
	result := &BulkTagResult{
		JobID:      jobID,
		AssetCount: len(req.AssetIDs),
		Status:     "queued",
	}

	// Fire-and-forget background goroutine — returns 202 immediately to caller.
	go func() {
		bgCtx := context.Background()
		success := 0
		for _, assetID := range req.AssetIDs {
			if err := s.applyTagsToAsset(bgCtx, assetID, req.Tags, req.Mode, req.ManualOverride); err != nil {
				log.Printf("[BULK-TAG] job=%s asset=%s error: %v", jobID, assetID, err)
			} else {
				success++
			}
		}
		log.Printf("[BULK-TAG] job=%s complete: %d/%d assets tagged", jobID, success, len(req.AssetIDs))
		if s.auditLogger != nil {
			_ = s.auditLogger.Record(bgCtx, "BULK_TAG_COMPLETE", "job", jobID, map[string]interface{}{
				"actor": actor, "asset_count": len(req.AssetIDs), "success": success,
			})
		}
	}()

	return result, nil
}

// applyTagsToAsset merges or replaces tags in file_metadata JSONB for one asset.
// Manual tags (file_metadata->>'_manual_tag_source' = 'manual') are preserved unless
// ManualOverride is true.
func (s *AssetService) applyTagsToAsset(ctx context.Context, assetID uuid.UUID, tags map[string]string, mode string, manualOverride bool) error {
	// Build JSONB update: use jsonb_set or || operator.
	// Using a safe UPDATE with COALESCE ensures we don't lose existing metadata.
	if mode == "replace" {
		// Replace entire file_metadata tags namespace (but keep _manual_* keys if not overriding)
		updateSQL := `
			UPDATE assets
			   SET file_metadata = file_metadata || $2::jsonb,
			       updated_at = NOW()
			 WHERE id = $1
		`
		if !manualOverride {
			// Preserve manual keys — merge with manual values winning
			updateSQL = `
				UPDATE assets
				   SET file_metadata = ($2::jsonb || COALESCE(
				       (SELECT jsonb_object_agg(key, value)
				          FROM jsonb_each(file_metadata)
				         WHERE value->>'_manual_tag_source' = 'manual'
				            OR key LIKE '_manual_%'),
				       '{}'::jsonb
				   )),
				       updated_at = NOW()
				 WHERE id = $1
			`
		}
		tagsBytes, _ := encodeJSON(tags)
		_, err := s.repo.GetDB().ExecContext(ctx, updateSQL, assetID, string(tagsBytes))
		return err
	}

	// Default: merge — add/update only provided keys, keep everything else
	tagsBytes, _ := encodeJSON(tags)
	tagsJSON := string(tagsBytes)
	var updateSQL string
	if manualOverride {
		updateSQL = `
			UPDATE assets
			   SET file_metadata = COALESCE(file_metadata, '{}'::jsonb) || $2::jsonb,
			       updated_at = NOW()
			 WHERE id = $1
		`
	} else {
		// Build the merged JSONB without overwriting existing manual tags
		updateSQL = `
			UPDATE assets
			   SET file_metadata = (COALESCE(file_metadata, '{}'::jsonb) || $2::jsonb)
			                    || COALESCE(
			                           (SELECT jsonb_object_agg(key, value)
			                              FROM jsonb_each(COALESCE(file_metadata, '{}'::jsonb))
			                             WHERE value->>'_manual_tag_source' = 'manual'),
			                           '{}'::jsonb
			                       ),
			       updated_at = NOW()
			 WHERE id = $1
		`
	}
	_, err := s.repo.GetDB().ExecContext(ctx, updateSQL, assetID, tagsJSON)
	return err
}

// encodeJSON marshals v to JSON bytes.
func encodeJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
