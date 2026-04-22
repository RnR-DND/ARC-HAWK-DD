package worker

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/arc-platform/backend/modules/scanning/activities"
	"github.com/arc-platform/backend/modules/scanning/workflows"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// envInt reads an env var as int, falling back to def if unset or unparseable.
func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// TemporalWorker manages the Temporal worker lifecycle
type TemporalWorker struct {
	client client.Client
	worker worker.Worker
}

// NewTemporalWorker creates and starts a new Temporal worker
func NewTemporalWorker(temporalAddress string, db *sql.DB, neo4jDriver neo4j.DriverWithContext, lineageSync interfaces.LineageSync, auditLogger interfaces.AuditLogger) (*TemporalWorker, error) {
	// Create Temporal client. TLS is enabled when TEMPORAL_TLS_ENABLED=true
	// (required in release mode — see validateRequiredEnvVars in main.go).
	opts := client.Options{HostPort: temporalAddress}
	if strings.EqualFold(os.Getenv("TEMPORAL_TLS_ENABLED"), "true") {
		opts.ConnectionOptions = client.ConnectionOptions{
			TLS: &tls.Config{MinVersion: tls.VersionTLS12},
		}
		log.Println("Temporal: TLS enabled")
	}
	c, err := client.Dial(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}

	// Create worker with explicit concurrency caps. Defaults override the
	// Temporal SDK's implicit values (which are low, single-activity) and
	// prevent stall under parallel scan load. Tune via env:
	//   TEMPORAL_MAX_ACTIVITIES, TEMPORAL_MAX_WORKFLOW_TASKS,
	//   TEMPORAL_MAX_LOCAL_ACTIVITIES
	w := worker.New(c, "arc-hawk-task-queue", worker.Options{
		MaxConcurrentActivityExecutionSize:     envInt("TEMPORAL_MAX_ACTIVITIES", 100),
		MaxConcurrentWorkflowTaskExecutionSize: envInt("TEMPORAL_MAX_WORKFLOW_TASKS", 50),
		MaxConcurrentLocalActivityExecutionSize: envInt("TEMPORAL_MAX_LOCAL_ACTIVITIES", 50),
	})

	// Register workflows
	w.RegisterWorkflow(workflows.ScanLifecycleWorkflow)
	w.RegisterWorkflow(workflows.RemediationWorkflow)
	w.RegisterWorkflow(workflows.PolicyEvaluationWorkflow)
	w.RegisterWorkflow(workflows.StreamingSupervisorWorkflow)

	// Register activities
	scanActivities := activities.NewScanActivities(db, neo4jDriver, lineageSync, auditLogger)
	w.RegisterActivity(scanActivities.TransitionScanState)
	w.RegisterActivity(scanActivities.IngestScanFindings)
	w.RegisterActivity(scanActivities.SyncToNeo4j)
	w.RegisterActivity(scanActivities.CloseExposureWindow)
	w.RegisterActivity(scanActivities.ExecuteRemediation)
	w.RegisterActivity(scanActivities.RollbackRemediation)
	w.RegisterActivity(scanActivities.GetFinding)
	w.RegisterActivity(scanActivities.GetActivePolicies)
	w.RegisterActivity(scanActivities.EvaluatePolicyConditions)
	w.RegisterActivity(scanActivities.ExecutePolicyActions)
	w.RegisterActivity(scanActivities.RunStreamingWindowActivity)
	w.RegisterActivity(scanActivities.IngestStreamingFindings)
	w.RegisterActivity(scanActivities.PersistStreamingCheckpoints)

	return &TemporalWorker{
		client: c,
		worker: w,
	}, nil
}

// Start starts the Temporal worker
func (tw *TemporalWorker) Start() error {
	log.Println("Starting Temporal worker...")
	return tw.worker.Run(worker.InterruptCh())
}

// Stop stops the Temporal worker
func (tw *TemporalWorker) Stop() {
	log.Println("Stopping Temporal worker...")
	tw.worker.Stop()
	tw.client.Close()
}

// GetClient returns the Temporal client for workflow execution
func (tw *TemporalWorker) GetClient() client.Client {
	return tw.client
}
