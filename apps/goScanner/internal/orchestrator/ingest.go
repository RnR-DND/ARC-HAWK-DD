package orchestrator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/arc-platform/go-scanner/internal/classifier"
)

// Ingestion chunking parameters (match the Python scanner's behavior).
const (
	ingestChunkSize     = 2000
	ingestChunkPause    = 2 * time.Second
	ingestProgressEvery = 500
	ingestMaxAttempts   = 3
	ingestBackoffBase   = 2 * time.Second
)

// scannerServiceToken is read at package init and reused across requests.
// Empty string means no token; backend should then reject in release mode.
var scannerServiceToken = os.Getenv("SCANNER_SERVICE_TOKEN")

// ingestClient is reused across scans to keep the connection pool warm and
// amortize TLS / keep-alive setup against the backend.
var ingestClient = &http.Client{Timeout: 60 * time.Second}

// IngestFindings POSTs findings to the backend in chunks of 2000.
// Progress events are sent every ingestProgressEvery rows. Per-chunk failures
// are logged and skipped but counted; the caller receives a summary error when
// any chunk failed so a scan never reports "ingested" on silent data loss.
//
// tenantID is forwarded as X-Tenant-ID on every ingest call. An empty
// tenantID produces a best-effort call (the backend will reject it unless
// running in dev mode).
func IngestFindings(scanID, scanName, tenantID, backendURL string, findings []classifier.ClassifiedFinding) error {
	total := len(findings)

	if total == 0 {
		sendProgressEvent(ingestClient, tenantID, backendURL, scanID, 0, 100.0)
		sendComplete(tenantID, backendURL, scanID, "completed", "scan produced zero findings")
		return nil
	}

	var (
		sent         int
		failedChunks int
		totalChunks  = (total + ingestChunkSize - 1) / ingestChunkSize
	)
	for i := 0; i < total; i += ingestChunkSize {
		end := min(i+ingestChunkSize, total)
		batch := findings[i:end]

		payload := buildPayload(scanID, scanName, batch)
		data, err := json.Marshal(payload)
		if err != nil {
			log.Printf("WARN: ingest marshal failed: %v", err)
			failedChunks++
			continue
		}

		if err := sendIngestChunkWithRetry(tenantID, backendURL, data, i, end); err != nil {
			failedChunks++
			continue
		}
		sent = end

		if sent%ingestProgressEvery == 0 || sent == total {
			pct := float64(sent) / float64(total) * 100
			sendProgressEvent(ingestClient, tenantID, backendURL, scanID, sent, pct)
		}

		if end < total {
			time.Sleep(ingestChunkPause)
		}
	}

	log.Printf("Ingested %d/%d findings for scan %s (%d/%d chunks failed)", sent, total, scanID, failedChunks, totalChunks)
	if failedChunks == totalChunks {
		// All chunks failed — hard failure.
		sendComplete(tenantID, backendURL, scanID, "failed",
			fmt.Sprintf("all %d ingest chunks failed after %d attempts each", totalChunks, ingestMaxAttempts))
		return fmt.Errorf("ingest failed: all %d chunks failed", totalChunks)
	}
	if failedChunks > 0 {
		// Partial success — mark scan partial so operators know not to trust
		// the finding count as complete.
		sendComplete(tenantID, backendURL, scanID, "partial",
			fmt.Sprintf("%d of %d ingest chunks failed after %d retry attempts each (sent %d of %d findings)",
				failedChunks, totalChunks, ingestMaxAttempts, sent, total))
		return fmt.Errorf("ingest completed with %d failed chunks (sent %d/%d findings)", failedChunks, sent, total)
	}
	sendComplete(tenantID, backendURL, scanID, "completed", "")
	return nil
}

// sendIngestChunkWithRetry posts one chunk, retrying transient failures with
// exponential backoff (2s, 4s, 8s). Returns nil if the chunk eventually
// succeeded, or the last error after ingestMaxAttempts.
func sendIngestChunkWithRetry(tenantID, backendURL string, data []byte, startIdx, endIdx int) error {
	var lastErr error
	for attempt := 1; attempt <= ingestMaxAttempts; attempt++ {
		req, err := http.NewRequest("POST", backendURL+"/api/v1/scans/ingest-verified", bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if tenantID != "" {
			req.Header.Set("X-Tenant-ID", tenantID)
		}
		if scannerServiceToken != "" {
			req.Header.Set("X-Scanner-Token", scannerServiceToken)
		}
		resp, err := ingestClient.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("WARN: ingest chunk %d-%d attempt %d/%d transport error: %v",
				startIdx, endIdx, attempt, ingestMaxAttempts, err)
		} else {
			_ = resp.Body.Close()
			// 2xx counts as success. 4xx is unlikely to resolve on retry
			// (bad payload, bad auth), so fail fast instead of looping.
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				return fmt.Errorf("ingest chunk %d-%d rejected by backend with status %d", startIdx, endIdx, resp.StatusCode)
			}
			lastErr = fmt.Errorf("ingest chunk %d-%d got status %d", startIdx, endIdx, resp.StatusCode)
			log.Printf("WARN: %v (attempt %d/%d)", lastErr, attempt, ingestMaxAttempts)
		}
		if attempt < ingestMaxAttempts {
			time.Sleep(time.Duration(1<<uint(attempt-1)) * ingestBackoffBase)
		}
	}
	return lastErr
}

// sendComplete signals the backend that this scan is done. Without this call
// the scan stays in "running" until the backend's timeout sweeper flips it
// to "timeout" (10 min default), which misleads the UI into showing
// successful scans as timeouts.
func sendComplete(tenantID, backendURL, scanID, status, message string) {
	if backendURL == "" {
		return
	}
	body, _ := json.Marshal(map[string]any{
		"status":  status,
		"message": message,
	})
	req, err := http.NewRequest("POST", backendURL+"/api/v1/scans/"+scanID+"/complete", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}
	if scannerServiceToken != "" {
		req.Header.Set("X-Scanner-Token", scannerServiceToken)
	}
	resp, err := ingestClient.Do(req)
	if err != nil {
		log.Printf("WARN: scan complete signal failed for %s: %v", scanID, err)
		return
	}
	resp.Body.Close()
}

// buildPayload constructs the VerifiedScanInput envelope expected by
// apps/backend/modules/scanning/service/ingest_sdk_verified.go.
func buildPayload(scanID, scanName string, findings []classifier.ClassifiedFinding) map[string]any {
	items := make([]map[string]any, 0, len(findings))
	for _, f := range findings {
		items = append(items, map[string]any{
			"pii_type":          f.PIIType,
			"value_hash":        f.ValueHash,
			"matched_value":     f.MatchedValue,
			"pattern_name":      f.PatternName,
			"ml_confidence":     float64(f.Score) / 100.0,
			"ml_entity_type":    f.PIIType,
			"validators_passed": []string{"pattern_match"},
			"validation_method": "regex",
			"context_excerpt":   f.ContextExcerpt,
			"context_keywords":  []string{},
			"detected_at":       time.Now().UTC().Format(time.RFC3339),
			"scanner_version":   "2.0-go",
			"source": map[string]any{
				"path":        f.SourcePath,
				"column":      f.FieldName,
				"table":       f.Table,
				"data_source": f.DataSource,
				"host":        f.Host,
			},
		})
	}
	return map[string]any{
		"scan_id":   scanID,
		"scan_name": scanName,
		"findings":  items,
	}
}

// sendProgressEvent fires a best-effort progress update to the backend. The
// client argument is accepted so tests can inject one; production calls pass
// the package-level ingestClient.
func sendProgressEvent(client *http.Client, tenantID, backendURL, scanID string, found int, pct float64) {
	if backendURL == "" {
		return
	}
	evt := map[string]any{
		"scan_id":        scanID,
		"findings_found": found,
		"current_source": "scanner",
		"percent_done":   pct,
	}
	data, _ := json.Marshal(evt)
	url := fmt.Sprintf("%s/api/v1/scans/%s/progress-event", backendURL, scanID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}
	if scannerServiceToken != "" {
		req.Header.Set("X-Scanner-Token", scannerServiceToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
