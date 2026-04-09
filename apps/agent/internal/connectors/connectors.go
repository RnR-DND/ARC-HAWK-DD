package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/arc/hawk-agent/internal/auth"
	"github.com/arc/hawk-agent/internal/config"
	"go.uber.org/zap"
)

// ScannerConnector wraps HTTP calls to the Hawk Scanner API.
type ScannerConnector struct {
	cfg        *config.Config
	authClient *auth.Client
	httpClient *http.Client
	logger     *zap.Logger
}

// SyncRequest represents a batch of scan results to sync.
type SyncRequest struct {
	AgentID string       `json:"agent_id"`
	Results []SyncResult `json:"results"`
}

// SyncResult is a single scan result entry.
type SyncResult struct {
	ScanJobID string          `json:"scan_job_id"`
	BatchSeq  int             `json:"batch_seq"`
	Payload   json.RawMessage `json:"payload"`
}

// ScanTriggerRequest asks the backend to trigger a scan.
type ScanTriggerRequest struct {
	AgentID    string   `json:"agent_id"`
	ScanTypes  []string `json:"scan_types"`
	ScheduleID string   `json:"schedule_id,omitempty"`
}

// ScanTriggerResponse from the backend.
type ScanTriggerResponse struct {
	ScanJobID string `json:"scan_job_id"`
	Status    string `json:"status"`
}

// NewScannerConnector creates a connector to the Hawk Scanner API.
func NewScannerConnector(cfg *config.Config, authClient *auth.Client, logger *zap.Logger) *ScannerConnector {
	return &ScannerConnector{
		cfg:        cfg,
		authClient: authClient,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// HealthCheck pings the scanner API health endpoint.
// Returns nil if healthy, error otherwise.
func (sc *ScannerConnector) HealthCheck(ctx context.Context) error {
	url := sc.cfg.ServerURL + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build health request: %w", err)
	}

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned %d", resp.StatusCode)
	}
	return nil
}

// SyncResults posts a batch of scan results to the scanner API.
// Returns: HTTP status code and error.
func (sc *ScannerConnector) SyncResults(ctx context.Context, req *SyncRequest) (int, error) {
	token, err := sc.authClient.Token()
	if err != nil {
		return 0, fmt.Errorf("auth token: %w", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return 0, fmt.Errorf("marshal sync request: %w", err)
	}

	url := sc.cfg.ServerURL + "/api/v1/agent/sync"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("build sync request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := sc.httpClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("sync request failed: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, nil
}

// TriggerScan asks the backend to start a scan for this agent.
func (sc *ScannerConnector) TriggerScan(ctx context.Context, scanTypes []string) (*ScanTriggerResponse, error) {
	token, err := sc.authClient.Token()
	if err != nil {
		return nil, fmt.Errorf("auth token: %w", err)
	}

	payload := ScanTriggerRequest{
		AgentID:   sc.cfg.AgentID,
		ScanTypes: scanTypes,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal trigger request: %w", err)
	}

	url := sc.cfg.ServerURL + "/api/v1/agent/scan"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build trigger request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := sc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("trigger scan request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("trigger scan returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result ScanTriggerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode trigger response: %w", err)
	}
	return &result, nil
}

// StreamResult sends a single scan result directly (online mode).
func (sc *ScannerConnector) StreamResult(ctx context.Context, scanJobID string, batchSeq int, payload []byte) (int, error) {
	syncReq := &SyncRequest{
		AgentID: sc.cfg.AgentID,
		Results: []SyncResult{
			{
				ScanJobID: scanJobID,
				BatchSeq:  batchSeq,
				Payload:   json.RawMessage(payload),
			},
		},
	}
	return sc.SyncResults(ctx, syncReq)
}
