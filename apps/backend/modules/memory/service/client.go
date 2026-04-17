package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// ErrDisabled is returned when SUPERMEMORY_ENABLED is not "true" or the API key is missing.
var ErrDisabled = errors.New("supermemory: disabled (SUPERMEMORY_ENABLED!=true or SUPERMEMORY_API_KEY empty)")

// Client talks to the supermemory.ai REST API.
// Free tier: 1M tokens/mo + 10K queries/mo. Ingestion is async (POST returns queued).
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
	enabled bool
}

// NewClientFromEnv reads SUPERMEMORY_API_URL, SUPERMEMORY_API_KEY, SUPERMEMORY_ENABLED.
// Returns a disabled client (enabled=false) rather than nil if config is missing, so
// callers can no-op safely in dev environments without the key.
func NewClientFromEnv() *Client {
	url := os.Getenv("SUPERMEMORY_API_URL")
	if url == "" {
		url = "https://api.supermemory.ai"
	}
	key := os.Getenv("SUPERMEMORY_API_KEY")
	enabled := os.Getenv("SUPERMEMORY_ENABLED") == "true" && key != ""

	return &Client{
		baseURL: url,
		apiKey:  key,
		http:    &http.Client{Timeout: 15 * time.Second},
		enabled: enabled,
	}
}

// Enabled reports whether the client will actually talk to the API.
func (c *Client) Enabled() bool { return c.enabled }

// Document is the payload for POST /v3/documents.
type Document struct {
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Title    string                 `json:"title,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
}

// AddDocumentResponse is returned by POST /v3/documents.
type AddDocumentResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// AddDocument queues a document for ingestion.
// Ingestion is async; the returned ID is searchable once status becomes "processed"
// (typically within a few seconds for short text, longer for multi-modal).
func (c *Client) AddDocument(ctx context.Context, doc Document) (*AddDocumentResponse, error) {
	if !c.enabled {
		return nil, ErrDisabled
	}
	var out AddDocumentResponse
	if err := c.do(ctx, http.MethodPost, "/v3/documents", doc, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SearchQuery is the payload for POST /v3/search.
type SearchQuery struct {
	Q         string                 `json:"q"`
	Limit     int                    `json:"limit,omitempty"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	IncludeDocs bool                 `json:"include_documents,omitempty"`
}

// SearchResult is one row in a search response.
type SearchResult struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

// SearchResponse wraps /v3/search output.
type SearchResponse struct {
	Results  []SearchResult `json:"results"`
	Total    int            `json:"total"`
	TimingMs int            `json:"timing"`
}

// Search performs hybrid (memory + RAG) search.
func (c *Client) Search(ctx context.Context, q SearchQuery) (*SearchResponse, error) {
	if !c.enabled {
		return nil, ErrDisabled
	}
	if q.Limit == 0 {
		q.Limit = 10
	}
	var out SearchResponse
	if err := c.do(ctx, http.MethodPost, "/v3/search", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) do(ctx context.Context, method, path string, body, out interface{}) error {
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		rdr = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "arc-hawk-dd/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("do: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("supermemory %s %s: %d: %s", method, path, resp.StatusCode, string(raw))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
	}
	return nil
}
