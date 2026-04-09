package activities

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// ScanActivities contains all scan-related Temporal activities
type ScanActivities struct {
	db          *sql.DB
	neo4j       neo4j.DriverWithContext
	lineageSync interfaces.LineageSync
	auditLogger interfaces.AuditLogger
}

// NewScanActivities creates a new ScanActivities instance
func NewScanActivities(db *sql.DB, neo4jDriver neo4j.DriverWithContext, lineageSync interfaces.LineageSync, auditLogger interfaces.AuditLogger) *ScanActivities {
	if lineageSync == nil {
		lineageSync = &interfaces.NoOpLineageSync{}
	}
	return &ScanActivities{
		db:          db,
		neo4j:       neo4jDriver,
		lineageSync: lineageSync,
		auditLogger: auditLogger,
	}
}

// TransitionScanState records state transition in database
func (a *ScanActivities) TransitionScanState(ctx context.Context, scanID string, fromState string, toState string) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Update scan_runs status
	result, err := tx.ExecContext(ctx, `
		UPDATE scan_runs 
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND status = $3
	`, toState, scanID, fromState)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update scan status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("scan not found or already in different state: scanID=%s, expectedState=%s", scanID, fromState)
	}

	// Record state transition
	_, err = tx.ExecContext(ctx, `
		INSERT INTO scan_state_transitions 
		(scan_run_id, from_state, to_state, transitioned_at)
		VALUES ($1, $2, $3, NOW())
	`, scanID, fromState, toState)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record state transition: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// IngestScanFindings processes findings from scanner
// This will integrate with existing ingestion logic
func (a *ScanActivities) IngestScanFindings(ctx context.Context, scanID string) (int, error) {
	// TODO: Integrate with existing ingestion_service.go logic
	// For now, return count of findings for this scan
	var count int
	err := a.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM findings WHERE scan_run_id = $1
	`, scanID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count findings: %w", err)
	}

	return count, nil
}

// SyncToNeo4j synchronizes lineage for all assets affected by the scan.
func (a *ScanActivities) SyncToNeo4j(ctx context.Context, scanID string) error {
	if !a.lineageSync.IsAvailable() {
		log.Printf("INFO: lineage sync not configured — skipping Neo4j sync for scan %s", scanID)
		return nil
	}

	// Collect the distinct asset IDs that have findings from this scan.
	rows, err := a.db.QueryContext(ctx,
		`SELECT DISTINCT asset_id::text FROM findings WHERE scan_run_id = $1 AND asset_id IS NOT NULL`,
		scanID,
	)
	if err != nil {
		return fmt.Errorf("failed to query assets for scan %s: %w", scanID, err)
	}
	defer rows.Close()

	var failedAssets []string
	for rows.Next() {
		var assetIDStr string
		if err := rows.Scan(&assetIDStr); err != nil {
			continue
		}
		assetID, err := uuid.Parse(assetIDStr)
		if err != nil {
			continue
		}
		if syncErr := a.lineageSync.SyncAssetToNeo4j(ctx, assetID); syncErr != nil {
			log.Printf("WARN: lineage sync failed for asset %s (scan %s): %v", assetIDStr, scanID, syncErr)
			failedAssets = append(failedAssets, assetIDStr)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("row iteration error for scan %s: %w", scanID, err)
	}
	if len(failedAssets) > 0 {
		return fmt.Errorf("lineage sync failed for %d assets in scan %s: %v", len(failedAssets), scanID, failedAssets)
	}
	return nil
}

// CloseExposureWindow closes the exposure window for a finding in Neo4j
func (a *ScanActivities) CloseExposureWindow(ctx context.Context, findingID string, closedAt time.Time) error {
	session := a.neo4j.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Update EXPOSES edge to set 'until' timestamp
	_, err := session.Run(ctx, `
		MATCH (a:Asset)-[e:EXPOSES]->(p:PII_Category)
		WHERE e.finding_id = $findingID AND e.until IS NULL
		SET e.until = $closedAt
	`, map[string]interface{}{
		"findingID": findingID,
		"closedAt":  closedAt,
	})

	if err != nil {
		return fmt.Errorf("failed to close exposure window: %w", err)
	}

	return nil
}

// ExecuteRemediation performs remediation action
func (a *ScanActivities) ExecuteRemediation(ctx context.Context, findingID string, actionType string, userID string) (string, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create remediation action record
	actionID := uuid.New().String()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO remediation_actions 
		(id, finding_id, action_type, executed_by, executed_at, effective_from, status)
		VALUES ($1, $2, $3, $4, NOW(), NOW(), 'IN_PROGRESS')
	`, actionID, findingID, actionType, userID)
	if err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to create remediation action: %w", err)
	}

	// TODO: Execute actual remediation on source system
	// This will be implemented in remediation_service.go

	// Update status to COMPLETED
	_, err = tx.ExecContext(ctx, `
		UPDATE remediation_actions 
		SET status = 'COMPLETED'
		WHERE id = $1
	`, actionID)
	if err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to update remediation status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Record audit log via shared interface (outside transaction — non-blocking)
	if a.auditLogger != nil {
		_ = a.auditLogger.Record(ctx, "REMEDIATION_EXECUTED", "remediation_action", actionID, map[string]interface{}{
			"finding_id":  findingID,
			"action_type": actionType,
			"user_id":     userID,
		})
	}

	return actionID, nil
}

// RollbackRemediation undoes a remediation action
func (a *ScanActivities) RollbackRemediation(ctx context.Context, actionID string) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Update action status to ROLLED_BACK
	_, err = tx.ExecContext(ctx, `
		UPDATE remediation_actions 
		SET status = 'ROLLED_BACK', effective_until = NOW()
		WHERE id = $1
	`, actionID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update remediation status: %w", err)
	}

	// TODO: Execute actual rollback on source system

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetFinding retrieves finding details
func (a *ScanActivities) GetFinding(ctx context.Context, findingID string) (map[string]interface{}, error) {
	var finding map[string]interface{}
	// TODO: Implement finding retrieval
	return finding, nil
}

// GetActivePolicies retrieves active policies of a specific type
func (a *ScanActivities) GetActivePolicies(ctx context.Context, policyType string) ([]map[string]interface{}, error) {
	var policies []map[string]interface{}
	// TODO: Implement policy retrieval
	return policies, nil
}

// EvaluatePolicyConditions evaluates if a policy matches a finding
func (a *ScanActivities) EvaluatePolicyConditions(ctx context.Context, policy map[string]interface{}, finding map[string]interface{}) (bool, error) {
	// TODO: Implement policy condition evaluation
	return false, nil
}

// ExecutePolicyActions executes actions defined in a policy
func (a *ScanActivities) ExecutePolicyActions(ctx context.Context, policy map[string]interface{}, findingID string) error {
	// TODO: Implement policy action execution
	return nil
}
