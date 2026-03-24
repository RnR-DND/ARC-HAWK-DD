package activities

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// ScanActivities contains all scan-related Temporal activities
type ScanActivities struct {
	db    *sql.DB
	neo4j neo4j.DriverWithContext
}

// NewScanActivities creates a new ScanActivities instance
func NewScanActivities(db *sql.DB, neo4jDriver neo4j.DriverWithContext) *ScanActivities {
	return &ScanActivities{
		db:    db,
		neo4j: neo4jDriver,
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

// SyncToNeo4j synchronizes lineage to graph database
func (a *ScanActivities) SyncToNeo4j(ctx context.Context, scanID string) error {
	// TODO: Integrate with existing lineage sync logic
	// This is a placeholder for now
	session := a.neo4j.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Example: Create Scan node
	_, err := session.Run(ctx, `
		MERGE (s:Scan {id: $scanID})
		SET s.synced_at = datetime()
	`, map[string]interface{}{
		"scanID": scanID,
	})

	if err != nil {
		return fmt.Errorf("failed to sync to Neo4j: %w", err)
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

	// Record audit log
	_, err = tx.ExecContext(ctx, `
		INSERT INTO audit_logs 
		(event_type, event_time, user_id, resource_type, resource_id, action)
		VALUES ('REMEDIATION_EXECUTED', NOW(), $1, 'remediation_action', $2, $3)
	`, userID, actionID, actionType)
	if err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
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
