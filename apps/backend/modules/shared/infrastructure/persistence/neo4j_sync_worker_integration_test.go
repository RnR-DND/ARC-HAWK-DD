package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/arc-platform/backend/modules/shared/testutil"
)

// stubNeo4j implements neo4jSyncer for integration tests.
// It records calls and can be configured to return errors.
type stubNeo4j struct {
	calls  []string
	failOn map[string]error // assetID → error to return
}

func (s *stubNeo4j) SyncFindingsToPIICategories(_ context.Context, assetID string, _ map[string]int) error {
	s.calls = append(s.calls, assetID)
	if s.failOn != nil {
		if err, ok := s.failOn[assetID]; ok {
			return err
		}
	}
	return nil
}

// insertPendingRow inserts a row into neo4j_sync_queue and returns its id.
func insertPendingRow(t *testing.T, tdb *testutil.TestDB, assetID string, attempts int, status string) string {
	t.Helper()
	id := fmt.Sprintf("test-%d", time.Now().UnixNano())
	payload, _ := json.Marshal(map[string]interface{}{
		"asset_id":        assetID,
		"pii_type_counts": map[string]int{"AADHAAR": 3},
		"scan_id":         "scan-001",
	})
	_, err := tdb.DB.ExecContext(context.Background(), `
		INSERT INTO neo4j_sync_queue (id, operation, payload, status, attempts, created_at, updated_at)
		VALUES ($1, 'sync_findings', $2, $3, $4, NOW(), NOW())
	`, id, payload, status, attempts)
	if err != nil {
		t.Fatalf("insert queue row: %v", err)
	}
	return id
}

func rowStatus(t *testing.T, tdb *testutil.TestDB, id string) string {
	t.Helper()
	var status string
	err := tdb.DB.QueryRowContext(context.Background(),
		"SELECT status FROM neo4j_sync_queue WHERE id=$1", id).Scan(&status)
	if err != nil {
		t.Fatalf("query row status for %s: %v", id, err)
	}
	return status
}

func rowAttempts(t *testing.T, tdb *testutil.TestDB, id string) int {
	t.Helper()
	var attempts int
	err := tdb.DB.QueryRowContext(context.Background(),
		"SELECT attempts FROM neo4j_sync_queue WHERE id=$1", id).Scan(&attempts)
	if err != nil {
		t.Fatalf("query row attempts for %s: %v", id, err)
	}
	return attempts
}

func TestNeo4jSyncWorker_ProcessBatch_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skip: integration test requires Docker")
	}
	tdb := testutil.NewTestDB(t)

	neo4j := &stubNeo4j{}
	w := &Neo4jSyncWorker{db: tdb.DB, neo4jRepo: neo4j, stop: make(chan struct{}), interval: time.Second}

	id := insertPendingRow(t, tdb, "asset-aaa", 0, "pending")

	w.processBatch(context.Background())

	if got := rowStatus(t, tdb, id); got != "done" {
		t.Errorf("expected status=done, got %q", got)
	}
	if len(neo4j.calls) != 1 || neo4j.calls[0] != "asset-aaa" {
		t.Errorf("expected 1 neo4j call for asset-aaa, got %v", neo4j.calls)
	}
}

func TestNeo4jSyncWorker_ProcessBatch_FailureIncrementsAttempts(t *testing.T) {
	if testing.Short() {
		t.Skip("skip: integration test requires Docker")
	}
	tdb := testutil.NewTestDB(t)

	neo4j := &stubNeo4j{failOn: map[string]error{"asset-bbb": fmt.Errorf("neo4j unavailable")}}
	w := &Neo4jSyncWorker{db: tdb.DB, neo4jRepo: neo4j, stop: make(chan struct{}), interval: time.Second}

	id := insertPendingRow(t, tdb, "asset-bbb", 0, "pending")

	w.processBatch(context.Background())

	if got := rowStatus(t, tdb, id); got != "failed" {
		t.Errorf("expected status=failed after 1 failure, got %q", got)
	}
	if got := rowAttempts(t, tdb, id); got != 1 {
		t.Errorf("expected attempts=1, got %d", got)
	}
}

func TestNeo4jSyncWorker_ProcessBatch_DeadLetterAfterMaxAttempts(t *testing.T) {
	if testing.Short() {
		t.Skip("skip: integration test requires Docker")
	}
	tdb := testutil.NewTestDB(t)

	neo4j := &stubNeo4j{failOn: map[string]error{"asset-ccc": fmt.Errorf("permanent failure")}}
	w := &Neo4jSyncWorker{db: tdb.DB, neo4jRepo: neo4j, stop: make(chan struct{}), interval: time.Second}

	// Insert with attempts=4 (one more failure = 5 = dead letter).
	id := insertPendingRow(t, tdb, "asset-ccc", 4, "failed")

	w.processBatch(context.Background())

	if got := rowStatus(t, tdb, id); got != "dead_letter" {
		t.Errorf("expected status=dead_letter at attempt 5, got %q", got)
	}
}

func TestNeo4jSyncWorker_RecoverStaleRows(t *testing.T) {
	if testing.Short() {
		t.Skip("skip: integration test requires Docker")
	}
	tdb := testutil.NewTestDB(t)

	neo4j := &stubNeo4j{}
	w := &Neo4jSyncWorker{db: tdb.DB, neo4jRepo: neo4j, stop: make(chan struct{}), interval: time.Second}

	// Insert a row stuck in 'processing' (simulating a crashed worker).
	id := insertPendingRow(t, tdb, "asset-ddd", 1, "processing")
	// Back-date updated_at so it looks stale (>10 minutes old).
	_, err := tdb.DB.ExecContext(context.Background(),
		"UPDATE neo4j_sync_queue SET updated_at = NOW() - INTERVAL '15 minutes' WHERE id=$1", id)
	if err != nil {
		t.Fatalf("back-date row: %v", err)
	}

	w.recoverStaleRows(context.Background())

	if got := rowStatus(t, tdb, id); got != "pending" {
		t.Errorf("expected stale processing row reset to pending, got %q", got)
	}
}

func TestNeo4jSyncWorker_EmptyQueue_NoOp(t *testing.T) {
	if testing.Short() {
		t.Skip("skip: integration test requires Docker")
	}
	tdb := testutil.NewTestDB(t)

	neo4j := &stubNeo4j{}
	w := &Neo4jSyncWorker{db: tdb.DB, neo4jRepo: neo4j, stop: make(chan struct{}), interval: time.Second}

	// Should complete without error on empty queue.
	w.processBatch(context.Background())

	if len(neo4j.calls) != 0 {
		t.Errorf("expected no neo4j calls on empty queue, got %d", len(neo4j.calls))
	}
}
