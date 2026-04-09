package buffer

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arc/hawk-agent/internal/config"
	"github.com/arc/hawk-agent/internal/connectors"
	"go.uber.org/zap"
)

const (
	healthPollInterval  = 30 * time.Second
	syncBatchSize       = 100
	maxRetryAttempts    = 10
	consecutiveOnline   = 5 // health checks needed to go back online
)

// SyncLoop manages the background sync of buffered results.
type SyncLoop struct {
	cfg       *config.Config
	queue     *LocalQueue
	connector *connectors.ScannerConnector
	logger    *zap.Logger

	online         atomic.Bool
	healthStreak   atomic.Int32
	lastOnlineAt   time.Time

	stopOnce sync.Once
	done     chan struct{}
}

// NewSyncLoop creates a new sync loop.
func NewSyncLoop(cfg *config.Config, queue *LocalQueue, connector *connectors.ScannerConnector, logger *zap.Logger) *SyncLoop {
	return &SyncLoop{
		cfg:       cfg,
		queue:     queue,
		connector: connector,
		logger:    logger,
		done:      make(chan struct{}),
	}
}

// Start begins the connectivity poller and sync loop.
func (sl *SyncLoop) Start(ctx context.Context) {
	// Start the health check poller.
	go sl.healthPoller(ctx)
	// Start the sync worker.
	go sl.syncWorker(ctx)
}

// IsOnline reports current connectivity status.
func (sl *SyncLoop) IsOnline() bool {
	return sl.online.Load()
}

// LastOnlineAt returns when connectivity was last established.
func (sl *SyncLoop) LastOnlineAt() time.Time {
	return sl.lastOnlineAt
}

// FlushAndStop attempts to flush all pending items then stops.
func (sl *SyncLoop) FlushAndStop(ctx context.Context) {
	sl.stopOnce.Do(func() {
		// Attempt one final flush if online.
		if sl.online.Load() {
			sl.logger.Info("performing final sync flush before shutdown")
			sl.syncBatch(ctx)
		}
		close(sl.done)
	})
}

func (sl *SyncLoop) healthPoller(ctx context.Context) {
	ticker := time.NewTicker(healthPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sl.done:
			return
		case <-ticker.C:
			err := sl.connector.HealthCheck(ctx)
			if err != nil {
				sl.healthStreak.Store(0)
				if sl.online.Load() {
					sl.logger.Warn("connectivity lost", zap.Error(err))
					sl.online.Store(false)
				}
			} else {
				streak := sl.healthStreak.Add(1)
				if !sl.online.Load() && streak >= consecutiveOnline {
					sl.logger.Info("connectivity restored",
						zap.Int32("consecutive_checks", streak),
					)
					sl.online.Store(true)
					sl.lastOnlineAt = time.Now()
				} else if sl.online.Load() {
					sl.lastOnlineAt = time.Now()
				}
			}
		}
	}
}

func (sl *SyncLoop) syncWorker(ctx context.Context) {
	// Check every 5 seconds if we have pending items and are online.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sl.done:
			return
		case <-ticker.C:
			if sl.online.Load() {
				sl.syncBatch(ctx)
			}
		}
	}
}

func (sl *SyncLoop) syncBatch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		pending, err := sl.queue.FetchPending(syncBatchSize)
		if err != nil {
			sl.logger.Error("fetch pending results", zap.Error(err))
			return
		}
		if len(pending) == 0 {
			return // Queue empty.
		}

		sl.logger.Info("syncing batch", zap.Int("count", len(pending)))

		// Build sync request.
		syncReq := &connectors.SyncRequest{
			AgentID: sl.cfg.AgentID,
			Results: make([]connectors.SyncResult, 0, len(pending)),
		}
		for _, p := range pending {
			syncReq.Results = append(syncReq.Results, connectors.SyncResult{
				ScanJobID: p.ScanJobID,
				BatchSeq:  p.BatchSeq,
				Payload:   json.RawMessage(p.Payload),
			})
		}

		statusCode, err := sl.connector.SyncResults(ctx, syncReq)
		if err != nil {
			sl.logger.Error("sync request error (network/timeout)", zap.Error(err))
			// Increment attempts for all in batch.
			for _, p := range pending {
				if incrErr := sl.queue.IncrementAttempts(p.ID); incrErr != nil {
					sl.logger.Error("increment attempts", zap.Int64("id", p.ID), zap.Error(incrErr))
				}
				sl.checkMaxAttempts(p.ID)
			}
			return // Will retry on next cycle.
		}

		switch {
		case statusCode == http.StatusOK:
			// Success: mark all as sent.
			for _, p := range pending {
				if err := sl.queue.MarkSent(p.ID); err != nil {
					sl.logger.Error("mark sent", zap.Int64("id", p.ID), zap.Error(err))
				}
			}
			sl.logger.Info("batch synced successfully", zap.Int("count", len(pending)))

		case statusCode >= 400 && statusCode < 500:
			// Client error: mark as failed, no retry.
			sl.logger.Warn("sync returned 4xx, marking batch as failed",
				zap.Int("status", statusCode),
				zap.Int("count", len(pending)),
			)
			for _, p := range pending {
				if err := sl.queue.MarkFailed(p.ID); err != nil {
					sl.logger.Error("mark failed", zap.Int64("id", p.ID), zap.Error(err))
				}
			}

		default:
			// 5xx or unexpected: keep as pending, increment attempts.
			sl.logger.Warn("sync returned server error, will retry",
				zap.Int("status", statusCode),
				zap.Int("count", len(pending)),
			)
			for _, p := range pending {
				if incrErr := sl.queue.IncrementAttempts(p.ID); incrErr != nil {
					sl.logger.Error("increment attempts", zap.Int64("id", p.ID), zap.Error(incrErr))
				}
				sl.checkMaxAttempts(p.ID)
			}
			return // Back off until next sync cycle.
		}

		// Continue draining the queue if the batch was successful.
		if statusCode != http.StatusOK {
			return
		}
	}
}

func (sl *SyncLoop) checkMaxAttempts(id int64) {
	attempts, err := sl.queue.GetAttempts(id)
	if err != nil {
		return
	}
	if attempts > maxRetryAttempts {
		sl.logger.Error("max retry attempts exceeded, marking as failed",
			zap.Int64("id", id),
			zap.Int("attempts", attempts),
		)
		if err := sl.queue.MarkFailed(id); err != nil {
			sl.logger.Error("mark failed after max attempts", zap.Int64("id", id), zap.Error(err))
		}
	}
}
