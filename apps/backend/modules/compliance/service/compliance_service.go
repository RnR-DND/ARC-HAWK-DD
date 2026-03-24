package service

import (
	"context"
	"fmt"

	"github.com/arc-platform/backend/modules/shared/domain/repository"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// ComplianceService provides DPDPA compliance posture analytics
type ComplianceService struct {
	pgRepo    *persistence.PostgresRepository
	neo4jRepo *persistence.Neo4jRepository
}

// ComplianceOverview represents the DPDPA compliance dashboard
type ComplianceOverview struct {
	ComplianceScore        float64            `json:"compliance_score"` // % of assets compliant
	TotalAssets            int                `json:"total_assets"`
	CompliantAssets        int                `json:"compliant_assets"`
	NonCompliantAssets     int                `json:"non_compliant_assets"`
	CriticalExposure       *CriticalExposure  `json:"critical_exposure"`
	ConsentViolations      *ConsentViolations `json:"consent_violations"`
	RemediationQueue       []RemediationItem  `json:"remediation_queue"`
	DPDPACategoryBreakdown map[string]int     `json:"dpdpa_category_breakdown"`
}

// CriticalExposure represents assets with critical PII
type CriticalExposure struct {
	TotalAssets      int      `json:"total_assets"`
	CriticalPIITypes []string `json:"critical_pii_types"`
	TotalFindings    int      `json:"total_findings"`
}

// ConsentViolations represents assets requiring consent
type ConsentViolations struct {
	TotalAssets      int      `json:"total_assets"`
	RequiresConsent  int      `json:"requires_consent"`
	MissingConsent   int      `json:"missing_consent"`
	AffectedPIITypes []string `json:"affected_pii_types"`
}

// RemediationItem represents an asset requiring remediation
type RemediationItem struct {
	AssetID      uuid.UUID `json:"asset_id"`
	AssetName    string    `json:"asset_name"`
	AssetPath    string    `json:"asset_path"`
	RiskLevel    string    `json:"risk_level"`
	PIITypes     []string  `json:"pii_types"`
	FindingCount int       `json:"finding_count"`
	Priority     string    `json:"priority"` // critical, high, medium, low
}

// NewComplianceService creates a new compliance service
func NewComplianceService(pgRepo *persistence.PostgresRepository, neo4jRepo *persistence.Neo4jRepository) *ComplianceService {
	return &ComplianceService{
		pgRepo:    pgRepo,
		neo4jRepo: neo4jRepo,
	}
}

// GetComplianceOverview returns the DPDPA compliance dashboard
func (s *ComplianceService) GetComplianceOverview(ctx context.Context) (*ComplianceOverview, error) {
	// Get all assets
	assets, err := s.pgRepo.ListAssets(ctx, 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}

	overview := &ComplianceOverview{
		TotalAssets:            len(assets),
		CompliantAssets:        0,
		NonCompliantAssets:     0,
		RemediationQueue:       []RemediationItem{},
		DPDPACategoryBreakdown: make(map[string]int),
	}

	// Critical PII types (India-specific)
	criticalPIITypes := []string{"IN_AADHAAR", "IN_PAN", "IN_PASSPORT", "CREDIT_CARD"}
	criticalAssets := make(map[uuid.UUID]bool)
	consentRequiredAssets := make(map[uuid.UUID]bool)

	criticalFindingsCount := 0
	consentPIITypes := make(map[string]bool)

	// Analyze each asset
	for _, asset := range assets {
		// Get findings for this asset
		findings, err := s.pgRepo.ListFindings(ctx, repository.FindingFilters{
			AssetID: &asset.ID,
		}, 1000, 0)
		if err != nil {
			continue
		}

		if len(findings) == 0 {
			overview.CompliantAssets++
			continue
		}

		overview.NonCompliantAssets++

		// Analyze findings
		assetPIITypes := make(map[string]bool)
		hasCritical := false
		requiresConsent := false
		totalFindings := 0
		highestSeverity := "Low"

		for _, finding := range findings {
			// Get classification
			classifications, err := s.pgRepo.GetClassificationsByFindingID(ctx, finding.ID)
			if err != nil || len(classifications) == 0 {
				continue
			}

			classification := classifications[0]
			piiType := classification.SubCategory

			if piiType == "" {
				continue
			}

			assetPIITypes[piiType] = true
			totalFindings++

			// Track highest severity
			if finding.Severity == "Critical" {
				highestSeverity = "Critical"
			} else if finding.Severity == "High" && highestSeverity != "Critical" {
				highestSeverity = "High"
			} else if finding.Severity == "Medium" && highestSeverity != "Critical" && highestSeverity != "High" {
				highestSeverity = "Medium"
			}

			// Check if critical
			for _, criticalType := range criticalPIITypes {
				if piiType == criticalType {
					hasCritical = true
					criticalAssets[asset.ID] = true
					criticalFindingsCount++
					break
				}
			}

			// Check if requires consent
			if classification.RequiresConsent {
				requiresConsent = true
				consentRequiredAssets[asset.ID] = true
				consentPIITypes[piiType] = true
			}

			// Track DPDPA category
			if classification.DPDPACategory != "" {
				overview.DPDPACategoryBreakdown[classification.DPDPACategory]++
			}
		}

		// Add to remediation queue if critical or requires consent
		if hasCritical || requiresConsent {
			piiTypesList := make([]string, 0, len(assetPIITypes))
			for piiType := range assetPIITypes {
				piiTypesList = append(piiTypesList, piiType)
			}

			priority := "medium"
			if hasCritical {
				priority = "critical"
			} else if requiresConsent {
				priority = "high"
			}

			overview.RemediationQueue = append(overview.RemediationQueue, RemediationItem{
				AssetID:      asset.ID,
				AssetName:    asset.Name,
				AssetPath:    asset.Path,
				RiskLevel:    highestSeverity,
				PIITypes:     piiTypesList,
				FindingCount: totalFindings,
				Priority:     priority,
			})
		}
	}

	// Calculate compliance score
	if overview.TotalAssets > 0 {
		overview.ComplianceScore = float64(overview.CompliantAssets) / float64(overview.TotalAssets) * 100
	}

	// Build critical exposure
	overview.CriticalExposure = &CriticalExposure{
		TotalAssets:      len(criticalAssets),
		CriticalPIITypes: criticalPIITypes,
		TotalFindings:    criticalFindingsCount,
	}

	// Build consent violations
	consentPIITypesList := make([]string, 0, len(consentPIITypes))
	for piiType := range consentPIITypes {
		consentPIITypesList = append(consentPIITypesList, piiType)
	}

	overview.ConsentViolations = &ConsentViolations{
		TotalAssets:      len(consentRequiredAssets),
		RequiresConsent:  len(consentRequiredAssets),
		MissingConsent:   len(consentRequiredAssets), // Assume all missing for now
		AffectedPIITypes: consentPIITypesList,
	}

	return overview, nil
}

// GetCriticalAssets returns assets with critical PII exposure
func (s *ComplianceService) GetCriticalAssets(ctx context.Context) ([]RemediationItem, error) {
	overview, err := s.GetComplianceOverview(ctx)
	if err != nil {
		return nil, err
	}

	// Filter for critical priority only
	criticalItems := []RemediationItem{}
	for _, item := range overview.RemediationQueue {
		if item.Priority == "critical" {
			criticalItems = append(criticalItems, item)
		}
	}

	return criticalItems, nil
}

// GetConsentViolations returns assets violating consent rules
func (s *ComplianceService) GetConsentViolations(ctx context.Context) ([]RemediationItem, error) {
	overview, err := s.GetComplianceOverview(ctx)
	if err != nil {
		return nil, err
	}

	// Filter for high priority (consent-related)
	consentItems := []RemediationItem{}
	for _, item := range overview.RemediationQueue {
		if item.Priority == "high" || item.Priority == "critical" {
			consentItems = append(consentItems, item)
		}
	}

	return consentItems, nil
}
