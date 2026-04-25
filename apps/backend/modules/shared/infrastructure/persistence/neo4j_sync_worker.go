package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var neo4jSyncDeadLetterTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "neo4j_sync_dead_letter_total",
	Help: "Total neo4j sync queue rows transitioned to dead_letter status",
})

// neo4jSyncer is the subset of Neo4jRepository used by the sync worker.
// Extracted to allow test doubles without a real Neo4j connection.
type neo4jSyncer interface {
	SyncFindingsToPIICategories(ctx context.Context, assetID string, piiTypeCounts map[string]int) error
}

// Neo4jSyncWorker drains the neo4j_sync_queue outbox table.
// It runs as a background goroutine providing at-least-once delivery of Neo4j writes.
type Neo4jSyncWorker struct {
	db        *sql.DB
	neo4jRepo neo4jSyncer
	stop      chan struct{}
	interval  time.Duration
}

func NewNeo4jSyncWorker(db *sql.DB, neo4jRepo *Neo4jRepository) *Neo4jSyncWorker {
	return &Neo4jSyncWorker{
		db:        db,
		neo4jRepo: neo4jRepo,
		stop:      make(chan struct{}),
		interval:  5 * time.Second,
	}
}

func (w *Neo4jSyncWorker) Start(ctx context.Context) {
	// Reset rows stuck in 'processing' from a previous crash before starting the loop.
	// Rows in 'processing' longer than 10 minutes indicate a worker that died between
	// the commit that locked them and the follow-up status updates.
	w.recoverStaleRows(ctx)
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-w.stop:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.processBatch(ctx)
			}
		}
	}()
}

func (w *Neo4jSyncWorker) Stop() {
	close(w.stop)
}

// recoverStaleRows resets rows stuck in 'processing' back to 'pending' so they
// are retried. A row is considered stale when it has been in 'processing' for
// more than 10 minutes — longer than any reasonable Neo4j write should take.
// We reset to 'pending' (not 'failed') because a crashed worker did not
// actually attempt the operation, so the attempt count should not increment.
func (w *Neo4jSyncWorker) recoverStaleRows(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	res, err := w.db.ExecContext(ctx, `
		UPDATE neo4j_sync_queue
		SET    status     = 'pending',
		       last_error = 'recovered from stale processing state',
		       updated_at = NOW()
		WHERE  status = 'processing'
		  AND  updated_at < NOW() - INTERVAL '10 minutes'
	`)
	if err != nil {
		log.Printf("neo4j_sync_worker: stale row recovery error: %v", err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("neo4j_sync_worker: recovered %d stale rows from 'processing'", n)
	}
}

type qrow struct {
	id        string
	operation string
	payload   json.RawMessage
	attempts  int
}

func (w *Neo4jSyncWorker) processBatch(ctx context.Context) {
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tx, err := w.db.BeginTx(queryCtx, nil)
	if err != nil {
		log.Printf("neo4j_sync_worker: begin tx error: %v", err)
		return
	}

	rows, err := tx.QueryContext(queryCtx, `
		SELECT id, operation, payload, attempts
		FROM   neo4j_sync_queue
		WHERE  status IN ('pending', 'failed') AND attempts < 5
		ORDER  BY created_at ASC
		LIMIT  50
		FOR UPDATE SKIP LOCKED
	`)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("neo4j_sync_worker: rollback error after query failure: %v", rbErr)
		}
		log.Printf("neo4j_sync_worker: query error: %v", err)
		return
	}

	var batch []qrow
	for rows.Next() {
		var r qrow
		if err := rows.Scan(&r.id, &r.operation, &r.payload, &r.attempts); err != nil {
			log.Printf("neo4j_sync_worker: scan error: %v", err)
			continue
		}
		batch = append(batch, r)
	}
	rows.Close()

	if len(batch) == 0 {
		tx.Rollback() //nolint:errcheck — nothing was written
		return
	}

	// Mark all as 'processing' before releasing the lock. Once committed, these
	// rows are excluded from future SELECT FOR UPDATE SKIP LOCKED queries, so
	// only this worker will update their final status.
	for _, r := range batch {
		if _, err := tx.ExecContext(queryCtx,
			`UPDATE neo4j_sync_queue SET status='processing', updated_at=NOW() WHERE id=$1`,
			r.id); err != nil {
			log.Printf("neo4j_sync_worker: failed to mark row %s as processing: %v", r.id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("neo4j_sync_worker: rollback error after commit failure: %v", rbErr)
		}
		log.Printf("neo4j_sync_worker: commit error marking batch as processing: %v", err)
		return
	}

	// Process without lock held. Each status update is a single-statement exec.
	// No transaction is needed here because rows are now uniquely owned by this
	// worker — 'processing' rows are excluded from all other workers' queries.
	for _, r := range batch {
		if err := w.process(ctx, r.operation, r.payload); err != nil {
			attempts := r.attempts + 1
			if attempts >= 5 {
				if _, updateErr := w.db.ExecContext(ctx,
					`UPDATE neo4j_sync_queue
					 SET status='dead_letter', attempts=$1, last_error=$2, updated_at=NOW()
					 WHERE id=$3`,
					attempts, err.Error(), r.id); updateErr != nil {
					log.Printf("neo4j_sync_worker: failed to mark row %s as dead_letter: %v", r.id, updateErr)
				}
				neo4jSyncDeadLetterTotal.Inc()
			} else {
				if _, updateErr := w.db.ExecContext(ctx,
					`UPDATE neo4j_sync_queue
					 SET status='failed', attempts=$1, last_error=$2, updated_at=NOW()
					 WHERE id=$3`,
					attempts, err.Error(), r.id); updateErr != nil {
					log.Printf("neo4j_sync_worker: failed to mark row %s as failed: %v", r.id, updateErr)
				}
			}
		} else {
			if _, updateErr := w.db.ExecContext(ctx,
				`UPDATE neo4j_sync_queue
				 SET status='done', processed_at=NOW(), updated_at=NOW()
				 WHERE id=$1`,
				r.id); updateErr != nil {
				log.Printf("neo4j_sync_worker: failed to mark row %s as done: %v", r.id, updateErr)
			}
		}
	}
}

func (w *Neo4jSyncWorker) process(ctx context.Context, operation string, payload json.RawMessage) error {
	switch operation {
	case "sync_findings":
		var p struct {
			AssetID       string         `json:"asset_id"`
			PIITypeCounts map[string]int `json:"pii_type_counts"`
			ScanID        string         `json:"scan_id"`
		}
		if err := json.Unmarshal(payload, &p); err != nil {
			return err
		}
		return w.neo4jRepo.SyncFindingsToPIICategories(ctx, p.AssetID, p.PIITypeCounts)
	default:
		log.Printf("neo4j_sync_worker: unknown operation %q", operation)
		return nil
	}
}
