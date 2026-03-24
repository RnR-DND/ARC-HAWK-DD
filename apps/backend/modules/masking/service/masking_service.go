package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// MaskingService handles data masking operations
type MaskingService struct {
	assetRepo        *persistence.PostgresRepository
	findingRepo      *persistence.PostgresRepository
	maskingAuditRepo *persistence.MaskingAuditRepository
}

// NewMaskingService creates a new masking service
func NewMaskingService(
	assetRepo *persistence.PostgresRepository,
	findingRepo *persistence.PostgresRepository,
	maskingAuditRepo *persistence.MaskingAuditRepository,
) *MaskingService {
	return &MaskingService{
		assetRepo:        assetRepo,
		findingRepo:      findingRepo,
		maskingAuditRepo: maskingAuditRepo,
	}
}

// MaskingStrategy defines the masking approach
type MaskingStrategy string

const (
	MaskingStrategyRedact   MaskingStrategy = "REDACT"
	MaskingStrategyPartial  MaskingStrategy = "PARTIAL"
	MaskingStrategyTokenize MaskingStrategy = "TOKENIZE"
)

// MaskAsset masks all findings for a given asset
func (s *MaskingService) MaskAsset(ctx context.Context, assetID uuid.UUID, strategy MaskingStrategy, maskedBy string) error {
	// Validate strategy
	if !isValidStrategy(strategy) {
		return fmt.Errorf("invalid masking strategy: %s", strategy)
	}

	// Get the asset
	asset, err := s.assetRepo.GetAssetByID(ctx, assetID)
	if err != nil {
		return fmt.Errorf("failed to get asset: %w", err)
	}

	// Check if already masked
	if asset.IsMasked {
		return fmt.Errorf("asset is already masked")
	}

	// Get all findings for this asset
	findings, err := s.findingRepo.ListFindingsByAsset(ctx, assetID, 10000, 0)
	if err != nil {
		return fmt.Errorf("failed to get findings: %w", err)
	}

	if len(findings) == 0 {
		return fmt.Errorf("no findings to mask for this asset")
	}

	// Apply masking to each finding
	maskedData := make(map[uuid.UUID]string)
	for _, finding := range findings {
		if len(finding.Matches) > 0 {
			// Mask the first match (representative value)
			originalValue := finding.Matches[0]
			maskedValue := s.applyMaskingStrategy(originalValue, finding.PatternName, strategy)
			maskedData[finding.ID] = maskedValue
		}
	}

	// Update findings with masked values
	if err := s.findingRepo.UpdateMaskedValues(ctx, maskedData); err != nil {
		return fmt.Errorf("failed to update masked values: %w", err)
	}

	// Update asset masking status
	if err := s.assetRepo.UpdateMaskingStatus(ctx, assetID, true, string(strategy)); err != nil {
		return fmt.Errorf("failed to update asset masking status: %w", err)
	}

	// Create audit log entry
	auditEntry := &entity.MaskingAudit{
		ID:              uuid.New(),
		AssetID:         assetID,
		MaskedBy:        maskedBy,
		MaskingStrategy: string(strategy),
		FindingsCount:   len(findings),
		MaskedAt:        time.Now(),
		Metadata: map[string]interface{}{
			"asset_name": asset.Name,
			"asset_path": asset.Path,
		},
	}

	if err := s.maskingAuditRepo.CreateAuditEntry(ctx, auditEntry); err != nil {
		return fmt.Errorf("failed to create audit entry: %w", err)
	}

	return nil
}

// applyMaskingStrategy applies the specified masking strategy to a value
func (s *MaskingService) applyMaskingStrategy(value, piiType string, strategy MaskingStrategy) string {
	switch strategy {
	case MaskingStrategyRedact:
		return "[REDACTED]"

	case MaskingStrategyPartial:
		return s.applyPartialMasking(value, piiType)

	case MaskingStrategyTokenize:
		return s.applyTokenization(value)

	default:
		return "[REDACTED]"
	}
}

// applyPartialMasking masks the middle portion of a value, keeping first and last characters
func (s *MaskingService) applyPartialMasking(value, piiType string) string {
	// Remove whitespace and special characters for processing
	cleaned := strings.ReplaceAll(value, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	length := len(cleaned)

	// For very short values, just redact
	if length <= 4 {
		return "[REDACTED]"
	}

	// Different strategies based on PII type
	switch {
	case strings.Contains(strings.ToUpper(piiType), "AADHAAR"):
		// Aadhaar: Show last 4 digits (e.g., XXXX-XXXX-1234)
		if length >= 12 {
			return "XXXX-XXXX-" + cleaned[length-4:]
		}
		return "XXXX-XXXX-" + cleaned[length-4:]

	case strings.Contains(strings.ToUpper(piiType), "PAN"):
		// PAN: Show first 3 and last 4 (e.g., ABC****1234)
		if length >= 10 {
			return cleaned[:3] + "****" + cleaned[length-4:]
		}
		return cleaned[:2] + "****" + cleaned[length-2:]

	case strings.Contains(strings.ToUpper(piiType), "PHONE"):
		// Phone: Show last 4 digits (e.g., ******1234)
		if length >= 10 {
			return "******" + cleaned[length-4:]
		}
		return "****" + cleaned[length-4:]

	case strings.Contains(strings.ToUpper(piiType), "EMAIL"):
		// Email: Show first 2 chars and domain (e.g., ab****@example.com)
		parts := strings.Split(value, "@")
		if len(parts) == 2 && len(parts[0]) > 2 {
			return parts[0][:2] + "****@" + parts[1]
		}
		return "****@" + parts[len(parts)-1]

	default:
		// Generic: Show first 2 and last 4
		if length > 6 {
			return cleaned[:2] + strings.Repeat("X", length-6) + cleaned[length-4:]
		}
		return cleaned[:1] + strings.Repeat("X", length-2) + cleaned[length-1:]
	}
}

// applyTokenization generates a consistent token for a value
func (s *MaskingService) applyTokenization(value string) string {
	// Generate SHA-256 hash
	hash := sha256.Sum256([]byte(value))
	hashStr := hex.EncodeToString(hash[:])

	// Return first 16 characters as token
	return "TOKEN_" + strings.ToUpper(hashStr[:16])
}

// GetMaskingStatus retrieves the masking status of an asset
func (s *MaskingService) GetMaskingStatus(ctx context.Context, assetID uuid.UUID) (*MaskingStatusResponse, error) {
	asset, err := s.assetRepo.GetAssetByID(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	response := &MaskingStatusResponse{
		AssetID:         assetID.String(),
		IsMasked:        asset.IsMasked,
		MaskingStrategy: asset.MaskingStrategy,
		MaskedAt:        asset.MaskedAt,
		FindingsCount:   asset.TotalFindings,
	}

	return response, nil
}

// GetMaskingAuditLog retrieves the audit log for an asset
func (s *MaskingService) GetMaskingAuditLog(ctx context.Context, assetID uuid.UUID) ([]entity.MaskingAudit, error) {
	return s.maskingAuditRepo.GetAuditLogByAsset(ctx, assetID)
}

// MaskingStatusResponse represents the masking status of an asset
type MaskingStatusResponse struct {
	AssetID         string     `json:"asset_id"`
	IsMasked        bool       `json:"is_masked"`
	MaskingStrategy string     `json:"masking_strategy,omitempty"`
	MaskedAt        *time.Time `json:"masked_at,omitempty"`
	FindingsCount   int        `json:"findings_count"`
}

// isValidStrategy checks if a masking strategy is valid
func isValidStrategy(strategy MaskingStrategy) bool {
	switch strategy {
	case MaskingStrategyRedact, MaskingStrategyPartial, MaskingStrategyTokenize:
		return true
	default:
		return false
	}
}
