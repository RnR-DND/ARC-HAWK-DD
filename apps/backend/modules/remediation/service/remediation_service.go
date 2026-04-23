package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/arc-platform/backend/modules/remediation/connectors"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/google/uuid"
)

// ErrRemediationDisabled is returned when REMEDIATION_ENABLED is not set to "true".
// This is a safety gate: the connector implementations are stubs that report success
// without actually mutating source data. Enabling in production without completing
// connector work would silently mislead users about compliance status.
var ErrRemediationDisabled = errors.New("remediation is disabled (REMEDIATION_ENABLED != true); connectors are stubs and will not mutate source data")

func isRemediationEnabled() bool {
	return strings.EqualFold(os.Getenv("REMEDIATION_ENABLED"), "true")
}

// RemediationService handles remediation operations
type RemediationService struct {
	db               *sql.DB
	lineageSync      interfaces.LineageSync
	auditLogger      interfaces.AuditLogger
	connectorFactory *connectors.ConnectorFactory
}

// NewRemediationService creates a new remediation service
func NewRemediationService(db *sql.DB, lineageSync interfaces.LineageSync, auditLogger interfaces.AuditLogger) *RemediationService {
	if lineageSync == nil {
		lineageSync = &interfaces.NoOpLineageSync{}
	}
	return &RemediationService{
		db:               db,
		lineageSync:      lineageSync,
		auditLogger:      auditLogger,
		connectorFactory: &connectors.ConnectorFactory{},
	}
}

// GetDB returns the database connection
func (s *RemediationService) GetDB() *sql.DB {
	return s.db
}

// Finding represents a PII finding
type Finding struct {
	ID           string
	AssetID      string
	SystemID     string
	AssetName    string
	Location     string // Asset path/location
	AssetPath    string
	SourceSystem string
	SourceType   string
	FieldName    string
	PIIType      string
	RecordID     string
	SampleText   string
	Context      string
}

// RemediationRequest represents a remediation request
type RemediationRequest struct {
	FindingIDs []string
	ActionType string // MASK, DELETE, ENCRYPT
	UserID     string
}

// ExecuteRemediation performs remediation on source system
func (s *RemediationService) ExecuteRemediation(ctx context.Context, findingID string, actionType string, userID string) (string, error) {
	if !isRemediationEnabled() {
		log.Printf("WARN: remediation blocked by REMEDIATION_ENABLED flag (finding=%s action=%s user=%s)", findingID, actionType, userID)
		return "", ErrRemediationDisabled
	}
	// 1. Get finding details
	finding, err := s.getFinding(ctx, findingID)
	if err != nil {
		return "", fmt.Errorf("failed to get finding: %w", err)
	}

	// 2. Get source connection config
	config, err := s.getSourceConfig(ctx, finding.SourceSystem)
	if err != nil {
		return "", fmt.Errorf("failed to get source config: %w", err)
	}

	// 3. Create connector
	connector, err := s.connectorFactory.NewConnector(finding.SourceType)
	if err != nil {
		return "", fmt.Errorf("failed to create connector: %w", err)
	}
	defer connector.Close()

	// 4. Connect to source
	if err := connector.Connect(ctx, config); err != nil {
		return "", fmt.Errorf("failed to connect to source: %w", err)
	}

	// 5. Get original value (for rollback)
	originalValue, err := connector.GetOriginalValue(ctx, finding.AssetPath, finding.FieldName, finding.RecordID)
	if err != nil {
		return "", fmt.Errorf("failed to get original value: %w", err)
	}

	// 6. Create remediation action record (PENDING)
	actionID, err := s.createRemediationAction(ctx, findingID, actionType, userID, originalValue)
	if err != nil {
		return "", fmt.Errorf("failed to create remediation action: %w", err)
	}

	// 7. Update status to IN_PROGRESS
	if err := s.updateRemediationStatus(ctx, actionID, "IN_PROGRESS"); err != nil {
		return "", fmt.Errorf("failed to update status: %w", err)
	}

	// 8. Execute remediation on source system
	switch actionType {
	case "MASK":
		err = connector.Mask(ctx, finding.AssetPath, finding.FieldName, finding.RecordID)
	case "DELETE":
		err = connector.Delete(ctx, finding.AssetPath, finding.RecordID)
	case "ENCRYPT":
		err = connector.Encrypt(ctx, finding.AssetPath, finding.FieldName, finding.RecordID, "encryption-key")
	default:
		err = fmt.Errorf("unsupported action type: %s", actionType)
	}

	if err != nil {
		s.updateRemediationStatus(ctx, actionID, "FAILED")
		return "", fmt.Errorf("failed to execute remediation: %w", err)
	}

	// 9. Update status to COMPLETED
	if err := s.updateRemediationStatus(ctx, actionID, "COMPLETED"); err != nil {
		return "", fmt.Errorf("failed to update status: %w", err)
	}

	// 10. Sync asset to lineage graph (data has changed)
	if s.lineageSync.IsAvailable() {
		assetUUID, parseErr := uuid.Parse(finding.AssetID)
		if parseErr == nil {
			if err := s.lineageSync.SyncAssetToNeo4j(ctx, assetUUID); err != nil {
				// Source is mutated, Postgres is committed — cannot fully roll back.
				// Transition to COMPLETED_WITH_LINEAGE_DRIFT and alert.
				s.db.ExecContext(ctx, `
					UPDATE remediation_actions SET status='COMPLETED_WITH_LINEAGE_DRIFT' WHERE id=$1
				`, actionID)
				remediationLineageDriftTotal.Inc()
				log.Printf("ERROR: remediation %s completed but Neo4j lineage sync failed: %v", actionID, err)
				return actionID, nil // don't fail the request — PII is masked, just lineage is stale
			}
		}
	}

	// 11. Record audit log via shared interface (hash-chained)
	if s.auditLogger != nil {
		_ = s.auditLogger.Record(ctx, "REMEDIATION_EXECUTED", "remediation_action", actionID, map[string]interface{}{
			"finding_id":  findingID,
			"action_type": actionType,
			"asset_name":  finding.AssetName,
			"user_id":     userID,
		})
	}

	return actionID, nil
}

// RollbackRemediation undoes a remediation action
func (s *RemediationService) RollbackRemediation(ctx context.Context, actionID string) error {
	// 1. Get remediation action
	action, err := s.GetRemediationAction(ctx, actionID)
	if err != nil {
		return fmt.Errorf("failed to get remediation action: %w", err)
	}

	if action.Status != "COMPLETED" {
		return fmt.Errorf("can only rollback completed actions, current status: %s", action.Status)
	}

	// 2. Get finding details
	finding, err := s.getFinding(ctx, action.FindingID)
	if err != nil {
		return fmt.Errorf("failed to get finding: %w", err)
	}

	// 3. Get source config
	config, err := s.getSourceConfig(ctx, finding.SourceSystem)
	if err != nil {
		return fmt.Errorf("failed to get source config: %w", err)
	}

	// 4. Create connector
	connector, err := s.connectorFactory.NewConnector(finding.SourceType)
	if err != nil {
		return fmt.Errorf("failed to create connector: %w", err)
	}
	defer connector.Close()

	// 5. Connect to source
	if err := connector.Connect(ctx, config); err != nil {
		return fmt.Errorf("failed to connect to source: %w", err)
	}

	// 6. Restore original value
	if err := connector.RestoreValue(ctx, finding.AssetPath, finding.FieldName, finding.RecordID, action.OriginalValue); err != nil {
		return fmt.Errorf("failed to restore value: %w", err)
	}

	// 7 & 8. Update status to ROLLED_BACK and set effective_until in a single transaction.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin rollback transaction: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE remediation_actions
		SET status = 'ROLLED_BACK', effective_until = NOW()
		WHERE id = $1
	`, actionID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update status and effective_until: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback transaction: %w", err)
	}

	// 9. Reopen exposure window in Neo4j since source data is restored.
	if s.lineageSync != nil {
		assetUUID, parseErr := uuid.Parse(finding.AssetID)
		if parseErr == nil {
			if syncErr := s.lineageSync.SyncAssetToNeo4j(ctx, assetUUID); syncErr != nil {
				log.Printf("WARN: rollback %s succeeded but Neo4j re-sync failed: %v", actionID, syncErr)
			}
		}
	}

	// 10. Record audit log via shared interface (hash-chained)
	if s.auditLogger != nil {
		_ = s.auditLogger.Record(ctx, "REMEDIATION_ROLLED_BACK", "remediation_action", actionID, map[string]interface{}{
			"finding_id": action.FindingID,
		})
	}

	return nil
}

// GenerateRemediationPreview generates a preview of remediation impact
func (s *RemediationService) GenerateRemediationPreview(ctx context.Context, findingIDs []string, actionType string) (*RemediationPreview, error) {
	// Get findings details
	findings := make([]FindingPreview, 0, len(findingIDs))
	affectedAssets := make(map[string]bool)
	affectedSystems := make(map[string]bool)
	piiTypes := make(map[string]bool)

	for _, findingID := range findingIDs {
		finding, err := s.getFinding(ctx, findingID)
		if err != nil {
			return nil, fmt.Errorf("failed to get finding %s: %w", findingID, err)
		}

		// Get sample value (for preview only)
		sampleBefore := "***REDACTED***" // In production, fetch from source
		sampleAfter := s.generateSampleAfter(sampleBefore, actionType)

		findings = append(findings, FindingPreview{
			FindingID:    findingID,
			AssetName:    finding.AssetName,
			AssetPath:    finding.Location,
			PIIType:      finding.PIIType,
			FieldName:    finding.FieldName,
			SampleBefore: sampleBefore,
			SampleAfter:  sampleAfter,
		})

		affectedAssets[finding.AssetID] = true
		affectedSystems[finding.SystemID] = true
		piiTypes[finding.PIIType] = true
	}

	// Convert maps to slices
	piiTypeList := make([]string, 0, len(piiTypes))
	for piiType := range piiTypes {
		piiTypeList = append(piiTypeList, piiType)
	}

	// Generate request ID
	requestID := uuid.New().String()

	preview := &RemediationPreview{
		RequestID:  requestID,
		FindingIDs: findingIDs,
		ActionType: actionType,
		Impact: RemediationImpact{
			TotalFindings:    len(findingIDs),
			AffectedAssets:   len(affectedAssets),
			AffectedSystems:  len(affectedSystems),
			PIITypes:         piiTypeList,
			EstimatedRecords: len(findingIDs), // Simplified estimate
		},
		Findings:             findings,
		RequiresConfirmation: true,
	}

	// Persist preview for single-use execution via ExecuteRemediationRequest.
	previewJSON, err := json.Marshal(preview)
	if err == nil {
		_, storeErr := s.db.ExecContext(ctx, `
			INSERT INTO remediation_previews (id, preview_data, expires_at)
			VALUES ($1, $2, NOW() + INTERVAL '1 hour')
		`, requestID, previewJSON)
		if storeErr != nil {
			log.Printf("WARNING: Failed to store remediation preview %s: %v", requestID, storeErr)
		}
	}

	return preview, nil
}

// ExecuteRemediationRequest executes a previously previewed remediation request.
// The preview is single-use — it is deleted after successful execution.
func (s *RemediationService) ExecuteRemediationRequest(ctx context.Context, requestID string, userID string) (*RemediationResult, error) {
	if !isRemediationEnabled() {
		log.Printf("WARN: batch remediation blocked by REMEDIATION_ENABLED flag (request=%s user=%s)", requestID, userID)
		return nil, ErrRemediationDisabled
	}
	// 1. Retrieve and validate the stored preview.
	var previewJSON []byte
	var expiresAt time.Time
	err := s.db.QueryRowContext(ctx, `
		SELECT preview_data, expires_at FROM remediation_previews WHERE id = $1
	`, requestID).Scan(&previewJSON, &expiresAt)
	if err != nil {
		return nil, fmt.Errorf("remediation preview not found (id=%s): %w", requestID, err)
	}

	if time.Now().After(expiresAt) {
		// Clean up expired preview.
		_, _ = s.db.ExecContext(ctx, `DELETE FROM remediation_previews WHERE id = $1`, requestID)
		return nil, fmt.Errorf("remediation preview %s has expired", requestID)
	}

	var preview RemediationPreview
	if err := json.Unmarshal(previewJSON, &preview); err != nil {
		return nil, fmt.Errorf("failed to decode remediation preview: %w", err)
	}

	// 2. Execute remediation for each finding in the preview (saga pattern).
	result := &RemediationResult{
		RequestID:  requestID,
		ExecutedBy: userID,
		ExecutedAt: time.Now().UTC().Format(time.RFC3339),
	}

	var completedActionIDs []string
	sagaFailed := false

	for _, findingID := range preview.FindingIDs {
		actionID, err := s.ExecuteRemediation(ctx, findingID, preview.ActionType, userID)
		if err != nil {
			result.FailureCount++
			result.FailedFindingIDs = append(result.FailedFindingIDs, findingID)
			log.Printf("WARNING: Remediation failed for finding %s: %v — triggering saga rollback for %d prior actions", findingID, err, len(completedActionIDs))
			remediationActionsFailedTotal.WithLabelValues(preview.ActionType, "execution_error").Inc()

			// Compensating rollback for all previously-succeeded actions.
			for _, previousActionID := range completedActionIDs {
				if rollbackErr := s.RollbackRemediation(ctx, previousActionID); rollbackErr != nil {
					log.Printf("WARN: saga rollback failed for action %s: %v", previousActionID, rollbackErr)
				}
			}
			sagaFailed = true
			break
		}
		result.SuccessCount++
		result.ActionID = actionID // last successful action ID for reference
		completedActionIDs = append(completedActionIDs, actionID)
	}

	if sagaFailed {
		result.Status = "FAILED"
		result.SuccessCount = 0
	} else {
		result.Status = "COMPLETED"
	}

	// 3. Delete the preview — single-use only.
	if _, err := s.db.ExecContext(ctx, `DELETE FROM remediation_previews WHERE id = $1`, requestID); err != nil {
		log.Printf("WARNING: Failed to delete remediation preview %s after execution: %v", requestID, err)
	}

	return result, nil
}

// Helper function to generate sample after value
func (s *RemediationService) generateSampleAfter(sampleBefore string, actionType string) string {
	switch actionType {
	case "MASK":
		return "***REDACTED***"
	case "DELETE":
		return "[DELETED]"
	case "ENCRYPT":
		return "[ENCRYPTED]"
	default:
		return sampleBefore
	}
}

// RemediationPreview represents a preview of remediation impact
type RemediationPreview struct {
	RequestID            string            `json:"request_id"`
	FindingIDs           []string          `json:"finding_ids"`
	ActionType           string            `json:"action_type"`
	Impact               RemediationImpact `json:"impact"`
	Findings             []FindingPreview  `json:"findings"`
	RequiresConfirmation bool              `json:"requires_confirmation"`
}

// RemediationImpact represents the impact of remediation
type RemediationImpact struct {
	TotalFindings    int      `json:"total_findings"`
	AffectedAssets   int      `json:"affected_assets"`
	AffectedSystems  int      `json:"affected_systems"`
	PIITypes         []string `json:"pii_types"`
	EstimatedRecords int      `json:"estimated_records"`
}

// FindingPreview represents a finding in the preview
type FindingPreview struct {
	FindingID    string `json:"finding_id"`
	AssetName    string `json:"asset_name"`
	AssetPath    string `json:"asset_path"`
	PIIType      string `json:"pii_type"`
	FieldName    string `json:"field_name"`
	SampleBefore string `json:"sample_before"`
	SampleAfter  string `json:"sample_after"`
}

// RemediationResult represents the result of remediation execution
type RemediationResult struct {
	RequestID        string   `json:"request_id"`
	ExecutedBy       string   `json:"executed_by"`
	ExecutedAt       string   `json:"executed_at"`
	SuccessCount     int      `json:"success_count"`
	FailureCount     int      `json:"failure_count"`
	FailedFindingIDs []string `json:"failed_finding_ids,omitempty"`
	ActionID         string   `json:"action_id,omitempty"`
	FindingID        string   `json:"finding_id,omitempty"`
	Status           string   `json:"status,omitempty"`
	OriginalValue    string   `json:"original_value,omitempty"`
	Error            string   `json:"error,omitempty"`
}

// Helper functions

func (s *RemediationService) getFinding(ctx context.Context, findingID string) (*Finding, error) {
	query := `
		SELECT f.id, f.asset_id, a.name, a.path, sp.name as source_system, sp.source_type,
		       f.field_name, f.pii_type, f.record_id, f.sample_text, f.context
		FROM findings f
		JOIN assets a ON f.asset_id = a.id
		JOIN source_profiles sp ON a.source_profile_id = sp.id
		WHERE f.id = $1
	`

	var finding Finding
	err := s.db.QueryRowContext(ctx, query, findingID).Scan(
		&finding.ID, &finding.AssetID, &finding.AssetName, &finding.AssetPath,
		&finding.SourceSystem, &finding.SourceType, &finding.FieldName,
		&finding.PIIType, &finding.RecordID, &finding.SampleText, &finding.Context,
	)
	if err != nil {
		return nil, err
	}

	return &finding, nil
}

func (s *RemediationService) getSourceConfig(ctx context.Context, sourceName string) (map[string]interface{}, error) {
	var configJSON string
	err := s.db.QueryRowContext(ctx, `
		SELECT connection_config FROM source_profiles WHERE name = $1
	`, sourceName).Scan(&configJSON)
	if err != nil {
		return nil, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, err
	}

	return config, nil
}

func (s *RemediationService) createRemediationAction(ctx context.Context, findingID string, actionType string, userID string, originalValue string) (string, error) {
	actionID := uuid.New().String()

	metadata := map[string]interface{}{
		"original_value": originalValue,
	}
	metadataJSON, _ := json.Marshal(metadata)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO remediation_actions 
		(id, finding_id, action_type, executed_by, executed_at, effective_from, status, metadata)
		VALUES ($1, $2, $3, $4, NOW(), NOW(), 'PENDING', $5)
	`, actionID, findingID, actionType, userID, metadataJSON)

	return actionID, err
}

func (s *RemediationService) updateRemediationStatus(ctx context.Context, actionID string, status string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE remediation_actions 
		SET status = $1
		WHERE id = $2
	`, status, actionID)
	return err
}

type RemediationAction struct {
	ID            string
	FindingID     string
	ActionType    string
	ExecutedBy    string
	ExecutedAt    time.Time
	Status        string
	OriginalValue string
	// Enriched fields from JOIN with findings + assets
	AssetName string
	AssetPath string
	PIIType   string
	RiskLevel string
}

func (s *RemediationService) GetRemediationActions(ctx context.Context, findingID string) ([]*RemediationAction, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, finding_id, action_type, executed_by, executed_at, status
		FROM remediation_actions
		WHERE finding_id = $1
		ORDER BY executed_at DESC
	`, findingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []*RemediationAction
	for rows.Next() {
		var action RemediationAction
		err := rows.Scan(&action.ID, &action.FindingID, &action.ActionType, &action.ExecutedBy, &action.ExecutedAt, &action.Status)
		if err != nil {
			return nil, err
		}
		actions = append(actions, &action)
	}

	return actions, nil
}

// GetAllRemediationActions retrieves all remediation actions with pagination and filtering.
// It JOINs with findings and assets to enrich each record with asset_name, asset_path,
// pii_type (pattern_name), and risk_level (severity) so the frontend can display
// meaningful context without additional round-trips.
func (s *RemediationService) GetAllRemediationActions(ctx context.Context, limit, offset int, actionFilter string) ([]*RemediationAction, int, error) {
	// Enriched query: LEFT JOIN findings + assets to get context fields.
	// COALESCE handles cases where a finding or asset has been deleted.
	query := `
		SELECT ra.id, ra.finding_id, ra.action_type, ra.executed_by, ra.executed_at, ra.status,
		       COALESCE(a.name, '')       AS asset_name,
		       COALESCE(a.path, '')       AS asset_path,
		       COALESCE(f.pattern_name, '') AS pii_type,
		       COALESCE(f.severity, 'Medium') AS risk_level
		FROM remediation_actions ra
		LEFT JOIN findings f ON f.id = ra.finding_id
		LEFT JOIN assets a   ON a.id = f.asset_id
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM remediation_actions WHERE 1=1`

	args := []interface{}{}
	argCount := 1

	// Add filter
	if actionFilter != "" && actionFilter != "ALL" {
		filterClause := fmt.Sprintf(" AND ra.action_type = $%d", argCount)
		query += filterClause
		countQuery += fmt.Sprintf(" AND action_type = $%d", argCount)
		args = append(args, actionFilter)
		argCount++
	}

	// Add ordering and pagination
	query += fmt.Sprintf(" ORDER BY ra.executed_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	// Execute count query (uses only filter args, not limit/offset)
	var total int
	countArgs := args[:len(args)-2]
	err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count remediation actions: %w", err)
	}

	// Execute enriched data query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list remediation actions: %w", err)
	}
	defer rows.Close()

	var actions []*RemediationAction
	for rows.Next() {
		var action RemediationAction
		err := rows.Scan(
			&action.ID, &action.FindingID, &action.ActionType,
			&action.ExecutedBy, &action.ExecutedAt, &action.Status,
			&action.AssetName, &action.AssetPath, &action.PIIType, &action.RiskLevel,
		)
		if err != nil {
			return nil, 0, err
		}

		actions = append(actions, &action)
	}

	return actions, total, nil
}

func (s *RemediationService) GetRemediationHistory(ctx context.Context, assetID string) ([]*RemediationAction, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ra.id, ra.finding_id, ra.action_type, ra.executed_by, ra.executed_at, ra.status
		FROM remediation_actions ra
		JOIN findings f ON ra.finding_id = f.id
		WHERE f.asset_id = $1
		ORDER BY ra.executed_at DESC
	`, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []*RemediationAction
	for rows.Next() {
		var action RemediationAction
		err := rows.Scan(&action.ID, &action.FindingID, &action.ActionType, &action.ExecutedBy, &action.ExecutedAt, &action.Status)
		if err != nil {
			return nil, err
		}
		actions = append(actions, &action)
	}

	return actions, nil
}

func (s *RemediationService) GetPIIPreview(ctx context.Context, findingID string) (map[string]interface{}, error) {
	var finding struct {
		SampleText string
		PIIType    string
	}
	err := s.db.QueryRowContext(ctx, `
		SELECT sample_text, pii_type
		FROM findings
		WHERE id = $1
	`, findingID).Scan(&finding.SampleText, &finding.PIIType)
	if err != nil {
		return nil, err
	}

	// Simple masking for preview
	maskedText := s.maskText(finding.SampleText, finding.PIIType)

	return map[string]interface{}{
		"finding_id":    findingID,
		"original_text": finding.SampleText,
		"masked_text":   maskedText,
		"pii_type":      finding.PIIType,
	}, nil
}

func (s *RemediationService) maskText(text, piiType string) string {
	// Simple masking logic
	switch piiType {
	case "EMAIL":
		return strings.ReplaceAll(text, "@", "[AT]")
	case "PHONE":
		return strings.Repeat("*", len(text))
	case "CREDIT_CARD":
		if len(text) > 4 {
			return strings.Repeat("*", len(text)-4) + text[len(text)-4:]
		}
		return strings.Repeat("*", len(text))
	default:
		return strings.Repeat("*", len(text))
	}
}

func (s *RemediationService) GetRemediationAction(ctx context.Context, actionID string) (*RemediationAction, error) {
	var action RemediationAction
	var metadataJSON string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, finding_id, action_type, executed_by, executed_at, status, metadata
		FROM remediation_actions
		WHERE id = $1
	`, actionID).Scan(
		&action.ID, &action.FindingID, &action.ActionType,
		&action.ExecutedBy, &action.ExecutedAt, &action.Status, &metadataJSON,
	)
	if err != nil {
		return nil, err
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err == nil {
		if val, ok := metadata["original_value"].(string); ok {
			action.OriginalValue = val
		}
	}

	return &action, nil
}
