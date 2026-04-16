package service

import (
	"context"
	"fmt"
	"time"

	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// DPDPAObligation identifies a specific DPDPA 2023 section.
type DPDPAObligation string

const (
	ObligationSec4LawfulProcessing  DPDPAObligation = "Sec4_LawfulProcessing"
	ObligationSec5PurposeLimitation DPDPAObligation = "Sec5_PurposeLimitation"
	ObligationSec6Consent           DPDPAObligation = "Sec6_Consent"
	ObligationSec7DataPrincipalRights DPDPAObligation = "Sec7_DataPrincipalRights"
	ObligationSec8DataAccuracy      DPDPAObligation = "Sec8_DataAccuracy"
	ObligationSec9ChildrensData     DPDPAObligation = "Sec9_ChildrensData"
	ObligationSec10DataFiduciary    DPDPAObligation = "Sec10_DataFiduciary"
	ObligationSec11GRO              DPDPAObligation = "Sec11_GRO"
	ObligationSec12CrossBorder      DPDPAObligation = "Sec12_CrossBorder"
	ObligationSec17Retention        DPDPAObligation = "Sec17_Retention"
)

// ObligationStatus is whether a DPDPA obligation is met, violated, or unknown.
type ObligationStatus string

const (
	StatusPass    ObligationStatus = "pass"
	StatusFail    ObligationStatus = "fail"
	StatusUnknown ObligationStatus = "unknown"
)

// ObligationGap is one compliance gap record — one per (asset, obligation) pair.
type ObligationGap struct {
	AssetID     uuid.UUID        `json:"asset_id"`
	AssetName   string           `json:"asset_name"`
	Obligation  DPDPAObligation  `json:"obligation"`
	Status      ObligationStatus `json:"status"`
	Detail      string           `json:"detail"`
	EvidenceIDs []uuid.UUID      `json:"evidence_ids,omitempty"` // finding IDs
}

// ComplianceGapReport is the full DPDPA compliance posture for the tenant.
type ComplianceGapReport struct {
	GeneratedAt    time.Time                          `json:"generated_at"`
	TotalAssets    int                                `json:"total_assets"`
	GapsBySection  map[DPDPAObligation][]ObligationGap `json:"gaps_by_section"`
	Summary        GapSummary                         `json:"summary"`
}

// GapSummary aggregates counts across all sections.
type GapSummary struct {
	TotalGaps      int `json:"total_gaps"`
	PassCount      int `json:"pass_count"`
	FailCount      int `json:"fail_count"`
	UnknownCount   int `json:"unknown_count"`
}

// DPDPAObligationService checks each DPDPA 2023 obligation against live asset data.
type DPDPAObligationService struct {
	pgRepo *persistence.PostgresRepository
}

// NewDPDPAObligationService creates a new obligation mapping service.
func NewDPDPAObligationService(pgRepo *persistence.PostgresRepository) *DPDPAObligationService {
	return &DPDPAObligationService{pgRepo: pgRepo}
}

// BuildGapReport runs all DPDPA obligation checks and returns a ComplianceGapReport.
func (s *DPDPAObligationService) BuildGapReport(ctx context.Context) (*ComplianceGapReport, error) {
	report := &ComplianceGapReport{
		GeneratedAt:   time.Now().UTC(),
		GapsBySection: make(map[DPDPAObligation][]ObligationGap),
	}

	checks := []func(context.Context, *ComplianceGapReport) error{
		s.checkSec4LawfulProcessing,
		s.checkSec5PurposeLimitation,
		s.checkSec6Consent,
		s.checkSec7DataPrincipalRights,
		s.checkSec8DataAccuracy,
		s.checkSec9ChildrensData,
		s.checkSec10DataFiduciary,
		s.checkSec11GRO,
		s.checkSec12CrossBorder,
		s.checkSec17Retention,
	}

	for _, check := range checks {
		if err := check(ctx, report); err != nil {
			return nil, err
		}
	}

	// Count total assets from any section's gaps (approximate from distinct asset IDs)
	assetSeen := make(map[uuid.UUID]bool)
	for _, gaps := range report.GapsBySection {
		for _, g := range gaps {
			assetSeen[g.AssetID] = true
			switch g.Status {
			case StatusPass:
				report.Summary.PassCount++
			case StatusFail:
				report.Summary.FailCount++
			default:
				report.Summary.UnknownCount++
			}
			report.Summary.TotalGaps++
		}
	}
	report.TotalAssets = len(assetSeen)
	return report, nil
}

// checkSec4LawfulProcessing — Sec 4: assets with PII requiring consent must have an active consent record.
func (s *DPDPAObligationService) checkSec4LawfulProcessing(ctx context.Context, report *ComplianceGapReport) error {
	// Only returns rows where consent IS required (HAVING) AND no active consent record exists.
	// Every row returned is therefore a FAIL.
	const query = `
		SELECT
			a.id,
			a.name,
			COALESCE(ARRAY_AGG(DISTINCT f.id::text) FILTER (WHERE cl.requires_consent = TRUE), ARRAY[]::text[]) AS evidence_ids
		FROM assets a
		JOIN findings f ON f.asset_id = a.id
		JOIN classifications cl ON cl.finding_id = f.id
		LEFT JOIN consent_records cr ON cr.asset_id = a.id AND cr.is_active = TRUE
		GROUP BY a.id, a.name
		HAVING BOOL_OR(cl.requires_consent) = TRUE AND COUNT(cr.id) = 0
	`
	rows, err := s.pgRepo.GetDB().QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("sec4 check: %w", err)
	}
	defer rows.Close()

	var gaps []ObligationGap
	for rows.Next() {
		var assetID uuid.UUID
		var assetName string
		var evidenceIDStrs []string
		if err := rows.Scan(&assetID, &assetName, pq.Array(&evidenceIDStrs)); err != nil {
			continue
		}
		gap := ObligationGap{
			AssetID:    assetID,
			AssetName:  assetName,
			Obligation: ObligationSec4LawfulProcessing,
			Status:     StatusFail,
			Detail:     "Asset contains PII requiring consent but no active consent record found (DPDPA Sec 4)",
		}
		for _, eidStr := range evidenceIDStrs {
			if id, err := uuid.Parse(eidStr); err == nil {
				gap.EvidenceIDs = append(gap.EvidenceIDs, id)
			}
		}
		gaps = append(gaps, gap)
	}
	report.GapsBySection[ObligationSec4LawfulProcessing] = gaps
	return rows.Err()
}

// checkSec5PurposeLimitation — Sec 5: does the asset have a declared_purpose tag?
func (s *DPDPAObligationService) checkSec5PurposeLimitation(ctx context.Context, report *ComplianceGapReport) error {
	// Use the dedicated column added in migration 000033; fall back to JSONB for
	// deployments that have not yet run the migration.
	const query = `
		SELECT a.id, a.name,
			COALESCE(a.declared_purpose, a.file_metadata->>'declared_purpose') AS declared_purpose
		FROM assets a
		WHERE EXISTS (SELECT 1 FROM findings f WHERE f.asset_id = a.id)
	`
	rows, err := s.pgRepo.GetDB().QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("sec5 check: %w", err)
	}
	defer rows.Close()

	var gaps []ObligationGap
	for rows.Next() {
		var assetID uuid.UUID
		var assetName string
		var declaredPurpose *string
		if err := rows.Scan(&assetID, &assetName, &declaredPurpose); err != nil {
			continue
		}
		status := StatusPass
		detail := "Declared purpose tag present"
		if declaredPurpose == nil || *declaredPurpose == "" {
			status = StatusFail
			detail = "No declared_purpose metadata on asset — purpose limitation cannot be verified"
		}
		gaps = append(gaps, ObligationGap{
			AssetID:    assetID,
			AssetName:  assetName,
			Obligation: ObligationSec5PurposeLimitation,
			Status:     status,
			Detail:     detail,
		})
	}
	report.GapsBySection[ObligationSec5PurposeLimitation] = gaps
	return rows.Err()
}

// checkSec6Consent — Sec 6: independently verify consent record health for each asset.
// Queries consent_status_view for expired or withdrawn consents.
func (s *DPDPAObligationService) checkSec6Consent(ctx context.Context, report *ComplianceGapReport) error {
	const query = `
		SELECT csv.asset_id, a.name, COUNT(*) AS invalid_count
		FROM consent_status_view csv
		JOIN assets a ON a.id = csv.asset_id
		WHERE csv.status IN ('WITHDRAWN', 'EXPIRED')
		  AND csv.asset_id IS NOT NULL
		GROUP BY csv.asset_id, a.name
	`
	rows, err := s.pgRepo.GetDB().QueryContext(ctx, query)
	if err != nil {
		// consent_status_view may not exist in older deployments — skip gracefully.
		report.GapsBySection[ObligationSec6Consent] = nil
		return nil
	}
	defer rows.Close()

	var gaps []ObligationGap
	for rows.Next() {
		var assetID uuid.UUID
		var assetName string
		var invalidCount int
		if err := rows.Scan(&assetID, &assetName, &invalidCount); err != nil {
			continue
		}
		gaps = append(gaps, ObligationGap{
			AssetID:    assetID,
			AssetName:  assetName,
			Obligation: ObligationSec6Consent,
			Status:     StatusFail,
			Detail:     fmt.Sprintf("Asset has %d expired or withdrawn consent record(s) — re-obtain consent (DPDPA Sec 6)", invalidCount),
		})
	}
	report.GapsBySection[ObligationSec6Consent] = gaps
	return rows.Err()
}

// checkSec8DataAccuracy — Sec 8: flag assets not re-scanned within 90 days (stale data).
func (s *DPDPAObligationService) checkSec8DataAccuracy(ctx context.Context, report *ComplianceGapReport) error {
	const query = `
		SELECT a.id, a.name, MAX(sr.scan_completed_at) AS last_scan
		FROM assets a
		LEFT JOIN findings f ON f.asset_id = a.id
		LEFT JOIN scan_runs sr ON sr.id = f.scan_run_id
		GROUP BY a.id, a.name
	`
	rows, err := s.pgRepo.GetDB().QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("sec8 check: %w", err)
	}
	defer rows.Close()

	// Try to get tenant-specific threshold, fall back to 90 days.
	// TODO: Query tenant settings for data_accuracy_rescan_days when available.
	rescanDays := 90
	threshold := time.Now().AddDate(0, 0, -rescanDays)
	var gaps []ObligationGap
	for rows.Next() {
		var assetID uuid.UUID
		var assetName string
		var lastScan *time.Time
		if err := rows.Scan(&assetID, &assetName, &lastScan); err != nil {
			continue
		}
		status := StatusPass
		detail := fmt.Sprintf("Data scanned within %d days", rescanDays)
		if lastScan == nil || lastScan.Before(threshold) {
			status = StatusFail
			detail = fmt.Sprintf("Asset data not re-scanned within %d days — accuracy cannot be confirmed. Configure tenant-specific rescan interval via tenant settings. (DPDPA Sec 8)", rescanDays)
		}
		gaps = append(gaps, ObligationGap{
			AssetID:    assetID,
			AssetName:  assetName,
			Obligation: ObligationSec8DataAccuracy,
			Status:     status,
			Detail:     detail,
		})
	}
	report.GapsBySection[ObligationSec8DataAccuracy] = gaps
	return rows.Err()
}

// checkSec9ChildrensData — Sec 9: age-indicator fields flagged separately?
func (s *DPDPAObligationService) checkSec9ChildrensData(ctx context.Context, report *ComplianceGapReport) error {
	const query = `
		SELECT DISTINCT a.id, a.name, f.id AS finding_id
		FROM assets a
		JOIN findings f ON f.asset_id = a.id
		JOIN classifications cl ON cl.finding_id = f.id
		WHERE cl.dpdpa_category = 'AGE_INDICATOR'
		   OR cl.sub_category ILIKE '%age%'
		   OR cl.sub_category ILIKE '%dob%'
		   OR cl.sub_category ILIKE '%date_of_birth%'
	`
	rows, err := s.pgRepo.GetDB().QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("sec9 check: %w", err)
	}
	defer rows.Close()

	assetFindings := make(map[uuid.UUID]struct {
		name    string
		findIDs []uuid.UUID
	})
	for rows.Next() {
		var assetID uuid.UUID
		var assetName string
		var findingID uuid.UUID
		if err := rows.Scan(&assetID, &assetName, &findingID); err != nil {
			continue
		}
		e := assetFindings[assetID]
		e.name = assetName
		e.findIDs = append(e.findIDs, findingID)
		assetFindings[assetID] = e
	}
	if err := rows.Err(); err != nil {
		return err
	}

	var gaps []ObligationGap
	for assetID, info := range assetFindings {
		gaps = append(gaps, ObligationGap{
			AssetID:     assetID,
			AssetName:   info.name,
			Obligation:  ObligationSec9ChildrensData,
			Status:      StatusFail,
			Detail:      "Asset contains age-indicator fields — children's data processing requires explicit verification (DPDPA Sec 9)",
			EvidenceIDs: info.findIDs,
		})
	}

	// If no assets with age-indicator patterns were found, determine whether this is
	// because there is genuinely no children's data, or because no assets have been
	// scanned at all (NOT_ASSESSED).
	if len(assetFindings) == 0 {
		var totalAssetsScanned int
		_ = s.pgRepo.GetDB().QueryRowContext(ctx, `
			SELECT COUNT(DISTINCT a.id) FROM assets a
			WHERE EXISTS (SELECT 1 FROM findings f WHERE f.asset_id = a.id)
		`).Scan(&totalAssetsScanned)

		if totalAssetsScanned == 0 {
			gaps = append(gaps, ObligationGap{
				AssetID:    uuid.Nil,
				AssetName:  "",
				Obligation: ObligationSec9ChildrensData,
				Status:     StatusUnknown,
				Detail:     "No assets have been scanned for children's data indicators (AGE_INDICATOR pattern). Configure scanner to detect age-related PII and rescan all assets.",
			})
		}
	}

	report.GapsBySection[ObligationSec9ChildrensData] = gaps
	return nil
}

// checkSec10DataFiduciary — Sec 10: Significant Data Fiduciaries must have a DPO assigned.
func (s *DPDPAObligationService) checkSec10DataFiduciary(ctx context.Context, report *ComplianceGapReport) error {
	const query = `
		SELECT a.id, a.name,
			COALESCE(a.dpo_assigned, a.file_metadata->>'dpo_assigned') AS dpo_assigned
		FROM assets a
		JOIN tenants t ON t.id = a.tenant_id
		WHERE t.is_significant_data_fiduciary = TRUE
	`
	rows, err := s.pgRepo.GetDB().QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("sec10 check: %w", err)
	}
	defer rows.Close()

	var gaps []ObligationGap
	for rows.Next() {
		var assetID uuid.UUID
		var assetName string
		var dpoAssigned *string
		if err := rows.Scan(&assetID, &assetName, &dpoAssigned); err != nil {
			continue
		}
		status := StatusPass
		detail := "DPO assigned for Significant Data Fiduciary asset"
		if dpoAssigned == nil || *dpoAssigned == "" {
			status = StatusFail
			detail = "Significant Data Fiduciary asset has no DPO assigned (DPDPA Sec 10)"
		}
		gaps = append(gaps, ObligationGap{
			AssetID:    assetID,
			AssetName:  assetName,
			Obligation: ObligationSec10DataFiduciary,
			Status:     status,
			Detail:     detail,
		})
	}
	report.GapsBySection[ObligationSec10DataFiduciary] = gaps
	return rows.Err()
}

// checkSec7DataPrincipalRights — Sec 7: no data principal requests pending > 30 days.
func (s *DPDPAObligationService) checkSec7DataPrincipalRights(ctx context.Context, report *ComplianceGapReport) error {
	const query = `
		SELECT t.id, t.name, COUNT(dpr.id) AS overdue_count
		FROM tenants t
		JOIN data_principal_requests dpr ON dpr.tenant_id = t.id
		WHERE dpr.status = 'PENDING' AND dpr.created_at < NOW() - INTERVAL '30 days'
		GROUP BY t.id, t.name
	`
	rows, err := s.pgRepo.GetDB().QueryContext(ctx, query)
	if err != nil {
		// Table may not exist in older deployments — degrade gracefully
		report.GapsBySection[ObligationSec7DataPrincipalRights] = nil
		return nil
	}
	defer rows.Close()

	var gaps []ObligationGap
	for rows.Next() {
		var tenantID uuid.UUID
		var tenantName string
		var overdueCount int
		if err := rows.Scan(&tenantID, &tenantName, &overdueCount); err != nil {
			continue
		}
		gaps = append(gaps, ObligationGap{
			AssetID:    tenantID,
			AssetName:  tenantName + " [Tenant]",
			Obligation: ObligationSec7DataPrincipalRights,
			Status:     StatusFail,
			Detail:     fmt.Sprintf("%d data principal request(s) pending > 30 days violates DPDPA Sec 7 response timeline", overdueCount),
		})
	}
	report.GapsBySection[ObligationSec7DataPrincipalRights] = gaps
	return rows.Err()
}

// checkSec11GRO — Sec 11: Grievance Redressal Officer must be configured for each tenant.
func (s *DPDPAObligationService) checkSec11GRO(ctx context.Context, report *ComplianceGapReport) error {
	const query = `
		SELECT id, name FROM tenants
		WHERE gro_email IS NULL OR gro_email = ''
	`
	rows, err := s.pgRepo.GetDB().QueryContext(ctx, query)
	if err != nil {
		// Column may not exist in older deployments — degrade gracefully
		report.GapsBySection[ObligationSec11GRO] = nil
		return nil
	}
	defer rows.Close()

	var gaps []ObligationGap
	for rows.Next() {
		var tenantID uuid.UUID
		var tenantName string
		if err := rows.Scan(&tenantID, &tenantName); err != nil {
			continue
		}
		gaps = append(gaps, ObligationGap{
			AssetID:    tenantID,
			AssetName:  tenantName + " [Tenant]",
			Obligation: ObligationSec11GRO,
			Status:     StatusFail,
			Detail:     "Grievance Redressal Officer email not configured — required by DPDPA Sec 11",
		})
	}
	report.GapsBySection[ObligationSec11GRO] = gaps
	return rows.Err()
}

// checkSec12CrossBorder — Sec 12: assets stored outside India require cross-border transfer compliance.
func (s *DPDPAObligationService) checkSec12CrossBorder(ctx context.Context, report *ComplianceGapReport) error {
	const query = `
		SELECT a.id, a.name, a.data_residency_country
		FROM assets a
		WHERE a.data_residency_country IS NOT NULL
		  AND a.data_residency_country NOT IN ('IN')
	`
	rows, err := s.pgRepo.GetDB().QueryContext(ctx, query)
	if err != nil {
		// Column may not exist in older deployments — degrade gracefully
		report.GapsBySection[ObligationSec12CrossBorder] = nil
		return nil
	}
	defer rows.Close()

	var gaps []ObligationGap
	for rows.Next() {
		var assetID uuid.UUID
		var assetName, country string
		if err := rows.Scan(&assetID, &assetName, &country); err != nil {
			continue
		}
		gaps = append(gaps, ObligationGap{
			AssetID:    assetID,
			AssetName:  assetName,
			Obligation: ObligationSec12CrossBorder,
			Status:     StatusFail,
			Detail:     fmt.Sprintf("Asset stored in %q (outside India) requires cross-border transfer compliance documentation (DPDPA Sec 12)", country),
		})
	}
	report.GapsBySection[ObligationSec12CrossBorder] = gaps
	return rows.Err()
}

// checkSec17Retention — Sec 17: retention policy defined and no violations.
func (s *DPDPAObligationService) checkSec17Retention(ctx context.Context, report *ComplianceGapReport) error {
	// Assets with retention violations
	const violationQuery = `
		SELECT DISTINCT asset_id, asset_name
		FROM retention_violations
	`
	vRows, err := s.pgRepo.GetDB().QueryContext(ctx, violationQuery)
	if err != nil {
		// retention_violations view may not exist in older deployments
		report.GapsBySection[ObligationSec17Retention] = nil
		return nil
	}
	defer vRows.Close()

	violatingAssets := make(map[uuid.UUID]string)
	for vRows.Next() {
		var assetID uuid.UUID
		var assetName string
		if err := vRows.Scan(&assetID, &assetName); err != nil {
			continue
		}
		violatingAssets[assetID] = assetName
	}

	// Assets with no retention policy at all
	const noPolicyQuery = `
		SELECT a.id, a.name
		FROM assets a
		WHERE a.retention_policy_days IS NULL
		  AND EXISTS (SELECT 1 FROM findings f WHERE f.asset_id = a.id)
	`
	pRows, err := s.pgRepo.GetDB().QueryContext(ctx, noPolicyQuery)
	if err != nil {
		return fmt.Errorf("sec17 no-policy check: %w", err)
	}
	defer pRows.Close()

	noPolicyAssets := make(map[uuid.UUID]string)
	for pRows.Next() {
		var assetID uuid.UUID
		var assetName string
		if err := pRows.Scan(&assetID, &assetName); err != nil {
			continue
		}
		noPolicyAssets[assetID] = assetName
	}

	var gaps []ObligationGap
	for assetID, name := range violatingAssets {
		gaps = append(gaps, ObligationGap{
			AssetID:    assetID,
			AssetName:  name,
			Obligation: ObligationSec17Retention,
			Status:     StatusFail,
			Detail:     "Asset has findings past their retention deadline (DPDPA Sec 17)",
		})
	}
	for assetID, name := range noPolicyAssets {
		if _, alreadyFailed := violatingAssets[assetID]; !alreadyFailed {
			gaps = append(gaps, ObligationGap{
				AssetID:    assetID,
				AssetName:  name,
				Obligation: ObligationSec17Retention,
				Status:     StatusFail,
				Detail:     "No retention policy defined for asset containing PII (DPDPA Sec 17)",
			})
		}
	}
	report.GapsBySection[ObligationSec17Retention] = gaps
	return nil
}
