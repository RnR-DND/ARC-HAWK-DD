package service

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/arc-platform/backend/modules/shared/infrastructure/audit"
	"github.com/google/uuid"
)

// EvidencePackageService generates a ZIP containing all DPDP Act 2023
// compliance evidence for a tenant on demand.
type EvidencePackageService struct {
	db     *sql.DB
	logger *audit.LedgerLogger
}

// NewEvidencePackageService creates a new EvidencePackageService.
func NewEvidencePackageService(db *sql.DB, logger *audit.LedgerLogger) *EvidencePackageService {
	return &EvidencePackageService{db: db, logger: logger}
}

// EvidencePackage is the result of Generate.
type EvidencePackage struct {
	TenantID    uuid.UUID
	GeneratedAt time.Time
	ZipBytes    []byte
	Filename    string
}

// Generate collects compliance evidence and returns a ZIP archive.
// The generation event is logged to audit_ledger.
func (s *EvidencePackageService) Generate(ctx context.Context, tenantID uuid.UUID, actorID uuid.UUID, actorEmail string) (*EvidencePackage, error) {
	now := time.Now().UTC()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	readme := fmt.Sprintf(`ARC-HAWK-DD DPDP Act 2023 Evidence Package
Generated: %s
Tenant ID: %s

This package contains compliance evidence for the Digital Personal Data
Protection Act 2023 (India). Sections correspond to DPDP Act obligations
under Sections 4-17.

Files:
  01_obligation_scorecard.json  - Overall DPDP compliance scoring per obligation
  02_dpr_log.json               - Data Principal Rights request log (Sec 11)
  03_gro_details.json           - Grievance Redressal Officer details (Sec 11)
  04_consent_records.json       - Consent grants and revocations (Sec 6)
  05_scan_history.json          - PII discovery scan history
  06_remediation_actions.json   - Applied remediations
  07_audit_trail.json           - Immutable audit log (last 90 days)
  08_pii_categories_found.json  - All PII categories discovered
`, now.Format(time.RFC3339), tenantID)

	if err := writeZipEntry(zw, "README.txt", []byte(readme)); err != nil {
		return nil, err
	}

	type section struct {
		name string
		fn   func() (interface{}, error)
	}
	sections := []section{
		{"01_obligation_scorecard.json", func() (interface{}, error) { return s.buildObligationScorecard(ctx, tenantID) }},
		{"02_dpr_log.json", func() (interface{}, error) { return s.buildDPRLog(ctx, tenantID) }},
		{"03_gro_details.json", func() (interface{}, error) { return s.buildGRODetails(ctx, tenantID) }},
		{"04_consent_records.json", func() (interface{}, error) { return s.buildConsentRecords(ctx, tenantID) }},
		{"05_scan_history.json", func() (interface{}, error) { return s.buildScanHistory(ctx, tenantID) }},
		{"06_remediation_actions.json", func() (interface{}, error) { return s.buildRemediationActions(ctx, tenantID) }},
		{"08_pii_categories_found.json", func() (interface{}, error) { return s.buildPIICategories(ctx, tenantID) }},
	}
	for _, sec := range sections {
		data, err := sec.fn()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", sec.name, err)
		}
		if err := writeZipJSON(zw, sec.name, data); err != nil {
			return nil, err
		}
	}

	// Audit trail goes last (largest section)
	auditTrail, err := s.logger.Query(ctx, tenantID, nil, now.AddDate(0, 0, -90), now, 10000)
	if err != nil {
		return nil, fmt.Errorf("07_audit_trail.json: %w", err)
	}
	if err := writeZipJSON(zw, "07_audit_trail.json", auditTrail); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	// Log the generation event itself
	_ = s.logger.Log(ctx, audit.LogEntry{
		TenantID:     tenantID,
		EventType:    audit.EventEvidencePackageGen,
		ActorID:      &actorID,
		ActorEmail:   actorEmail,
		ResourceType: "evidence_package",
		Payload: map[string]interface{}{
			"generated_at": now.Format(time.RFC3339),
			"sections":     9,
		},
	})

	filename := fmt.Sprintf("dpdp_evidence_%s_%s.zip", tenantID.String()[:8], now.Format("2006-01-02"))
	return &EvidencePackage{
		TenantID:    tenantID,
		GeneratedAt: now,
		ZipBytes:    buf.Bytes(),
		Filename:    filename,
	}, nil
}

func (s *EvidencePackageService) buildObligationScorecard(ctx context.Context, tenantID uuid.UUID) (interface{}, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT obligation_id, obligation_name, status, score, last_evaluated_at
		FROM compliance_obligations
		WHERE tenant_id = $1
		ORDER BY obligation_id`, tenantID)
	if err != nil {
		return map[string]interface{}{"obligations": []interface{}{}, "note": "No compliance evaluations recorded yet"}, nil
	}
	defer rows.Close()

	var obligations []map[string]interface{}
	for rows.Next() {
		var obligationID, name, status string
		var score float64
		var evaluatedAt *time.Time
		if err := rows.Scan(&obligationID, &name, &status, &score, &evaluatedAt); err != nil {
			continue
		}
		ob := map[string]interface{}{
			"obligation_id": obligationID,
			"name":          name,
			"status":        status,
			"score":         score,
		}
		if evaluatedAt != nil {
			ob["last_evaluated_at"] = evaluatedAt.Format(time.RFC3339)
		}
		obligations = append(obligations, ob)
	}
	return map[string]interface{}{"obligations": obligations}, nil
}

func (s *EvidencePackageService) buildDPRLog(ctx context.Context, tenantID uuid.UUID) (interface{}, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, request_type, status, principal_email, submitted_at, resolved_at
		FROM dpr_requests
		WHERE tenant_id = $1
		ORDER BY submitted_at DESC`, tenantID)
	if err != nil {
		return map[string]interface{}{"requests": []interface{}{}, "error": "dpr_requests not available"}, nil
	}
	defer rows.Close()

	var requests []map[string]interface{}
	for rows.Next() {
		var id, reqType, status, email string
		var submittedAt time.Time
		var resolvedAt *time.Time
		if err := rows.Scan(&id, &reqType, &status, &email, &submittedAt, &resolvedAt); err != nil {
			continue
		}
		r := map[string]interface{}{
			"id": id, "request_type": reqType, "status": status,
			"principal_email": email, "submitted_at": submittedAt.Format(time.RFC3339),
		}
		if resolvedAt != nil {
			r["resolved_at"] = resolvedAt.Format(time.RFC3339)
		}
		requests = append(requests, r)
	}
	return map[string]interface{}{"dpr_requests": requests}, nil
}

func (s *EvidencePackageService) buildGRODetails(ctx context.Context, tenantID uuid.UUID) (interface{}, error) {
	var name, email, phone, address string
	err := s.db.QueryRowContext(ctx, `
		SELECT name, email, phone, address FROM gro_settings WHERE tenant_id = $1`, tenantID).
		Scan(&name, &email, &phone, &address)
	if err != nil {
		return map[string]interface{}{"gro": nil, "note": "No GRO configured"}, nil
	}
	return map[string]interface{}{"gro": map[string]string{
		"name": name, "email": email, "phone": phone, "address": address,
	}}, nil
}

func (s *EvidencePackageService) buildConsentRecords(ctx context.Context, tenantID uuid.UUID) (interface{}, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, principal_id, purpose, status, granted_at, revoked_at
		FROM consent_records
		WHERE tenant_id = $1
		ORDER BY granted_at DESC
		LIMIT 10000`, tenantID)
	if err != nil {
		return map[string]interface{}{"consent_records": []interface{}{}}, nil
	}
	defer rows.Close()

	var records []map[string]interface{}
	for rows.Next() {
		var id, principalID, purpose, status string
		var grantedAt time.Time
		var revokedAt *time.Time
		if err := rows.Scan(&id, &principalID, &purpose, &status, &grantedAt, &revokedAt); err != nil {
			continue
		}
		r := map[string]interface{}{
			"id": id, "principal_id": principalID, "purpose": purpose,
			"status": status, "granted_at": grantedAt.Format(time.RFC3339),
		}
		if revokedAt != nil {
			r["revoked_at"] = revokedAt.Format(time.RFC3339)
		}
		records = append(records, r)
	}
	return map[string]interface{}{"consent_records": records}, nil
}

func (s *EvidencePackageService) buildScanHistory(ctx context.Context, tenantID uuid.UUID) (interface{}, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, connection_id, status, total_findings, started_at, completed_at
		FROM scan_runs
		WHERE tenant_id = $1
		ORDER BY started_at DESC
		LIMIT 100`, tenantID)
	if err != nil {
		return map[string]interface{}{"scan_runs": []interface{}{}}, nil
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var id, connID, status string
		var totalFindings int
		var startedAt time.Time
		var completedAt *time.Time
		if err := rows.Scan(&id, &connID, &status, &totalFindings, &startedAt, &completedAt); err != nil {
			continue
		}
		sc := map[string]interface{}{
			"id": id, "connection_id": connID, "status": status,
			"total_findings": totalFindings, "started_at": startedAt.Format(time.RFC3339),
		}
		if completedAt != nil {
			sc["completed_at"] = completedAt.Format(time.RFC3339)
		}
		scans = append(scans, sc)
	}
	return map[string]interface{}{"scan_runs": scans}, nil
}

func (s *EvidencePackageService) buildRemediationActions(ctx context.Context, tenantID uuid.UUID) (interface{}, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, finding_id, action_type, status, applied_at, applied_by
		FROM remediation_actions
		WHERE tenant_id = $1
		ORDER BY applied_at DESC
		LIMIT 1000`, tenantID)
	if err != nil {
		return map[string]interface{}{"remediation_actions": []interface{}{}}, nil
	}
	defer rows.Close()

	var actions []map[string]interface{}
	for rows.Next() {
		var id, findingID, actionType, status, appliedBy string
		var appliedAt time.Time
		if err := rows.Scan(&id, &findingID, &actionType, &status, &appliedAt, &appliedBy); err != nil {
			continue
		}
		actions = append(actions, map[string]interface{}{
			"id": id, "finding_id": findingID, "action_type": actionType,
			"status": status, "applied_at": appliedAt.Format(time.RFC3339), "applied_by": appliedBy,
		})
	}
	return map[string]interface{}{"remediation_actions": actions}, nil
}

func (s *EvidencePackageService) buildPIICategories(ctx context.Context, tenantID uuid.UUID) (interface{}, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT f.pii_category, COUNT(*) AS count, MAX(f.created_at) AS last_seen
		FROM findings f
		JOIN scan_runs sr ON f.scan_run_id = sr.id
		WHERE sr.tenant_id = $1 AND f.pii_category IS NOT NULL
		GROUP BY f.pii_category
		ORDER BY count DESC`, tenantID)
	if err != nil {
		return map[string]interface{}{"pii_categories": []interface{}{}}, nil
	}
	defer rows.Close()

	var categories []map[string]interface{}
	for rows.Next() {
		var cat string
		var count int
		var lastSeen time.Time
		if err := rows.Scan(&cat, &count, &lastSeen); err != nil {
			continue
		}
		categories = append(categories, map[string]interface{}{
			"pii_category": cat, "finding_count": count, "last_seen": lastSeen.Format(time.RFC3339),
		})
	}
	return map[string]interface{}{"pii_categories": categories}, nil
}

func writeZipEntry(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func writeZipJSON(zw *zip.Writer, name string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeZipEntry(zw, name, data)
}
