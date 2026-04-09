package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/arc/hawk-agent/internal/buffer"
	"github.com/arc/hawk-agent/internal/config"
	"github.com/arc/hawk-agent/internal/connectors"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Orchestrator manages scan scheduling and execution.
type Orchestrator struct {
	cfg       *config.Config
	queue     *buffer.LocalQueue
	connector *connectors.ScannerConnector
	logger    *zap.Logger
	cron      *cron.Cron

	mu          sync.RWMutex
	lastScanAt  time.Time
	scanRunning bool
}

// NewOrchestrator creates a scan orchestrator with cron scheduling.
func NewOrchestrator(cfg *config.Config, queue *buffer.LocalQueue, connector *connectors.ScannerConnector, logger *zap.Logger) *Orchestrator {
	return &Orchestrator{
		cfg:       cfg,
		queue:     queue,
		connector: connector,
		logger:    logger,
	}
}

// Start begins the cron scheduler.
func (o *Orchestrator) Start(ctx context.Context) {
	o.cron = cron.New(cron.WithLogger(cron.VerbosePrintfLogger(
		newCronLogAdapter(o.logger),
	)))

	_, err := o.cron.AddFunc(o.cfg.ScanSchedule, func() {
		o.runScan(ctx)
	})
	if err != nil {
		o.logger.Error("invalid cron schedule, falling back to weekly",
			zap.String("schedule", o.cfg.ScanSchedule),
			zap.Error(err),
		)
		// Fallback: every Sunday at midnight.
		o.cron.AddFunc("0 0 * * 0", func() {
			o.runScan(ctx)
		})
	}

	o.cron.Start()
	o.logger.Info("scan scheduler started",
		zap.String("schedule", o.cfg.ScanSchedule),
	)
}

// Stop halts the cron scheduler.
func (o *Orchestrator) Stop() {
	if o.cron != nil {
		o.cron.Stop()
	}
}

// LastScanAt returns the timestamp of the last completed scan.
func (o *Orchestrator) LastScanAt() time.Time {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.lastScanAt
}

// IsRunning reports if a scan is currently in progress.
func (o *Orchestrator) IsRunning() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.scanRunning
}

func (o *Orchestrator) runScan(ctx context.Context) {
	o.mu.Lock()
	if o.scanRunning {
		o.mu.Unlock()
		o.logger.Warn("scan already in progress, skipping")
		return
	}
	o.scanRunning = true
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		o.scanRunning = false
		o.lastScanAt = time.Now()
		o.mu.Unlock()
	}()

	o.logger.Info("starting scheduled scan")

	// Trigger scan via backend.
	scanTypes := []string{"vulnerability", "compliance", "inventory"}
	triggerResp, err := o.connector.TriggerScan(ctx, scanTypes)
	if err != nil {
		o.logger.Error("failed to trigger scan", zap.Error(err))
		// Even if the backend is unreachable, generate a local heartbeat result.
		o.bufferHeartbeat()
		return
	}

	o.logger.Info("scan triggered",
		zap.String("scan_job_id", triggerResp.ScanJobID),
		zap.String("status", triggerResp.Status),
	)

	// Collect local endpoint data and push results.
	o.collectResults(ctx, triggerResp.ScanJobID)
}

func (o *Orchestrator) collectResults(ctx context.Context, scanJobID string) {
	// Gather local system data and push it as scan results.
	localResults := o.gatherLocalData()

	for i, result := range localResults {
		payload, err := json.Marshal(result)
		if err != nil {
			o.logger.Error("marshal local result", zap.Error(err))
			continue
		}

		// Try streaming online first.
		statusCode, streamErr := o.connector.StreamResult(ctx, scanJobID, i+1, payload)
		if streamErr != nil || statusCode != http.StatusOK {
			// Fall back to offline buffer.
			o.logger.Info("buffering result offline",
				zap.String("scan_job_id", scanJobID),
				zap.Int("batch_seq", i+1),
			)
			if err := o.queue.Enqueue(scanJobID, i+1, payload); err != nil {
				o.logger.Error("enqueue result", zap.Error(err))
			}
		}
	}

	o.logger.Info("scan results collected",
		zap.String("scan_job_id", scanJobID),
		zap.Int("result_count", len(localResults)),
	)
}

// gatherLocalData collects endpoint information for the scan.
func (o *Orchestrator) gatherLocalData() []map[string]interface{} {
	var results []map[string]interface{}

	// System info result.
	results = append(results, map[string]interface{}{
		"type":      "system_info",
		"agent_id":  o.cfg.AgentID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"hostname":  hostname(),
		"data_dir":  o.cfg.DataDir,
	})

	// Agent health result.
	queueDepth, _ := o.queue.QueueDepth()
	bufferSize, _ := o.queue.BufferSizeMB()
	results = append(results, map[string]interface{}{
		"type":           "agent_health",
		"agent_id":       o.cfg.AgentID,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"queue_depth":    queueDepth,
		"buffer_size_mb": bufferSize,
	})

	return results
}

func (o *Orchestrator) bufferHeartbeat() {
	heartbeat := map[string]interface{}{
		"type":      "heartbeat",
		"agent_id":  o.cfg.AgentID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"status":    "offline_heartbeat",
	}

	payload, err := json.Marshal(heartbeat)
	if err != nil {
		o.logger.Error("marshal heartbeat", zap.Error(err))
		return
	}

	jobID := fmt.Sprintf("heartbeat-%d", time.Now().UnixNano())
	if err := o.queue.Enqueue(jobID, 1, payload); err != nil {
		o.logger.Error("enqueue heartbeat", zap.Error(err))
	}
}

func hostname() string {
	h, err := hostnameOS()
	if err != nil {
		return "unknown"
	}
	return h
}

// cronLogAdapter bridges zap to cron's Printf logger.
type cronLogAdapter struct {
	logger *zap.Logger
}

func newCronLogAdapter(logger *zap.Logger) *cronLogAdapter {
	return &cronLogAdapter{logger: logger}
}

func (a *cronLogAdapter) Printf(format string, args ...interface{}) {
	a.logger.Debug(fmt.Sprintf(format, args...))
}
