package buffer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arc/hawk-agent/internal/config"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

// PendingResult represents a row in the pending_results table.
type PendingResult struct {
	ID            int64
	ScanJobID     string
	BatchSeq      int
	Payload       []byte
	CreatedAt     time.Time
	Attempts      int
	LastAttemptAt sql.NullTime
	Status        string
}

// LocalQueue manages the SQLite offline buffer.
// Canonical implementation — keep in sync with hawk/agent/internal/buffer/local_queue.go.
type LocalQueue struct {
	db         *sql.DB
	cfg        *config.Config
	logger     *zap.Logger
	mu         sync.Mutex
	pauseScans atomic.Bool // set true when buffer is at capacity with all pending items
}

// NewLocalQueue opens (or creates) the SQLite queue database.
func NewLocalQueue(cfg *config.Config, logger *zap.Logger) (*LocalQueue, error) {
	// Ensure data directory exists.
	if err := os.MkdirAll(cfg.DataDir, 0750); err != nil {
		return nil, fmt.Errorf("create data dir %s: %w", cfg.DataDir, err)
	}

	dbPath := cfg.DBPath()
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", dbPath, err)
	}

	// Set connection pool for SQLite (single writer).
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	lq := &LocalQueue{
		db:     db,
		cfg:    cfg,
		logger: logger,
	}

	if err := lq.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate sqlite: %w", err)
	}

	return lq, nil
}

func (lq *LocalQueue) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS pending_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scan_job_id TEXT NOT NULL,
		batch_seq INTEGER NOT NULL,
		payload BLOB NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		attempts INTEGER DEFAULT 0,
		last_attempt_at DATETIME,
		status TEXT DEFAULT 'pending'
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_pending_job_batch
		ON pending_results(scan_job_id, batch_seq);
	`
	_, err := lq.db.Exec(schema)
	return err
}

// Enqueue inserts a scan result into the offline buffer.
// Returns (true, nil) if the result was queued, (false, nil) if the buffer is
// at capacity with all entries pending (scans should pause — call IsPaused()).
// Returns (false, err) only on genuine I/O failures.
func (lq *LocalQueue) Enqueue(scanJobID string, batchSeq int, payload []byte) (bool, error) {
	lq.mu.Lock()
	defer lq.mu.Unlock()

	// Short-circuit if already paused.
	if lq.pauseScans.Load() {
		lq.logger.Warn("queue is at capacity with all pending items – new scans paused",
			zap.String("scan_job_id", scanJobID),
			zap.Int("batch_seq", batchSeq),
		)
		return false, nil
	}

	// Check buffer size limit.
	exceeded, allPending, err := lq.checkBufferLimit()
	if err != nil {
		return false, fmt.Errorf("check buffer limit: %w", err)
	}
	if exceeded {
		if allPending {
			// Buffer full and nothing to purge — pause new scans gracefully.
			lq.logger.Error("ALERT: buffer at max capacity with all pending items – pausing new scans",
				zap.Int("buffer_max_mb", lq.cfg.BufferMaxMB),
			)
			lq.pauseScans.Store(true)
			return false, nil
		}
		// Purge oldest sent entries first.
		lq.logger.Warn("buffer approaching limit, purging oldest sent entries")
		if err := lq.purgeOldestSent(); err != nil {
			return false, fmt.Errorf("purge sent entries: %w", err)
		}
	}

	// If we just purged, clear pause flag.
	lq.pauseScans.Store(false)

	_, err = lq.db.Exec(`
		INSERT OR REPLACE INTO pending_results (scan_job_id, batch_seq, payload, status, attempts)
		VALUES (?, ?, ?, 'pending', 0)
	`, scanJobID, batchSeq, payload)
	if err != nil {
		return false, fmt.Errorf("insert pending result: %w", err)
	}
	return true, nil
}

// IsPaused returns true if the buffer is at capacity and scans should pause
// until the sync loop drains pending items.
func (lq *LocalQueue) IsPaused() bool {
	return lq.pauseScans.Load()
}

// FetchPending returns up to `limit` oldest pending results for syncing.
func (lq *LocalQueue) FetchPending(limit int) ([]PendingResult, error) {
	lq.mu.Lock()
	defer lq.mu.Unlock()

	rows, err := lq.db.Query(`
		SELECT id, scan_job_id, batch_seq, payload, created_at, attempts, last_attempt_at, status
		FROM pending_results
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query pending: %w", err)
	}
	defer rows.Close()

	var results []PendingResult
	for rows.Next() {
		var r PendingResult
		if err := rows.Scan(&r.ID, &r.ScanJobID, &r.BatchSeq, &r.Payload,
			&r.CreatedAt, &r.Attempts, &r.LastAttemptAt, &r.Status); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// MarkSent updates a result to 'sent' status.
// If scans were paused due to buffer capacity, this also clears the pause flag
// so the scanner can resume once the queue drains sufficiently.
func (lq *LocalQueue) MarkSent(id int64) error {
	lq.mu.Lock()
	defer lq.mu.Unlock()

	_, err := lq.db.Exec(`
		UPDATE pending_results SET status = 'sent', last_attempt_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	if err != nil {
		return err
	}
	// Clear pause flag — a sent row means the queue is draining; the next Enqueue
	// call will re-check capacity and re-set the flag only if still over limit.
	lq.pauseScans.Store(false)
	return nil
}

// MarkFailed updates a result to 'failed' status.
func (lq *LocalQueue) MarkFailed(id int64) error {
	lq.mu.Lock()
	defer lq.mu.Unlock()

	_, err := lq.db.Exec(`
		UPDATE pending_results SET status = 'failed', last_attempt_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	return err
}

// IncrementAttempts bumps the attempt counter and updates last_attempt_at.
func (lq *LocalQueue) IncrementAttempts(id int64) error {
	lq.mu.Lock()
	defer lq.mu.Unlock()

	_, err := lq.db.Exec(`
		UPDATE pending_results
		SET attempts = attempts + 1, last_attempt_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	return err
}

// GetAttempts returns the current attempt count for a result.
func (lq *LocalQueue) GetAttempts(id int64) (int, error) {
	var attempts int
	err := lq.db.QueryRow(`SELECT attempts FROM pending_results WHERE id = ?`, id).Scan(&attempts)
	return attempts, err
}

// QueueDepth returns the number of pending results.
func (lq *LocalQueue) QueueDepth() (int, error) {
	var count int
	err := lq.db.QueryRow(`SELECT COUNT(*) FROM pending_results WHERE status = 'pending'`).Scan(&count)
	return count, err
}

// BufferSizeMB returns the current SQLite database file size in megabytes.
func (lq *LocalQueue) BufferSizeMB() (float64, error) {
	dbPath := lq.cfg.DBPath()
	info, err := os.Stat(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return float64(info.Size()) / (1024 * 1024), nil
}

// Close closes the SQLite database.
func (lq *LocalQueue) Close() error {
	return lq.db.Close()
}

func (lq *LocalQueue) checkBufferLimit() (exceeded bool, allPending bool, err error) {
	dbPath := lq.cfg.DBPath()
	info, err := os.Stat(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}

	sizeMB := float64(info.Size()) / (1024 * 1024)
	if sizeMB < float64(lq.cfg.BufferMaxMB) {
		return false, false, nil
	}

	// Buffer exceeded. Check if there are any sent entries to purge.
	var sentCount int
	err = lq.db.QueryRow(`SELECT COUNT(*) FROM pending_results WHERE status = 'sent'`).Scan(&sentCount)
	if err != nil {
		return true, false, err
	}
	return true, sentCount == 0, nil
}

func (lq *LocalQueue) purgeOldestSent() error {
	// Delete oldest 100 sent entries.
	_, err := lq.db.Exec(`
		DELETE FROM pending_results
		WHERE id IN (
			SELECT id FROM pending_results
			WHERE status = 'sent'
			ORDER BY created_at ASC
			LIMIT 100
		)
	`)
	if err != nil {
		return err
	}

	// Reclaim space.
	_, err = lq.db.Exec("VACUUM")
	return err
}

// DataDir returns the full path to the data directory (for health reporting).
func (lq *LocalQueue) DataDir() string {
	return filepath.Dir(lq.cfg.DBPath())
}
