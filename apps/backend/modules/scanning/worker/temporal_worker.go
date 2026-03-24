package worker

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/arc-platform/backend/modules/scanning/activities"
	"github.com/arc-platform/backend/modules/scanning/workflows"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// TemporalWorker manages the Temporal worker lifecycle
type TemporalWorker struct {
	client client.Client
	worker worker.Worker
}

// NewTemporalWorker creates and starts a new Temporal worker
func NewTemporalWorker(temporalAddress string, db *sql.DB, neo4jDriver neo4j.DriverWithContext) (*TemporalWorker, error) {
	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: temporalAddress,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}

	// Create worker
	w := worker.New(c, "arc-hawk-task-queue", worker.Options{})

	// Register workflows
	w.RegisterWorkflow(workflows.ScanLifecycleWorkflow)
	w.RegisterWorkflow(workflows.RemediationWorkflow)
	w.RegisterWorkflow(workflows.PolicyEvaluationWorkflow)

	// Register activities
	scanActivities := activities.NewScanActivities(db, neo4jDriver)
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
