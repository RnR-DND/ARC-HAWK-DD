package orchestrator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/arc-platform/go-scanner/internal/classifier"
)

// Ingestion chunking parameters (match the Python scanner's behavior).
const (
	ingestChunkSize     = 2000
	ingestChunkPause    = 2 * time.Second
	ingestProgressEvery = 500
)

// IngestFindings POSTs findings to the backend in chunks of 2000.
// Progress events are sent every ingestProgressEvery rows.
//
// tenantID is forwarded as X-Tenant-ID on every ingest call. An empty
// tenantID produces a best-effort call (the backend will reject it unless
// running in dev mode).
func IngestFindings(scanID, scanName, tenantID, backendURL string, findings []classifier.ClassifiedFinding) error {
	total := len(findings)
	client := &http.Client{Timeout: 60 * time.Second}

	if total == 0 {
		sendProgressEvent(client, tenantID, backendURL, scanID, 0, 100.0)
		return nil
	}

	sent := 0
	for i := 0; i < total; i += ingestChunkSize {
		end := min(i+ingestChunkSize, total)
		batch := findings[i:end]

		payload := buildPayload(scanID, scanName, batch)
		data, err := json.Marshal(payload)
		if err != nil {
			log.Printf("WARN: ingest marshal failed: %v", err)
			continue
		}

		httpReq, err := http.NewRequest("POST", backendURL+"/api/v1/scans/ingest-verified", bytes.NewReader(data))
		if err != nil {
			log.Printf("WARN: ingest request build failed: %v", err)
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if tenantID != "" {
			httpReq.Header.Set("X-Tenant-ID", tenantID)
		}
		resp, err := client.Do(httpReq)
		if err != nil {
			log.Printf("WARN: ingest chunk %d-%d failed: %v", i, end, err)
			continue
		}
		resp.Body.Close()
		sent = end

		if sent%ingestProgressEvery == 0 || sent == total {
			pct := float64(sent) / float64(total) * 100
			sendProgressEvent(client, tenantID, backendURL, scanID, sent, pct)
		}

		if end < total {
			time.Sleep(ingestChunkPause)
		}
	}

	log.Printf("Ingested %d/%d findings for scan %s", sent, total, scanID)
	return nil
}

// buildPayload constructs the VerifiedScanInput envelope expected by
// apps/backend/modules/scanning/service/ingest_sdk_verified.go.
func buildPayload(scanID, scanName string, findings []classifier.ClassifiedFinding) map[string]any {
	items := make([]map[string]any, 0, len(findings))
	for _, f := range findings {
		items = append(items, map[string]any{
			"pii_type":          f.PIIType,
			"value_hash":        f.ValueHash,
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
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
