package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/arc-platform/backend/modules/shared/infrastructure/audit"
	"github.com/google/uuid"
)

// ObligationRegressionDetector compares scan PII categories against historical
// baseline and emits audit events when new categories are discovered.
type ObligationRegressionDetector struct {
	db     *sql.DB
	logger *audit.LedgerLogger
}

// NewObligationRegressionDetector creates a new detector.
func NewObligationRegressionDetector(db *sql.DB, logger *audit.LedgerLogger) *ObligationRegressionDetector {
	return &ObligationRegressionDetector{db: db, logger: logger}
}

// DetectRegressions compares currentScanID findings against all prior completed
// scans for the tenant. Emits EventPIIDiscovered and inserts an
// obligation_regressions row for each brand-new PII category.
func (d *ObligationRegressionDetector) DetectRegressions(ctx context.Context, tenantID uuid.UUID, currentScanID string) error {
	currentCategories, err := d.getScanPIICategories(ctx, currentScanID)
	if err != nil {
		return fmt.Errorf("get current scan categories: %w", err)
	}

	previousCategories, err := d.getPreviousTenantPIICategories(ctx, tenantID, currentScanID)
	if err != nil {
		return fmt.Errorf("get previous categories: %w", err)
	}

	for cat := range currentCategories {
		if previousCategories[cat] {
			continue
		}
		// New PII category — log to audit ledger (best-effort)
		_ = d.logger.Log(ctx, audit.LogEntry{
			TenantID:     tenantID,
			EventType:    audit.EventPIIDiscovered,
			ResourceID:   currentScanID,
			ResourceType: "scan_run",
			Payload: map[string]interface{}{
				"new_pii_category": cat,
				"regression_type":  "new_category_discovered",
				"recommendation":   fmt.Sprintf("Review DPDP obligations for newly discovered PII category: %s", cat),
			},
		})

		// Upsert regression record
		if _, err := d.db.ExecContext(ctx, `
			INSERT INTO obligation_regressions (tenant_id, scan_id, pii_category, detected_at)
			VALUES ($1, $2, $3, NOW())
			ON CONFLICT (tenant_id, pii_category) DO UPDATE SET
				scan_id = EXCLUDED.scan_id,
				detected_at = NOW()`,
			tenantID, currentScanID, cat,
		); err != nil {
			log.Printf("WARN: obligation regression upsert for tenant %s category %s: %v", tenantID, cat, err)
		}
	}
	return nil
}

func (d *ObligationRegressionDetector) getScanPIICategories(ctx context.Context, scanID string) (map[string]bool, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT DISTINCT pii_category FROM findings
		WHERE scan_run_id = $1 AND pii_category IS NOT NULL`, scanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cats := map[string]bool{}
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err == nil {
			cats[cat] = true
		}
	}
	return cats, rows.Err()
}

func (d *ObligationRegressionDetector) getPreviousTenantPIICategories(ctx context.Context, tenantID uuid.UUID, excludeScanID string) (map[string]bool, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT DISTINCT f.pii_category
		FROM findings f
		JOIN scan_runs sr ON f.scan_run_id = sr.id
		WHERE sr.tenant_id = $1
		  AND sr.id != $2::uuid
		  AND f.pii_category IS NOT NULL
		  AND sr.status = 'completed'`, tenantID, excludeScanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cats := map[string]bool{}
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err == nil {
			cats[cat] = true
		}
	}
	return cats, rows.Err()
}
