package workflows

import (
	"time"

	"github.com/arc-platform/backend/modules/scanning/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ScanLifecycleWorkflow orchestrates the complete scan lifecycle
// States: CREATED → RUNNING → COMPLETED | FAILED | CANCELLED
func ScanLifecycleWorkflow(ctx workflow.Context, scanID string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting scan lifecycle workflow", "scanID", scanID)

	// Configure activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    3,
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// State: CREATED → RUNNING
	err := workflow.ExecuteActivity(ctx, "TransitionScanState", scanID, "CREATED", "RUNNING").Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to transition to RUNNING", "error", err)
		RecordWorkflowFailure("ScanLifecycleWorkflow")
		return err
	}

	// Execute scan ingestion (with retry)
	var findingCount int
	err = workflow.ExecuteActivity(ctx, "IngestScanFindings", scanID).Get(ctx, &findingCount)
	if err != nil {
		logger.Error("Scan ingestion failed", "error", err)
		// State: RUNNING → FAILED
		workflow.ExecuteActivity(ctx, "TransitionScanState", scanID, "RUNNING", "FAILED")
		RecordWorkflowFailure("ScanLifecycleWorkflow")
		return err
	}

	// H21: Sync Neo4j — on failure, transition scan to DEGRADED and propagate error
	if syncErr := workflow.ExecuteActivity(ctx, "SyncToNeo4j", scanID).Get(ctx, nil); syncErr != nil {
		logger.Error("Neo4j sync failed — transitioning scan to DEGRADED", "scanID", scanID, "error", syncErr)
		workflow.ExecuteActivity(ctx, "TransitionScanState", scanID, "RUNNING", "DEGRADED")
		RecordWorkflowFailure("ScanLifecycleWorkflow")
		return syncErr
	}

	// State: RUNNING → COMPLETED
	err = workflow.ExecuteActivity(ctx, "TransitionScanState", scanID, "RUNNING", "COMPLETED").Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to transition to COMPLETED", "error", err)
		RecordWorkflowFailure("ScanLifecycleWorkflow")
		return err
	}

	logger.Info("Scan lifecycle completed successfully", "scanID", scanID, "findingCount", findingCount)
	return nil
}

// RemediationWorkflow orchestrates remediation with rollback support
func RemediationWorkflow(ctx workflow.Context, findingIDs []string, actionType string, userID string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting remediation workflow", "findingCount", len(findingIDs), "actionType", actionType)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    3,
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var executedActions []string

	// Execute remediation for each finding
	for i, findingID := range findingIDs {
		var result activities.RemediationResult
		err := workflow.ExecuteActivity(ctx, "ExecuteRemediation", findingID, actionType, userID).Get(ctx, &result)

		if err != nil {
			logger.Error("Remediation failed, rolling back", "findingID", findingID, "error", err)

			// Rollback all previously executed actions
			for _, prevActionID := range executedActions {
				rollbackErr := workflow.ExecuteActivity(ctx, "RollbackRemediation", prevActionID).Get(ctx, nil)
				if rollbackErr != nil {
					logger.Error("Rollback failed", "actionID", prevActionID, "error", rollbackErr)
				}
			}

			RecordWorkflowFailure("RemediationWorkflow")
			return err
		}

		executedActions = append(executedActions, result.ActionID)
		logger.Info("Remediation executed", "progress", i+1, "total", len(findingIDs), "actionID", result.ActionID)

		// Fix C2+H8: use workflow.Now(ctx) for determinism; pass assetID+piiType from result
		if closeErr := workflow.ExecuteActivity(ctx, "CloseExposureWindow", result.AssetID, result.PIIType, workflow.Now(ctx)).Get(ctx, nil); closeErr != nil {
			logger.Error("CloseExposureWindow failed", "assetID", result.AssetID, "error", closeErr)
		}
	}

	logger.Info("Remediation workflow completed", "actionCount", len(executedActions))
	return nil
}

// PolicyEvaluationWorkflow evaluates and executes policies for a finding
func PolicyEvaluationWorkflow(ctx workflow.Context, findingID string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting policy evaluation workflow", "findingID", findingID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 3 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2,
			InitialInterval: time.Second,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Get finding details
	var finding map[string]interface{}
	err := workflow.ExecuteActivity(ctx, "GetFinding", findingID).Get(ctx, &finding)
	if err != nil {
		logger.Error("Failed to get finding", "error", err)
		RecordWorkflowFailure("PolicyEvaluationWorkflow")
		return err
	}

	// Get active policies
	var policies []map[string]interface{}
	err = workflow.ExecuteActivity(ctx, "GetActivePolicies", "REMEDIATION").Get(ctx, &policies)
	if err != nil {
		logger.Error("Failed to get policies", "error", err)
		RecordWorkflowFailure("PolicyEvaluationWorkflow")
		return err
	}

	// Evaluate each policy
	for _, policy := range policies {
		var matches bool
		err = workflow.ExecuteActivity(ctx, "EvaluatePolicyConditions", policy, finding).Get(ctx, &matches)
		if err != nil {
			logger.Error("Policy evaluation failed", "policyID", policy["id"], "error", err)
			continue
		}

		if matches {
			logger.Info("Policy matched, executing actions", "policyID", policy["id"])

			// Execute policy actions
			err = workflow.ExecuteActivity(ctx, "ExecutePolicyActions", policy, findingID).Get(ctx, nil)
			if err != nil {
				logger.Error("Policy action execution failed", "policyID", policy["id"], "error", err)
			}
		}
	}

	logger.Info("Policy evaluation completed", "findingID", findingID)
	return nil
}
