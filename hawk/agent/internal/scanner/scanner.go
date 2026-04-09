package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/arc/hawk/agent/internal/auth"
	"github.com/arc/hawk/agent/internal/buffer"
	"github.com/arc/hawk/agent/internal/config"
	"github.com/arc/hawk/agent/internal/connectors"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Scanner orchestrates local scan jobs on a cron schedule.
type Scanner struct {
	cfg        *config.Config
	client     *connectors.Client
	queue      *buffer.LocalQueue
	syncLoop   *buffer.SyncLoop
	authClient *auth.Client
	logger     *zap.Logger
	cron       *cron.Cron

	mu         sync.RWMutex
	lastScanAt time.Time
	scanning   bool
}

// New creates a new Scanner.
func New(
	cfg *config.Config,
	client *connectors.Client,
	queue *buffer.LocalQueue,
	syncLoop *buffer.SyncLoop,
	authClient *auth.Client,
	logger *zap.Logger,
) *Scanner {
	return &Scanner{
		cfg:        cfg,
		client:     client,
		queue:      queue,
		syncLoop:   syncLoop,
		authClient: authClient,
		logger:     logger,
	}
}

// Start initialises the cron scheduler and registers the scan job.
func (s *Scanner) Start(ctx context.Context) error {
	s.cron = cron.New(cron.WithSeconds())

	schedule := s.cfg.GetScanSchedule()
	// robfig/cron/v3 uses 6-field format (sec min hour dom month dow).
	// Config uses standard 5-field (min hour dom month dow), so we prepend "0" for seconds.
	cronExpr := "0 " + schedule

	_, err := s.cron.AddFunc(cronExpr, func() {
		s.runScan(ctx)
	})
	if err != nil {
		return fmt.Errorf("invalid cron schedule %q: %w", schedule, err)
	}

	s.cron.Start()
	s.logger.Info("scan scheduler started", zap.String("schedule", schedule))
	return nil
}

// Stop gracefully shuts down the cron scheduler.
func (s *Scanner) Stop() {
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
		s.logger.Info("scan scheduler stopped")
	}
}

// LastScanAt returns the timestamp of the last completed scan.
func (s *Scanner) LastScanAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastScanAt
}

// IsScanning returns true if a scan is currently running.
func (s *Scanner) IsScanning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scanning
}

// runScan performs one complete scan cycle.
func (s *Scanner) runScan(ctx context.Context) {
	s.mu.Lock()
	if s.scanning {
		s.mu.Unlock()
		s.logger.Warn("scan already in progress – skipping scheduled run")
		return
	}
	s.scanning = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.scanning = false
		s.lastScanAt = time.Now()
		s.mu.Unlock()
	}()

	// Check if the queue is paused (at capacity).
	if s.queue.IsPaused() {
		s.logger.Error("scan skipped – offline buffer at capacity, pending sync required")
		return
	}

	s.logger.Info("starting scheduled scan")
	startTime := time.Now()

	// Step 1: Trigger scan via Scanner API.
	scanReq := connectors.ScanRequest{
		AgentID:  s.cfg.AgentID,
		ScanType: "full",
	}

	scanResp, err := s.client.TriggerScan(ctx, scanReq)
	if err != nil {
		s.logger.Error("failed to trigger scan – will buffer for offline", zap.Error(err))
		return
	}

	scanJobID := scanResp.ScanJobID
	s.logger.Info("scan triggered", zap.String("scan_job_id", scanJobID))

	// Step 2: Poll for completion.
	if err := s.waitForCompletion(ctx, scanJobID); err != nil {
		s.logger.Error("scan did not complete successfully", zap.String("scan_job_id", scanJobID), zap.Error(err))
		return
	}

	// Step 3: Fetch results.
	results, err := s.client.FetchResults(ctx, scanJobID)
	if err != nil {
		s.logger.Error("failed to fetch scan results", zap.String("scan_job_id", scanJobID), zap.Error(err))
		return
	}

	s.logger.Info("scan results fetched",
		zap.String("scan_job_id", scanJobID),
		zap.Int("result_count", len(results)),
	)

	// Step 4: Send results – online mode (streaming) or offline mode (buffering).
	if s.syncLoop.IsOnline() {
		s.streamResults(ctx, scanJobID, results)
	} else {
		s.bufferResults(scanJobID, results)
	}

	elapsed := time.Since(startTime)
	s.logger.Info("scan cycle complete",
		zap.String("scan_job_id", scanJobID),
		zap.Int("results", len(results)),
		zap.Duration("elapsed", elapsed),
	)
}

// streamResults sends results directly to the backend in online mode.
// Falls back to buffering on first failure.
func (s *Scanner) streamResults(ctx context.Context, scanJobID string, results []connectors.ScanResult) {
	for i, result := range results {
		statusCode, err := s.client.StreamResult(ctx, s.cfg.AgentID, result)
		if err != nil || statusCode != http.StatusOK {
			s.logger.Warn("online streaming failed – falling back to offline buffer",
				zap.String("scan_job_id", scanJobID),
				zap.Int("failed_at_index", i),
				zap.Error(err),
			)
			// Buffer the remaining results.
			s.bufferResults(scanJobID, results[i:])
			return
		}
	}
	s.logger.Info("all results streamed online", zap.String("scan_job_id", scanJobID), zap.Int("count", len(results)))
}

// bufferResults stores results in the local SQLite queue for later sync.
func (s *Scanner) bufferResults(scanJobID string, results []connectors.ScanResult) {
	buffered := 0
	for _, result := range results {
		payload, err := json.Marshal(result.Payload)
		if err != nil {
			s.logger.Error("failed to marshal result payload for buffering",
				zap.String("scan_job_id", scanJobID),
				zap.Int("batch_seq", result.BatchSeq),
				zap.Error(err),
			)
			continue
		}

		ok, err := s.queue.Enqueue(scanJobID, result.BatchSeq, payload)
		if err != nil {
			s.logger.Error("failed to enqueue result",
				zap.String("scan_job_id", scanJobID),
				zap.Int("batch_seq", result.BatchSeq),
				zap.Error(err),
			)
			continue
		}
		if !ok {
			s.logger.Error("queue at capacity – stopping buffer operation",
				zap.String("scan_job_id", scanJobID),
			)
			break
		}
		buffered++
	}
	s.logger.Info("results buffered for offline sync",
		zap.String("scan_job_id", scanJobID),
		zap.Int("buffered", buffered),
		zap.Int("total", len(results)),
	)
}

// waitForCompletion polls the scan status until it reaches a terminal state.
func (s *Scanner) waitForCompletion(ctx context.Context, scanJobID string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("scan %s timed out after 30 minutes", scanJobID)
		case <-ticker.C:
			resp, err := s.client.PollScanStatus(ctx, scanJobID)
			if err != nil {
				s.logger.Warn("error polling scan status", zap.String("scan_job_id", scanJobID), zap.Error(err))
				continue
			}

			switch resp.Status {
			case "completed", "complete":
				s.logger.Info("scan completed", zap.String("scan_job_id", scanJobID))
				return nil
			case "failed", "error", "cancelled":
				return fmt.Errorf("scan %s ended with status %q", scanJobID, resp.Status)
			default:
				s.logger.Debug("scan still running",
					zap.String("scan_job_id", scanJobID),
					zap.String("status", resp.Status),
				)
			}
		}
	}
}
