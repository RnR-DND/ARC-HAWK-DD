// Package connectors provides HTTP client adapters for the hawk agent to
// communicate with the ARC-HAWK backend.
package connectors

import (
	"context"
	"encoding/json"
)

// SyncBatch is a batch of scan records posted to the backend sync endpoint.
type SyncBatch struct {
	Results []SyncRecord `json:"results"`
}

// SyncRecord is a single scan result entry within a SyncBatch.
type SyncRecord struct {
	ScanJobID string          `json:"scan_job_id"`
	BatchSeq  int             `json:"batch_seq"`
	Payload   json.RawMessage `json:"payload"`
}

// ScanRequest is the payload to trigger a new scan job.
type ScanRequest struct {
	AgentID  string `json:"agent_id"`
	ScanType string `json:"scan_type"`
}

// ScanResponse is the response to a scan trigger request.
type ScanResponse struct {
	ScanJobID string `json:"scan_job_id"`
	Status    string `json:"status"`
}

// ScanStatusResponse is the response to a scan status poll.
type ScanStatusResponse struct {
	Status string `json:"status"`
}

// ScanResult is a single result entry from a completed scan.
type ScanResult struct {
	BatchSeq int             `json:"batch_seq"`
	Payload  json.RawMessage `json:"payload"`
}

// Client wraps HTTP calls to the ARC-HAWK backend scanner API.
// This is a stub — replace with a fully wired HTTP client for production.
type Client struct {
	serverURL string
}

// NewClient creates a new connector client targeting serverURL.
func NewClient(serverURL string) *Client {
	return &Client{serverURL: serverURL}
}

// HealthCheck returns true if the backend is reachable and healthy.
func (c *Client) HealthCheck(ctx context.Context) bool {
	return false // stub — wire real HTTP GET /health
}

// PostSyncBatch posts a SyncBatch to the backend.
// Returns the HTTP status code and any transport-level error.
func (c *Client) PostSyncBatch(ctx context.Context, batch SyncBatch) (int, error) {
	return 0, nil // stub — wire real HTTP POST /api/v1/agent/sync
}

// TriggerScan requests the backend to start a scan.
func (c *Client) TriggerScan(ctx context.Context, req ScanRequest) (*ScanResponse, error) {
	return nil, nil // stub
}

// PollScanStatus polls the status of a running scan.
func (c *Client) PollScanStatus(ctx context.Context, scanJobID string) (*ScanStatusResponse, error) {
	return nil, nil // stub
}

// FetchResults retrieves completed scan results from the backend.
func (c *Client) FetchResults(ctx context.Context, scanJobID string) ([]ScanResult, error) {
	return nil, nil // stub
}

// StreamResult sends a single scan result directly to the backend (online mode).
func (c *Client) StreamResult(ctx context.Context, agentID string, result ScanResult) (int, error) {
	return 0, nil // stub
}
