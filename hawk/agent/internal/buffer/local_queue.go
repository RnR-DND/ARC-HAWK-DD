// Canonical implementation kept in sync with apps/agent/internal/buffer/local_queue.go.
// When modifying, apply the same change to both files.
package buffer

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

const (
	createTableSQL = `
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
	`
	createIndexSQL = `
		CREATE UNIQUE INDEX IF NOT EXISTS idx_pending_job_batch
		ON pending_results(scan_job_id, batch_seq);
	`
)

// LocalQueue is an SQLite-backed offline buffer for scan results.
type LocalQueue struct {
	db         *sql.DB
	dbPath     string
	maxSizeMB  int
	logger     *zap.Logger
	mu         sync.Mutex
	pauseScans bool
}

// PendingResult represents a row from the pending_results table.
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

// NewLocalQueue opens (or creates) the SQLite database at dbPath and initialises the schema.
func NewLocalQueue(dbPath string, maxSizeMB int, logger *zap.Logger) (*LocalQueue, error) {
	// Enable WAL mode and busy timeout for concurrent access.
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite3 %s: %w", dbPath, err)
	}

	// Limit to a single connection – SQLite handles concurrency through its own locking.
	db.SetMaxOpenConns(1)

	// Create table and index.
	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating pending_results table: %w", err)
	}
	if _, err := db.Exec(createIndexSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating unique index: %w", err)
	}

	lq := &LocalQueue{
		db:        db,
		dbPath:    dbPath,
		maxSizeMB: maxSizeMB,
		logger:    logger,
	}

	logger.Info("local queue initialised", zap.String("path", dbPath), zap.Int("max_size_mb", maxSizeMB))
	return lq, nil
}

// Enqueue inserts a scan result into the offline buffer.
// Returns true if the result was enqueued, false if the queue is at capacity.
func (lq *LocalQueue) Enqueue(scanJobID string, batchSeq int, payload []byte) (bool, error) {
	lq.mu.Lock()
	defer lq.mu.Unlock()

	// Check if we need to enforce size limits.
	if err := lq.enforceCapacity(); err != nil {
		return false, err
	}

	if lq.pauseScans {
		lq.logger.Warn("queue is at capacity with all pending items – new scans paused",
			zap.String("scan_job_id", scanJobID),
			zap.Int("batch_seq", batchSeq),
		)
		return false, nil
	}

	_, err := lq.db.Exec(`
		INSERT OR REPLACE INTO pending_results (scan_job_id, batch_seq, payload, status, attempts)
		VALUES (?, ?, ?, 'pending', 0)
	`, scanJobID, batchSeq, payload)
	if err != nil {
		return false, fmt.Errorf("enqueueing result: %w", err)
	}

	return true, nil
}

// FetchPending returns up to `limit` oldest pending results.
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
		return nil, fmt.Errorf("fetching pending results: %w", err)
	}
	defer rows.Close()

	var results []PendingResult
	for rows.Next() {
		var r PendingResult
		if err := rows.Scan(&r.ID, &r.ScanJobID, &r.BatchSeq, &r.Payload,
			&r.CreatedAt, &r.Attempts, &r.LastAttemptAt, &r.Status); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// MarkSent sets the status of a result to 'sent'.
func (lq *LocalQueue) MarkSent(id int64) error {
	lq.mu.Lock()
	defer lq.mu.Unlock()

	_, err := lq.db.Exec(`
		UPDATE pending_results
		SET status = 'sent', last_attempt_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	return err
}

// MarkFailed sets the status of a result to 'failed'.
func (lq *LocalQueue) MarkFailed(id int64) error {
	lq.mu.Lock()
	defer lq.mu.Unlock()

	_, err := lq.db.Exec(`
		UPDATE pending_results
		SET status = 'failed', last_attempt_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	return err
}

// IncrementAttempts bumps the attempt counter and last_attempt_at for a result.
// Returns the new attempt count.
func (lq *LocalQueue) IncrementAttempts(id int64) (int, error) {
	lq.mu.Lock()
	defer lq.mu.Unlock()

	_, err := lq.db.Exec(`
		UPDATE pending_results
		SET attempts = attempts + 1, last_attempt_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	if err != nil {
		return 0, err
	}

	var attempts int
	err = lq.db.QueryRow(`SELECT attempts FROM pending_results WHERE id = ?`, id).Scan(&attempts)
	return attempts, err
}

// PendingCount returns the number of results with status='pending'.
func (lq *LocalQueue) PendingCount() (int, error) {
	var count int
	err := lq.db.QueryRow(`SELECT COUNT(*) FROM pending_results WHERE status = 'pending'`).Scan(&count)
	return count, err
}

// TotalCount returns the total number of results in the queue.
func (lq *LocalQueue) TotalCount() (int, error) {
	var count int
	err := lq.db.QueryRow(`SELECT COUNT(*) FROM pending_results`).Scan(&count)
	return count, err
}

// BufferSizeMB returns the current SQLite database file size in MB.
func (lq *LocalQueue) BufferSizeMB() float64 {
	info, err := os.Stat(lq.dbPath)
	if err != nil {
		return 0
	}
	return float64(info.Size()) / (1024 * 1024)
}

// IsPaused returns true if the queue is at capacity and scans should pause.
func (lq *LocalQueue) IsPaused() bool {
	lq.mu.Lock()
	defer lq.mu.Unlock()
	return lq.pauseScans
}

// Close closes the SQLite database.
func (lq *LocalQueue) Close() error {
	lq.logger.Info("closing local queue")
	return lq.db.Close()
}

// enforceCapacity checks the DB file size against maxSizeMB and purges old sent rows if needed.
// Must be called with lq.mu held.
func (lq *LocalQueue) enforceCapacity() error {
	sizeMB := lq.BufferSizeMB()
	if sizeMB < float64(lq.maxSizeMB) {
		lq.pauseScans = false
		return nil
	}

	lq.logger.Warn("buffer size exceeds limit, purging sent results",
		zap.Float64("size_mb", sizeMB),
		zap.Int("limit_mb", lq.maxSizeMB),
	)

	// Phase 1: purge rows with status='sent' (oldest first).
	result, err := lq.db.Exec(`
		DELETE FROM pending_results
		WHERE id IN (
			SELECT id FROM pending_results
			WHERE status = 'sent'
			ORDER BY created_at ASC
			LIMIT 1000
		)
	`)
	if err != nil {
		return fmt.Errorf("purging sent results: %w", err)
	}

	purged, _ := result.RowsAffected()
	lq.logger.Info("purged sent results", zap.Int64("count", purged))

	// Vacuum to reclaim space.
	if purged > 0 {
		if _, err := lq.db.Exec("VACUUM"); err != nil {
			lq.logger.Warn("VACUUM failed", zap.Error(err))
		}
	}

	// Re-check size after purge.
	sizeMB = lq.BufferSizeMB()
	if sizeMB < float64(lq.maxSizeMB) {
		lq.pauseScans = false
		return nil
	}

	// Phase 2: purge failed results.
	result, err = lq.db.Exec(`
		DELETE FROM pending_results
		WHERE id IN (
			SELECT id FROM pending_results
			WHERE status = 'failed'
			ORDER BY created_at ASC
			LIMIT 1000
		)
	`)
	if err != nil {
		return fmt.Errorf("purging failed results: %w", err)
	}

	purged, _ = result.RowsAffected()
	if purged > 0 {
		lq.logger.Info("purged failed results", zap.Int64("count", purged))
		if _, err := lq.db.Exec("VACUUM"); err != nil {
			lq.logger.Warn("VACUUM failed", zap.Error(err))
		}
	}

	// Final check: if still over limit, all rows are pending.
	sizeMB = lq.BufferSizeMB()
	if sizeMB >= float64(lq.maxSizeMB) {
		lq.logger.Error("ALERT: queue at capacity with all pending items – pausing new scans",
			zap.Float64("size_mb", sizeMB),
			zap.Int("limit_mb", lq.maxSizeMB),
		)
		lq.pauseScans = true
	}

	return nil
}
