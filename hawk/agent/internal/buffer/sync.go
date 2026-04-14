// Canonical implementation kept in sync with apps/agent/internal/buffer/sync.go.
// When modifying, apply the same change to both files.
package buffer

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arc/hawk/agent/internal/auth"
	"github.com/arc/hawk/agent/internal/connectors"
	"go.uber.org/zap"
)

const (
	healthPollInterval       = 30 * time.Second
	syncBatchSize            = 100
	consecutiveHealthChecks  = 5
	maxAttempts              = 10
)

// SyncLoop manages the background connectivity polling and result syncing.
type SyncLoop struct {
	queue      *LocalQueue
	client     *connectors.Client
	serverURL  string
	authClient *auth.Client
	logger     *zap.Logger

	mu                    sync.RWMutex
	online                atomic.Bool
	consecutiveSuccesses  int
	lastConnectivityCheck time.Time
}

// NewSyncLoop creates a new SyncLoop instance.
func NewSyncLoop(queue *LocalQueue, client *connectors.Client, serverURL string, authClient *auth.Client, logger *zap.Logger) *SyncLoop {
	return &SyncLoop{
		queue:      queue,
		client:     client,
		serverURL:  serverURL,
		authClient: authClient,
		logger:     logger,
	}
}

// IsOnline returns the current connectivity status.
func (s *SyncLoop) IsOnline() bool {
	return s.online.Load()
}

// ConnectivityStatus returns a human-readable connectivity status string.
func (s *SyncLoop) ConnectivityStatus() string {
	if s.online.Load() {
		return "online"
	}
	return "offline"
}

// Run starts the sync loop. It blocks until ctx is cancelled.
func (s *SyncLoop) Run(ctx context.Context) {
	s.logger.Info("sync loop started")
	ticker := time.NewTicker(healthPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("sync loop shutting down – draining pending results")
			s.drainOnShutdown()
			return
		case <-ticker.C:
			s.pollAndSync(ctx)
		}
	}
}

// pollAndSync performs a single health check cycle and syncs pending results if online.
func (s *SyncLoop) pollAndSync(ctx context.Context) {
	healthy := s.client.HealthCheck(ctx)

	// Update connectivity state under mu (guards consecutiveSuccesses and
	// lastConnectivityCheck only — online flag uses atomic.Bool directly to
	// avoid holding mu while doing I/O in syncPendingResults below).
	s.mu.Lock()
	s.lastConnectivityCheck = time.Now()

	if healthy {
		s.consecutiveSuccesses++
		if s.consecutiveSuccesses >= consecutiveHealthChecks {
			if !s.online.Load() {
				s.logger.Info("connectivity restored – switching to online mode",
					zap.Int("consecutive_checks", s.consecutiveSuccesses),
				)
			}
			s.online.Store(true)
		}
	} else {
		if s.online.Load() {
			s.logger.Warn("health check failed – switching to offline mode")
		}
		s.consecutiveSuccesses = 0
		s.online.Store(false)
	}
	s.mu.Unlock() // Release mu BEFORE calling syncPendingResults to avoid AB/BA deadlock.
	// (syncPendingResults → queue.FetchPending → lq.mu; Enqueue also holds lq.mu.
	// Keeping s.mu held here would create a lock-order inversion.)

	// If we are online, drain the pending queue.
	if s.online.Load() {
		s.syncPendingResults(ctx)
	}
}

// syncPendingResults drains the offline buffer by batching results and posting them.
func (s *SyncLoop) syncPendingResults(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		results, err := s.queue.FetchPending(syncBatchSize)
		if err != nil {
			s.logger.Error("failed to fetch pending results", zap.Error(err))
			return
		}
		if len(results) == 0 {
			return // queue is empty
		}

		s.logger.Info("syncing pending results", zap.Int("batch_size", len(results)))

		// Build the batch payload.
		batch := connectors.SyncBatch{
			Results: make([]connectors.SyncRecord, 0, len(results)),
		}
		for _, r := range results {
			batch.Results = append(batch.Results, connectors.SyncRecord{
				ScanJobID: r.ScanJobID,
				BatchSeq:  r.BatchSeq,
				Payload:   json.RawMessage(r.Payload),
			})
		}

		statusCode, err := s.client.PostSyncBatch(ctx, batch)
		if err != nil {
			// Network error – stay offline, don't touch attempts.
			s.logger.Error("sync batch failed (network error)", zap.Error(err))
			s.online.Store(false)
			s.mu.Lock()
			s.consecutiveSuccesses = 0
			s.mu.Unlock()
			return
		}

		switch {
		case statusCode == http.StatusOK:
			// Success – mark all as sent.
			for _, r := range results {
				if err := s.queue.MarkSent(r.ID); err != nil {
					s.logger.Error("failed to mark result as sent", zap.Int64("id", r.ID), zap.Error(err))
				}
			}
			s.logger.Info("batch synced successfully", zap.Int("count", len(results)))

		case statusCode >= 400 && statusCode < 500:
			// Client error (bad payload) – mark as failed, no retry.
			s.logger.Warn("sync batch returned 4xx – marking as failed",
				zap.Int("status", statusCode),
				zap.Int("count", len(results)),
			)
			for _, r := range results {
				if err := s.queue.MarkFailed(r.ID); err != nil {
					s.logger.Error("failed to mark result as failed", zap.Int64("id", r.ID), zap.Error(err))
				}
			}

		default:
			// Server error or timeout – increment attempts, keep pending.
			s.logger.Warn("sync batch returned server error – will retry",
				zap.Int("status", statusCode),
				zap.Int("count", len(results)),
			)
			for _, r := range results {
				attempts, err := s.queue.IncrementAttempts(r.ID)
				if err != nil {
					s.logger.Error("failed to increment attempts", zap.Int64("id", r.ID), zap.Error(err))
					continue
				}
				if attempts > maxAttempts {
					s.logger.Error("ALERT: result exceeded max attempts – marking as failed",
						zap.Int64("id", r.ID),
						zap.String("scan_job_id", r.ScanJobID),
						zap.Int("batch_seq", r.BatchSeq),
						zap.Int("attempts", attempts),
					)
					if err := s.queue.MarkFailed(r.ID); err != nil {
						s.logger.Error("failed to mark as failed after max attempts", zap.Int64("id", r.ID), zap.Error(err))
					}
				}
			}
			return // back off until next poll cycle
		}
	}
}

// drainOnShutdown attempts one final sync pass before the agent exits.
func (s *SyncLoop) drainOnShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !s.client.HealthCheck(ctx) {
		s.logger.Warn("server unreachable during shutdown – pending results will remain in local queue")
		return
	}

	s.syncPendingResults(ctx)
}
