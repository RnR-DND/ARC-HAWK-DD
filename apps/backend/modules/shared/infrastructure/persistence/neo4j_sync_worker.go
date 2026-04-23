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

// Neo4jSyncWorker drains the neo4j_sync_queue outbox table.
// It runs as a background goroutine providing at-least-once delivery of Neo4j writes.
type Neo4jSyncWorker struct {
	db        *sql.DB
	neo4jRepo *Neo4jRepository
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

type qrow struct {
	id        string
	operation string
	payload   json.RawMessage
	attempts  int
}

func (w *Neo4jSyncWorker) processBatch(ctx context.Context) {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("neo4j_sync_worker: begin tx error: %v", err)
		return
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id, operation, payload, attempts
		FROM neo4j_sync_queue
		WHERE status IN ('pending', 'failed') AND attempts < 5
		ORDER BY created_at ASC
		LIMIT 50
		FOR UPDATE SKIP LOCKED
	`)
	if err != nil {
		tx.Rollback()
		log.Printf("neo4j_sync_worker: query error: %v", err)
		return
	}

	var batch []qrow
	for rows.Next() {
		var r qrow
		if err := rows.Scan(&r.id, &r.operation, &r.payload, &r.attempts); err != nil {
			continue
		}
		batch = append(batch, r)
	}
	rows.Close()

	// Mark all as 'processing' before releasing the lock
	for _, r := range batch {
		_, _ = tx.ExecContext(ctx, `UPDATE neo4j_sync_queue SET status='processing' WHERE id=$1`, r.id)
	}
	tx.Commit()

	// Now process without lock held
	for _, r := range batch {
		if err := w.process(ctx, r.operation, r.payload); err != nil {
			attempts := r.attempts + 1
			if attempts >= 5 {
				_, _ = w.db.ExecContext(ctx,
					`UPDATE neo4j_sync_queue SET status='dead_letter', attempts=$1, last_error=$2 WHERE id=$3`,
					attempts, err.Error(), r.id)
				neo4jSyncDeadLetterTotal.Inc()
			} else {
				_, _ = w.db.ExecContext(ctx,
					`UPDATE neo4j_sync_queue SET status='failed', attempts=$1, last_error=$2 WHERE id=$3`,
					attempts, err.Error(), r.id)
			}
		} else {
			_, _ = w.db.ExecContext(ctx,
				`UPDATE neo4j_sync_queue SET status='done', processed_at=NOW() WHERE id=$1`,
				r.id)
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
