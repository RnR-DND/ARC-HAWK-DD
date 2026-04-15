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

// IngestFindings POSTs findings to the backend in batches of 100.
// Every 50 findings it also sends a progress event.
func IngestFindings(scanID, backendURL string, findings []classifier.ClassifiedFinding) error {
	const batchSize = 100
	const progressEvery = 50

	total := len(findings)
	if total == 0 {
		return nil
	}

	client := &http.Client{Timeout: 30 * time.Second}

	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}
		batch := findings[i:end]

		payload := buildPayload(scanID, batch)
		data, _ := json.Marshal(payload)

		resp, err := client.Post(
			backendURL+"/api/v1/scans/ingest-verified",
			"application/json",
			bytes.NewReader(data),
		)
		if err != nil {
			log.Printf("WARN: ingest batch failed: %v", err)
			continue
		}
		resp.Body.Close()

		if end%progressEvery == 0 || end == total {
			pct := float64(end) / float64(total) * 100
			sendProgressEvent(client, backendURL, scanID, end, pct)
		}
	}
	return nil
}

func buildPayload(scanID string, findings []classifier.ClassifiedFinding) map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(findings))
	for _, f := range findings {
		items = append(items, map[string]interface{}{
			"pii_type":         f.PIIType,
			"value_hash":       f.ValueHash,
			"pattern_name":     f.PatternName,
			"confidence_score": f.Score,
			"detector_type":    f.DetectorType,
			"source_path":      f.SourcePath,
			"context_excerpt":  f.ContextExcerpt,
		})
	}
	return map[string]interface{}{
		"scan_id":  scanID,
		"findings": items,
	}
}

func sendProgressEvent(client *http.Client, backendURL, scanID string, found int, pct float64) {
	evt := map[string]interface{}{
		"scan_id":        scanID,
		"findings_found": found,
		"current_source": "scanner",
		"percent_done":   pct,
	}
	data, _ := json.Marshal(evt)
	url := fmt.Sprintf("%s/api/v1/scans/%s/progress-event", backendURL, scanID)
	resp, err := client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return
	}
	resp.Body.Close()
}
