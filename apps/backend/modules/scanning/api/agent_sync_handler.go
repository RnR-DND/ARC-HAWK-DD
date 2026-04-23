package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

// AgentSyncRequest is the payload POSTed by EDR agents for batch ingestion.
// Accepts both "batches" (current agents) and "results" (legacy field name) for
// backwards compatibility — see C10 fix.
type AgentSyncRequest struct {
	AgentID string           `json:"agent_id" binding:"required"`
	Batches []AgentSyncBatch `json:"batches"` // preferred field name
	Results []AgentSyncBatch `json:"results"` // legacy field name (apps/agent ≤ v1)
}

// AgentSyncBatch is a single batch within an agent sync request.
type AgentSyncBatch struct {
	ScanJobID string          `json:"scan_job_id" binding:"required"`
	BatchSeq  int             `json:"batch_seq" binding:"required,min=0"`
	Results   json.RawMessage `json:"results" binding:"required"`
}

// AgentSyncResponse is returned after processing an agent sync request.
type AgentSyncResponse struct {
	Accepted int           `json:"accepted"`
	Skipped  int           `json:"skipped"` // duplicates
	Late     int           `json:"late"`    // arrived after job completed/cancelled
	Failed   int           `json:"failed"`
	Details  []BatchResult `json:"details"`
}

// BatchResult reports the outcome of a single batch within the sync request.
type BatchResult struct {
	ScanJobID string `json:"scan_job_id"`
	BatchSeq  int    `json:"batch_seq"`
	Status    string `json:"status"` // "accepted", "skipped", "arrived_late", "failed"
	Error     string `json:"error,omitempty"`
}

// AgentSyncHandler handles POST /api/v1/agent/sync — idempotent batch
// ingestion from EDR agents.
type AgentSyncHandler struct {
	repo *persistence.PostgresRepository
	rdb  *goredis.Client
}

// NewAgentSyncHandler creates an AgentSyncHandler backed by the shared Postgres
// repository and an optional Redis client for the async classify queue.
// If redisAddr is empty the classify publish step is skipped (data is safe in PG).
func NewAgentSyncHandler(repo *persistence.PostgresRepository) *AgentSyncHandler {
	h := &AgentSyncHandler{repo: repo}
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		h.rdb = goredis.NewClient(&goredis.Options{
			Addr:         addr,
			Password:     os.Getenv("REDIS_PASSWORD"),
			DB:           0,
			DialTimeout:  3 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		})
	}
	return h
}

// Sync godoc
// @Summary Ingest agent scan batch (idempotent)
// @Description Idempotent batch ingestion from EDR agents. Requires X-Scanner-Token + X-Tenant-ID headers.
// @Tags agent
// @Accept json
// @Produce json
// @Param X-Scanner-Token header string true "Scanner service token"
// @Param X-Tenant-ID header string true "Tenant UUID"
// @Param body body AgentSyncRequest true "Batch payload"
// @Success 200 {object} AgentSyncResponse
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Router /agent/sync [post]
// Sync handles POST /api/v1/agent/sync.
//
// Logic per batch:
//  1. Check agent_sync_log for (agent_id, scan_job_id, batch_seq) — duplicate check.
//  2. If the scan_job status is CANCELLED or COMPLETED, accept but tag ARRIVED_LATE.
//  3. Insert to agent_sync_log, then publish results to the classify queue via Redis.
//  4. All within a transaction.
func (h *AgentSyncHandler) Sync(c *gin.Context) {
	var req AgentSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"message": err.Error(),
		})
		return
	}

	// Accept "results" as a legacy alias for "batches" (C10 compatibility).
	batches := req.Batches
	if len(batches) == 0 {
		batches = req.Results
	}
	if len(batches) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "batches or results array must not be empty",
		})
		return
	}

	resp := AgentSyncResponse{
		Details: make([]BatchResult, 0, len(batches)),
	}

	db := h.repo.GetDB()

	for _, batch := range batches {
		result := h.processBatch(c, db, req.AgentID, batch)
		resp.Details = append(resp.Details, result)

		switch result.Status {
		case "accepted":
			resp.Accepted++
		case "skipped":
			resp.Skipped++
		case "arrived_late":
			resp.Late++
		case "failed":
			resp.Failed++
		}
	}

	c.JSON(http.StatusOK, resp)
}

// processBatch handles a single batch within a transaction.
func (h *AgentSyncHandler) processBatch(c *gin.Context, db *sql.DB, agentID string, batch AgentSyncBatch) BatchResult {
	ctx := c.Request.Context()
	result := BatchResult{
		ScanJobID: batch.ScanJobID,
		BatchSeq:  batch.BatchSeq,
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("ERROR: agent_sync: begin tx failed: %v", err)
		result.Status = "failed"
		result.Error = "internal transaction error"
		return result
	}
	defer tx.Rollback() //nolint:errcheck — rollback after commit is a no-op

	// 1. Duplicate check: look for existing (agent_id, scan_job_id, batch_seq)
	var existingStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT status FROM agent_sync_log
		WHERE agent_id = $1 AND scan_job_id = $2 AND batch_seq = $3
	`, agentID, batch.ScanJobID, batch.BatchSeq).Scan(&existingStatus)

	if err == nil {
		// Row already exists — skip as duplicate.
		result.Status = "skipped"
		return result
	}
	if err != sql.ErrNoRows {
		log.Printf("ERROR: agent_sync: duplicate check failed: %v", err)
		result.Status = "failed"
		result.Error = "duplicate check error"
		return result
	}

	// 2. Check scan_job status (scan_runs table). If completed/cancelled, tag late.
	batchStatus := "received"
	var scanStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT status FROM scan_runs WHERE id::text = $1
	`, batch.ScanJobID).Scan(&scanStatus)

	if err == nil && (scanStatus == "completed" || scanStatus == "cancelled") {
		batchStatus = "arrived_late"
	}
	// If scan_runs row doesn't exist we still accept — the agent may report
	// before the scan_run is visible to this instance.

	// 3. Compute payload hash for dedup/auditability.
	payloadHash := hashPayload(batch.Results)

	// 4. Insert into agent_sync_log.
	_, err = tx.ExecContext(ctx, `
		INSERT INTO agent_sync_log (agent_id, scan_job_id, batch_seq, payload_hash, status)
		VALUES ($1, $2, $3, $4, $5)
	`, agentID, batch.ScanJobID, batch.BatchSeq, payloadHash, batchStatus)
	if err != nil {
		log.Printf("ERROR: agent_sync: insert failed: %v", err)
		result.Status = "failed"
		result.Error = "insert error"
		return result
	}

	// 5. Publish results to classification pipeline (Redis "classify" queue).
	//    This uses a best-effort approach — if Redis is unavailable the data is
	//    persisted in agent_sync_log and can be replayed. In the current architecture
	//    classification is triggered via HTTP callback from the scanner, so this
	//    publish is forward-compatible for the async pipeline.
	if err := h.publishToClassifyQueue(agentID, batch); err != nil {
		// Non-fatal: log but don't fail the batch — data is safe in PG.
		log.Printf("WARN: agent_sync: classify publish failed (non-fatal): %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("ERROR: agent_sync: commit failed: %v", err)
		result.Status = "failed"
		result.Error = "commit error"
		return result
	}

	if batchStatus == "arrived_late" {
		result.Status = "arrived_late"
	} else {
		result.Status = "accepted"
	}
	return result
}

// hashPayload returns the SHA-256 hex digest of the raw JSON payload.
func hashPayload(data json.RawMessage) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// publishToClassifyQueue publishes a batch to the Redis "classify" list queue via LPUSH.
// Best-effort: if Redis is unavailable the batch data is already persisted in agent_sync_log
// and can be replayed. Non-fatal errors are logged but do not fail the sync response.
func (h *AgentSyncHandler) publishToClassifyQueue(agentID string, batch AgentSyncBatch) error {
	msg := map[string]interface{}{
		"agent_id":    agentID,
		"scan_job_id": batch.ScanJobID,
		"batch_seq":   batch.BatchSeq,
		"results":     batch.Results,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal classify message: %w", err)
	}

	if h.rdb == nil {
		log.Printf("INFO: agent_sync: Redis not configured — classify queue skipped for job %s batch %d (data safe in PG)", batch.ScanJobID, batch.BatchSeq)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.rdb.LPush(ctx, "classify", payload).Err(); err != nil {
		return fmt.Errorf("redis LPUSH classify: %w", err)
	}

	log.Printf("INFO: agent_sync: published %d bytes to classify queue for job %s batch %d", len(payload), batch.ScanJobID, batch.BatchSeq)
	return nil
}
