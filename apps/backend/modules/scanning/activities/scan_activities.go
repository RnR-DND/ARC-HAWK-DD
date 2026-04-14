package activities

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
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

// IngestScanFindings reports the number of findings persisted for a completed scan run.
// Full ingestion is performed by IngestionService.IngestScan / IngestSDKVerified; this
// activity is called after ingestion completes to confirm row count and log progress.
func (a *ScanActivities) IngestScanFindings(ctx context.Context, scanID string) (int, error) {
	var count int
	err := a.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM findings WHERE scan_run_id = $1
	`, scanID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count findings for scan %s: %w", scanID, err)
	}

	log.Printf("INFO: IngestScanFindings — scan %s has %d persisted findings", scanID, count)

	if a.auditLogger != nil {
		_ = a.auditLogger.Record(ctx, "SCAN_FINDINGS_INGESTED", "scan_run", scanID, map[string]interface{}{
			"finding_count": count,
		})
	}

	return count, nil
}

// SyncToNeo4j synchronizes lineage for all assets affected by the scan.
// L3: Returns an error on sync failure so Temporal can retry/alert instead of silently dropping lineage.
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
		// L3: Propagate the error from each asset sync so Temporal retries the whole activity
		if syncErr := a.lineageSync.SyncAssetToNeo4j(ctx, assetID); syncErr != nil {
			log.Printf("WARN: lineage sync failed for asset %s (scan %s): %v", assetIDStr, scanID, syncErr)
			failedAssets = append(failedAssets, assetIDStr)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("row iteration error for scan %s: %w", scanID, err)
	}
	if len(failedAssets) > 0 {
		// L3: Return error so Temporal retries/alerts; lineage may be incomplete
		return fmt.Errorf("neo4j sync failed (lineage may be incomplete): %d assets failed in scan %s: %v",
			len(failedAssets), scanID, failedAssets)
	}
	return nil
}

// CloseExposureWindow closes the exposure window for an (assetID, piiType) pair in Neo4j.
// H5: Anchors on (assetID, piiType) instead of the now-removed finding_id edge property.
func (a *ScanActivities) CloseExposureWindow(ctx context.Context, assetID, piiType string, closedAt time.Time) error {
	session := a.neo4j.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// H5: Match EXPOSES edge by (assetID, piiType) — no finding_id on edges in 3-level contract
	_, err := session.Run(ctx, `
		MATCH (a:Asset {id: $assetID})-[e:EXPOSES]->(p:PII_Category)
		WHERE (p.name = $piiType OR p.pii_type = $piiType) AND e.until IS NULL
		SET e.until = $closedAt,
		    e.closed_at = datetime($closedAtISO),
		    e.exposure_duration_hours = CASE
		        WHEN e.since IS NOT NULL
		        THEN duration.between(e.since, datetime($closedAtISO)).hours
		        ELSE null
		    END
	`, map[string]interface{}{
		"assetID":     assetID,
		"piiType":     piiType,
		"closedAt":    closedAt,
		"closedAtISO": closedAt.UTC().Format(time.RFC3339),
	})

	if err != nil {
		return fmt.Errorf("failed to close exposure window for asset %s / piiType %s: %w", assetID, piiType, err)
	}

	return nil
}

// RemediationResult is the structured result returned by ExecuteRemediation.
type RemediationResult struct {
	ActionID string `json:"action_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

// ExecuteRemediation queues and records a remediation action for a finding.
// Actual execution on the source system is delegated to remediation_service.go.
func (a *ScanActivities) ExecuteRemediation(ctx context.Context, findingID string, actionType string, userID string) (RemediationResult, error) {
	log.Printf("INFO: ExecuteRemediation — queuing %s for finding %s by user %s", actionType, findingID, userID)

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return RemediationResult{}, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create remediation action record
	actionID := uuid.New().String()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO remediation_actions
		(id, finding_id, action_type, executed_by, executed_at, effective_from, status)
		VALUES ($1, $2, $3, $4, NOW(), NOW(), 'PENDING')
	`, actionID, findingID, actionType, userID)
	if err != nil {
		tx.Rollback()
		return RemediationResult{}, fmt.Errorf("failed to create remediation action: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return RemediationResult{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Record audit log via shared interface (outside transaction — non-blocking)
	if a.auditLogger != nil {
		_ = a.auditLogger.Record(ctx, "REMEDIATION_QUEUED", "remediation_action", actionID, map[string]interface{}{
			"finding_id":  findingID,
			"action_type": actionType,
			"user_id":     userID,
		})
	}

	log.Printf("INFO: ExecuteRemediation — action %s queued for finding %s", actionID, findingID)
	return RemediationResult{
		ActionID: actionID,
		Status:   "pending",
		Message:  "remediation queued",
	}, nil
}

// RollbackRemediation undoes a remediation action and logs the rollback.
func (a *ScanActivities) RollbackRemediation(ctx context.Context, actionID string) error {
	log.Printf("INFO: RollbackRemediation — rolling back action %s", actionID)

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

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	if a.auditLogger != nil {
		_ = a.auditLogger.Record(ctx, "REMEDIATION_ROLLED_BACK", "remediation_action", actionID, map[string]interface{}{
			"action_id": actionID,
		})
	}

	log.Printf("INFO: RollbackRemediation — action %s marked ROLLED_BACK", actionID)
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

// StreamMessage represents a message from a Redis stream.
type StreamMessage struct {
	ID      string            `json:"id"`
	Payload map[string]string `json:"payload"`
}

// StreamCheckpoint holds the last-processed stream position.
type StreamCheckpoint struct {
	QueueName   string `json:"queue_name"`
	LastID      string `json:"last_id"`
	WindowCount int    `json:"window_count"`
}

// RunStreamingWindowActivity reads a batch of messages from a Redis stream.
// Temporal calls this once per micro-batch window.
func (a *ScanActivities) RunStreamingWindowActivity(ctx context.Context, queueName string, windowSec int) ([]StreamMessage, error) {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer rdb.Close()

	// XREAD with a short block timeout equal to the window duration
	deadline := time.Duration(windowSec) * time.Second
	streams, err := rdb.XRead(ctx, &redis.XReadArgs{
		Streams: []string{queueName, "0"},
		Count:   500,
		Block:   deadline,
	}).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("redis XREAD error on %s: %w", queueName, err)
	}

	var msgs []StreamMessage
	for _, stream := range streams {
		for _, msg := range stream.Messages {
			payload := make(map[string]string)
			for k, v := range msg.Values {
				if s, ok := v.(string); ok {
					payload[k] = s
				}
			}
			msgs = append(msgs, StreamMessage{
				ID:      msg.ID,
				Payload: payload,
			})
		}
	}
	log.Printf("INFO: streaming window read %d messages from %s", len(msgs), queueName)
	return msgs, nil
}

// IngestStreamingFindings ingests a batch of streaming messages via the existing ingestion service.
func (a *ScanActivities) IngestStreamingFindings(ctx context.Context, msgs []StreamMessage) (int, error) {
	if len(msgs) == 0 {
		return 0, nil
	}
	ingested := 0
	for _, msg := range msgs {
		scanID, ok := msg.Payload["scan_id"]
		if !ok || scanID == "" {
			log.Printf("WARN: streaming message %s missing scan_id, skipping", msg.ID)
			continue
		}
		// Record acknowledgement — the actual ingest was already done by the scanner
		// posting to /ingest-verified. Here we count and log for Temporal tracking.
		log.Printf("INFO: streaming ack scan_id=%s msg_id=%s", scanID, msg.ID)
		ingested++
	}
	return ingested, nil
}

// PersistStreamingCheckpoints saves the last-processed stream position to Redis.
func (a *ScanActivities) PersistStreamingCheckpoints(ctx context.Context, cp StreamCheckpoint) error {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer rdb.Close()

	key := fmt.Sprintf("arc-hawk:stream:checkpoint:%s", cp.QueueName)
	if err := rdb.HSet(ctx, key,
		"last_id", cp.LastID,
		"window_count", cp.WindowCount,
		"updated_at", time.Now().UTC().Format(time.RFC3339),
	).Err(); err != nil {
		return fmt.Errorf("failed to persist checkpoint for %s: %w", cp.QueueName, err)
	}
	log.Printf("INFO: persisted streaming checkpoint for %s: last_id=%s", cp.QueueName, cp.LastID)
	return nil
}
