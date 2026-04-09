package service

import (
	"context"
	"database/sql"
	"fmt"

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

// assetComplianceRow holds the result of a single aggregation join per asset.
type assetComplianceRow struct {
	assetID         uuid.UUID
	assetName       string
	assetPath       string
	findingCount    int
	hasCritical     bool
	requiresConsent bool
	maxSeverity     string
	piiType         sql.NullString
	dpdpaCategory   sql.NullString
}

// GetComplianceOverview returns the DPDPA compliance dashboard.
// Uses a single aggregation JOIN instead of per-asset queries (fixes N+1 — P0-2).
func (s *ComplianceService) GetComplianceOverview(ctx context.Context) (*ComplianceOverview, error) {
	criticalPIITypes := map[string]bool{
		"IN_AADHAAR":  true,
		"IN_PAN":      true,
		"IN_PASSPORT": true,
		"CREDIT_CARD": true,
	}

	// Single JOIN query: assets → findings → classifications
	// Returns one row per (asset, sub_category) pair so we can aggregate in Go.
	const query = `
		SELECT
			a.id                                        AS asset_id,
			a.name                                      AS asset_name,
			COALESCE(a.path, '')                        AS asset_path,
			COUNT(f.id)                                 AS finding_count,
			BOOL_OR(f.severity IN ('Critical','Highest')) AS has_critical,
			BOOL_OR(cl.requires_consent)                AS requires_consent,
			MAX(CASE f.severity
				WHEN 'Critical' THEN 4 WHEN 'Highest' THEN 4
				WHEN 'High'     THEN 3
				WHEN 'Medium'   THEN 2
				ELSE 1 END)                             AS severity_rank,
			cl.sub_category                             AS pii_type,
			cl.dpdpa_category                           AS dpdpa_category
		FROM assets a
		LEFT JOIN findings f  ON f.asset_id = a.id
		LEFT JOIN classifications cl ON cl.finding_id = f.id
		GROUP BY a.id, a.name, a.path, cl.sub_category, cl.dpdpa_category
		ORDER BY a.id
	`

	rows, err := s.pgRepo.GetDB().QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("compliance overview query failed: %w", err)
	}
	defer rows.Close()

	type assetState struct {
		name            string
		path            string
		findingCount    int
		hasCritical     bool
		requiresConsent bool
		maxSeverityRank int
		piiTypes        map[string]bool
	}

	assetStates := make(map[uuid.UUID]*assetState)
	dpdpaCategoryBreakdown := make(map[string]int)
	consentPIITypes := make(map[string]bool)
	criticalFindingsCount := 0

	for rows.Next() {
		var (
			assetID         uuid.UUID
			assetName       string
			assetPath       string
			findingCount    int
			hasCritical     bool
			requiresConsent bool
			severityRank    int
			piiType         sql.NullString
			dpdpaCategory   sql.NullString
		)
		if err := rows.Scan(&assetID, &assetName, &assetPath, &findingCount,
			&hasCritical, &requiresConsent, &severityRank,
			&piiType, &dpdpaCategory); err != nil {
			return nil, fmt.Errorf("compliance overview scan failed: %w", err)
		}

		st, exists := assetStates[assetID]
		if !exists {
			st = &assetState{
				name:     assetName,
				path:     assetPath,
				piiTypes: make(map[string]bool),
			}
			assetStates[assetID] = st
		}

		if findingCount > st.findingCount {
			st.findingCount = findingCount
		}
		if hasCritical {
			st.hasCritical = true
		}
		if requiresConsent {
			st.requiresConsent = true
		}
		if severityRank > st.maxSeverityRank {
			st.maxSeverityRank = severityRank
		}
		if piiType.Valid && piiType.String != "" {
			st.piiTypes[piiType.String] = true
			if criticalPIITypes[piiType.String] && hasCritical {
				criticalFindingsCount++
			}
			if requiresConsent {
				consentPIITypes[piiType.String] = true
			}
		}
		if dpdpaCategory.Valid && dpdpaCategory.String != "" {
			dpdpaCategoryBreakdown[dpdpaCategory.String]++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("compliance overview rows error: %w", err)
	}

	severityLabel := map[int]string{4: "Critical", 3: "High", 2: "Medium", 1: "Low", 0: "Low"}

	overview := &ComplianceOverview{
		TotalAssets:            len(assetStates),
		RemediationQueue:       []RemediationItem{},
		DPDPACategoryBreakdown: dpdpaCategoryBreakdown,
	}

	criticalAssetIDs := make(map[uuid.UUID]bool)
	consentAssetIDs := make(map[uuid.UUID]bool)

	for assetID, st := range assetStates {
		if st.findingCount == 0 {
			overview.CompliantAssets++
			continue
		}
		overview.NonCompliantAssets++

		if st.hasCritical {
			criticalAssetIDs[assetID] = true
		}
		if st.requiresConsent {
			consentAssetIDs[assetID] = true
		}

		if st.hasCritical || st.requiresConsent {
			piiList := make([]string, 0, len(st.piiTypes))
			for p := range st.piiTypes {
				piiList = append(piiList, p)
			}
			priority := "medium"
			if st.hasCritical {
				priority = "critical"
			} else if st.requiresConsent {
				priority = "high"
			}
			overview.RemediationQueue = append(overview.RemediationQueue, RemediationItem{
				AssetID:      assetID,
				AssetName:    st.name,
				AssetPath:    st.path,
				RiskLevel:    severityLabel[st.maxSeverityRank],
				PIITypes:     piiList,
				FindingCount: st.findingCount,
				Priority:     priority,
			})
		}
	}

	if overview.TotalAssets > 0 {
		overview.ComplianceScore = float64(overview.CompliantAssets) / float64(overview.TotalAssets) * 100
	}

	criticalPIIList := []string{"IN_AADHAAR", "IN_PAN", "IN_PASSPORT", "CREDIT_CARD"}
	overview.CriticalExposure = &CriticalExposure{
		TotalAssets:      len(criticalAssetIDs),
		CriticalPIITypes: criticalPIIList,
		TotalFindings:    criticalFindingsCount,
	}

	consentPIIList := make([]string, 0, len(consentPIITypes))
	for p := range consentPIITypes {
		consentPIIList = append(consentPIIList, p)
	}
	overview.ConsentViolations = &ConsentViolations{
		TotalAssets:      len(consentAssetIDs),
		RequiresConsent:  len(consentAssetIDs),
		MissingConsent:   len(consentAssetIDs),
		AffectedPIITypes: consentPIIList,
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
