package service

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// stalledScanThreshold is how long a scan can stay in "running" state before
// the watchdog considers it stalled and marks it failed.
const stalledScanThreshold = 2 * time.Hour

// ScanWatchdog polls for scan_runs rows that have been in "running" status
// longer than stalledScanThreshold and marks them "failed".
// This prevents ghost scans from blocking UI and consuming resources.
type ScanWatchdog struct {
	db       *sql.DB
	interval time.Duration
}

// NewScanWatchdog creates a watchdog that ticks every interval.
func NewScanWatchdog(db *sql.DB, interval time.Duration) *ScanWatchdog {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &ScanWatchdog{db: db, interval: interval}
}

// Start runs the watchdog in the background until ctx is cancelled.
func (w *ScanWatchdog) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := w.reapStalled(ctx); err != nil {
					log.Printf("WARN: scan watchdog reap failed: %v", err)
				}
			}
		}
	}()
}

func (w *ScanWatchdog) reapStalled(ctx context.Context) error {
	cutoff := time.Now().UTC().Add(-stalledScanThreshold)
	result, err := w.db.ExecContext(ctx, `
		UPDATE scan_runs
		SET status = 'failed',
		    updated_at = NOW(),
		    error_message = 'watchdog: scan exceeded maximum run time and was marked failed'
		WHERE status = 'running'
		  AND updated_at < $1
	`, cutoff)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n > 0 {
		log.Printf("INFO: scan watchdog reaped %d stalled scan(s) (threshold: %s)", n, stalledScanThreshold)
	}
	return nil
}
