// Package presidio is a thin HTTP client for the Presidio Analyzer service.
//
// Presidio complements the regex engine by running spaCy NER + built-in
// recognizers to catch PII that has no fixed format — names, locations,
// unstructured dates-of-birth, and generic organizations. For Indian PII with
// deterministic patterns (PAN, Aadhaar, IFSC, UPI, etc.) regex remains the
// primary detector and Presidio is additive.
package presidio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Entity is one detection returned by Presidio /analyze.
type Entity struct {
	Type  string  `json:"entity_type"`
	Start int     `json:"start"`
	End   int     `json:"end"`
	Score float64 `json:"score"`
}

// Client talks to a Presidio Analyzer HTTP endpoint.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a client pointed at the given Presidio URL.
// A zero-value baseURL disables all calls (Analyze returns nil without erroring).
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Enabled reports whether the client has a configured endpoint.
func (c *Client) Enabled() bool { return c != nil && c.baseURL != "" }

type analyzeRequest struct {
	Text             string            `json:"text"`
	Language         string            `json:"language"`
	Entities         []string          `json:"entities,omitempty"`
	ScoreThreshold   float64           `json:"score_threshold,omitempty"`
	Context          []string          `json:"context,omitempty"`
	AdHocRecognizers []AdHocRecognizer `json:"ad_hoc_recognizers,omitempty"`
}

// AnalyzeOptions collects the knobs an individual /analyze call can set.
type AnalyzeOptions struct {
	// Entities limits which supported_entity names Presidio may return.
	// Empty means all entities it knows about are eligible.
	Entities []string
	// ContextWords are passed to the global LemmaContextAwareEnhancer — they
	// apply to every recognizer, not just the ad-hoc ones.
	ContextWords []string
	// AdHocRecognizers teach Presidio about additional PatternRecognizers
	// for the duration of this single request. Used to ship the Indian
	// recognizer pack and user-defined custom regex patterns without
	// reconfiguring the Presidio service.
	AdHocRecognizers []AdHocRecognizer
}

// Analyze runs a text through Presidio and returns detected entities.
//
// Returns a nil slice when the client is disabled or the remote fails —
// Presidio is best-effort, never fatal.
func (c *Client) Analyze(ctx context.Context, text string, opts AnalyzeOptions) []Entity {
	if !c.Enabled() || text == "" {
		return nil
	}
	payload := analyzeRequest{
		Text:             text,
		Language:         "en",
		Entities:         opts.Entities,
		ScoreThreshold:   0.5,
		Context:          opts.ContextWords,
		AdHocRecognizers: opts.AdHocRecognizers,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/analyze", bytes.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	var out []Entity
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

// Health pings /health and returns nil on success.
func (c *Client) Health(ctx context.Context) error {
	if !c.Enabled() {
		return fmt.Errorf("presidio client not configured")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("presidio health %d", resp.StatusCode)
	}
	return nil
}
